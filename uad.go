/*
author: deadc0de6 (https://github.com/deadc0de6)
Copyright (c) 2020, deadc0de6
*/

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
)

const (
	fCreationRights = 0666
	version         = "0.7.2"
	// web endpoint for upload
	webUploadPath = "upload/"
	webFilePath   = "files/"
	title         = "uad"
)

var (
	dfltMaxUploadSize = "1G"
	units             = []string{"B", "K", "M", "G", "T", "P"}
	//go:embed "page.html"
	page string
)

// Param global parameters
type Param struct {
	Host            string
	Port            int
	Paths           []string
	MaxUploadSize   int64
	EnableUploads   bool
	EnableDownloads bool
	HiddenFiles     bool
}

// PathParam single exposed path parameter
type PathParam struct {
	WebPath         string
	FSPath          string
	EnableUploads   bool
	EnableDownloads bool
	MaxUploadSize   int64
	HiddenFiles     bool
	Others          []string
}

// TmplData template parameters
type TmplData struct {
	Title           string
	Version         string
	BaseName        string
	UploadPath      string
	EnableUploads   bool
	EnableDownloads bool
	Others          []string
}

// HTMLFile an uploaded file
type HTMLFile struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	Modified string `json:"modified"`
	// WPath web path
	WPath string `json:"wpath"`
	// RPath relative path on filesystem
	RPath string `json:"rpath"`
}

// is path a valid directory
func isValidDir(path string) error {
	dir, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path \"%s\" is not valid: %s", path, err.Error())
	}

	if !dir.IsDir() {
		return fmt.Errorf("%s is not a valid directory", path)
	}
	return nil
}

// return size in human readable format
func sizeToHuman(bytes int64) string {
	size := bytes
	rest := int64(0)
	for _, unit := range units {
		if size < 1024 {
			return fmt.Sprintf("%d.%d%s", size, rest, unit)
		}
		rest = size % 1024
		size = size / 1024
	}
	return "??"
}

// return human size in bytes
func humanToSize(size string) (int64, error) {
	unit := string(size[len(size)-1])
	sz := string(size[:len(size)-1])

	n, err := strconv.ParseInt(sz, 10, 64)
	if err != nil {
		return int64(0), err
	}

	var mult int64
	switch string(unit) {
	case "K":
		mult = int64(1 << 10)
	case "M":
		mult = int64(1 << 20)
	case "G":
		mult = int64(1 << 30)
	case "T":
		mult = int64(1 << 40)
	case "P":
		mult = int64(1 << 50)
	default:
		return int64(0), errors.New("bad size unit")
	}
	return int64(n * mult), nil
}

// walk a directory and return HTMLFiles list
func walker(base string, webBase string, hfiles *[]*HTMLFile, hiddenFiles bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error(err)
			return nil
		}

		log.Debugf("found \"%s\"", path)
		log.Debugf("  dir:%v", info.IsDir())
		//log.Debugf("  %#v", info)
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		rpath, err := filepath.Rel(base, path)
		if err != nil {
			log.Error(err)
			return nil
		}
		log.Debugf("  base:\"%s\"", base)
		log.Debugf("  fs path:\"%s\" -> \"%s\"", path, rpath)

		if !hiddenFiles {
			if strings.HasPrefix(rpath, ".") {
				log.Debugf("skipping hidden file: %s", rpath)
				return nil
			}
		}

		wpath := filepath.Join(webBase, rpath)
		log.Debugf("  web path:\"%s\"", wpath)
		hfile := &HTMLFile{
			Name:     name,
			Size:     sizeToHuman(info.Size()),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			WPath:    wpath,
			RPath:    rpath,
		}
		log.Debugf("  pushing file: %#v", hfile)

		*hfiles = append(*hfiles, hfile)
		return nil
	}
}

// get list of files in upload dir
func getFiles(path string, webPath string, hidden bool) ([]*HTMLFile, error) {
	var hfiles []*HTMLFile
	log.Debugf("walking \"%s\"...", path)
	err := filepath.Walk(path, walker(path, webPath, &hfiles, hidden))
	if err != nil {
		return nil, err
	}
	return hfiles, nil
}

// save file locally from upload form
func saveFile(file io.Reader, name string, dstdir string) error {
	mkdirp(dstdir)
	dst := filepath.Join(dstdir, name)
	fmt.Printf("saving file to \"%s\"\n", dst)
	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, fCreationRights)
	if err != nil {
		return err
	}
	io.Copy(dstf, file)
	fmt.Printf("file saved to \"%s\"\n", dst)
	return nil
}

// mkdir -p
func mkdirp(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
}

// handle /upload endpoints
func uploadHandler(param *PathParam) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("access to %s for uploadHandler", r.URL.Path)
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		if !param.EnableUploads {
			http.NotFound(w, r)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, param.MaxUploadSize)

		log.Debug("parsing multipart form ...")
		err := r.ParseMultipartForm(param.MaxUploadSize)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debugf("%v bytes received", r.ContentLength)

		file, fhandler, err := r.FormFile("file")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		name := fhandler.Filename
		err = saveFile(file, name, param.FSPath)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), 500)
			return
		}

		// redirect to main page
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
	}
}

// handle /api/files
func apiListFiles(param *PathParam) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("access to %s for apiListFiles", r.URL.Path)
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		// start spinner
		s := spinner.New(spinner.CharSets[12], 100*time.Millisecond)
		s.Start()
		defer s.Stop()

		// get the files
		wpath := fmt.Sprintf("%s/%s", param.WebPath, webFilePath)
		files, err := getFiles(param.FSPath, wpath, param.HiddenFiles)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(files)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

// handle / endpoint
func redirectorHandler(to string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("access to /")
		log.Debugf("redirect / to %s", to)
		http.Redirect(w, r, to, http.StatusFound)
	}
}

// handle /<path> endpoint
func viewHandler(param *PathParam) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fmt.Sprintf("/%s", param.WebPath) {
			log.Debugf("bad access to %s for viewHandler (%s)", r.URL.Path, param.WebPath)
			http.NotFound(w, r)
			return
		}
		log.Debugf("access to %s", r.URL.Path)
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		// build page
		t := template.New("t")
		t, err := t.Parse(page)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), 500)
			return
		}

		data := TmplData{
			Title:           title,
			Version:         version,
			BaseName:        param.WebPath,
			UploadPath:      fmt.Sprintf("%s/%s", param.WebPath, webUploadPath),
			EnableUploads:   param.EnableUploads,
			EnableDownloads: param.EnableDownloads,
			Others:          param.Others,
		}
		log.Debugf("execute template with %#v", data)
		t.Execute(w, data)
	}
}

// setup and start http server
func startServer(param *Param) error {
	addr := fmt.Sprintf("%s:%d", param.Host, param.Port)

	// construct all endpoints
	var subs []string
	for _, p := range param.Paths {
		base := path.Base(p)
		base = normalizeString(base)
		subs = append(subs, base)
	}

	// handle main page
	main := fmt.Sprintf("/%s", subs[0])
	log.Debugf("serving / to %s", main)
	http.HandleFunc("/", redirectorHandler(main))

	// setup all endpoints
	for _, p := range param.Paths {
		base := path.Base(p)
		base = normalizeString(base)

		pparam := &PathParam{
			WebPath:         base,
			FSPath:          p,
			EnableUploads:   param.EnableUploads,
			EnableDownloads: param.EnableDownloads,
			MaxUploadSize:   param.MaxUploadSize,
			HiddenFiles:     param.HiddenFiles,
			Others:          subs,
		}

		// handle main page
		webp := fmt.Sprintf("/%s", base)
		log.Debugf("serving for %s: %s", pparam.FSPath, webp)
		http.HandleFunc(webp, viewHandler(pparam))

		// handle uploads
		if param.EnableUploads {
			webp = fmt.Sprintf("/%s/upload", base)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.HandleFunc(webp, uploadHandler(pparam))
			webp = fmt.Sprintf("/%s/upload/", base)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.HandleFunc(webp, uploadHandler(pparam))
		}

		// handle downloads
		if param.EnableDownloads {
			fs := http.FileServer(http.Dir(pparam.FSPath))
			webp = fmt.Sprintf("/%s/files", base)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.Handle(webp, http.StripPrefix(webp, fs))
			webp = fmt.Sprintf("/%s/files/", base)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.Handle(webp, http.StripPrefix(webp, fs))
		}

		// API file listing endpoint
		webp = fmt.Sprintf("/%s/api/files", base)
		log.Debugf("serving for %s: %s", pparam.FSPath, webp)
		http.HandleFunc(webp, apiListFiles(pparam))
		webp = fmt.Sprintf("/%s/api/files/", base)
		log.Debugf("serving for %s: %s", pparam.FSPath, webp)
		http.HandleFunc(webp, apiListFiles(pparam))
	}

	// start the server
	fmt.Printf("listening on \"%s\"\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		return err
	}
	return nil
}

// resolves a symlink
func resolveSymlink(path string) string {
	info, err := os.Lstat(path)
	if err != nil {
		log.Error(err)
		return path
	}
	if info.Mode()&os.ModeSymlink != os.ModeSymlink {
		return path
	}
	log.Debugf("\"%s\" is a symlink", path)
	origin, err := filepath.EvalSymlinks(path)
	//origin, err := os.Readlink(path)
	if err != nil {
		log.Error(err)
		return path
	}
	return origin
}

// normalize a string
func normalizeString(str string) string {
	str = strings.Replace(str, " ", "-", -1)
	return str
}

// resolve path
func resolvePath(path string) (string, error) {
	var err error
	path = resolveSymlink(path)
	path, err = filepath.Abs(path)
	return path, err
}

// print usage
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [<options>] [<path>...]\n", os.Args[0])
	flag.PrintDefaults()
}

// validate and normalize path
func checkPath(path string) string {
	err := isValidDir(path)
	if err != nil {
		log.Fatal(err)
	}

	path, err = resolvePath(path)
	if err != nil {
		log.Fatal(err)
	}

	return path
}

// entry point
func main() {
	hostArg := flag.String("host", "", "Host to listen to")
	portArg := flag.Int("port", 6969, "Port to listen to")
	maxUploadSizeArg := flag.String("max-upload", dfltMaxUploadSize, "Max upload size in bytes")
	upArg := flag.Bool("no-uploads", false, "Disable uploads")
	downArg := flag.Bool("no-downloads", false, "Disable downloads")
	hiddenArg := flag.Bool("show-hidden", false, "Show hidden files")
	helpArg := flag.Bool("help", false, "Show usage")
	versArg := flag.Bool("version", false, "Show version")
	debugArg := flag.Bool("debug", false, "Debug mode")
	flag.Parse()

	if *debugArg {
		log.SetLevel(log.DebugLevel)
		log.Info("debug mode enabled")
	}

	if *helpArg {
		usage()
		os.Exit(0)
	}

	if *versArg {
		fmt.Printf("%s v%s\n", os.Args[0], version)
		os.Exit(0)
	}

	var paths []string
	if len(flag.Args()) < 1 {
		paths = append(paths, checkPath("."))
	} else {
		for _, p := range flag.Args() {
			paths = append(paths, checkPath(p))
		}
	}

	MaxUploadSize, err := humanToSize(*maxUploadSizeArg)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// parameters
	param := &Param{
		Host:            *hostArg,
		Port:            *portArg,
		Paths:           paths,
		EnableUploads:   !*upArg,
		EnableDownloads: !*downArg,
		MaxUploadSize:   MaxUploadSize,
		HiddenFiles:     *hiddenArg,
	}
	log.Debugf("%#v", param)

	fmt.Printf("- version: %s\n", version)
	fmt.Printf("- paths:")
	for _, p := range param.Paths {
		fmt.Printf("  \"%s\"\n", p)
	}
	fmt.Printf("- download enabled: %v\n", param.EnableDownloads)
	fmt.Printf("- upload enabled: %v\n", param.EnableUploads)
	if param.EnableUploads {
		fmt.Printf("- upload max size: %v\n", sizeToHuman(param.MaxUploadSize))
	}
	fmt.Printf("- show hidden files: %v\n", param.HiddenFiles)

	if len(param.Paths) < 1 {
		log.Fatal("no path provided")
	}

	err = startServer(param)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}
