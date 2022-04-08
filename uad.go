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
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

const (
	fCreationRights = 0666
	version         = "0.6.4"
	// web endpoint for files
	downloadPath = "files/"
	// web endpoint for upload
	uploadPath = "upload/"
	title      = "uad"
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
	FilePath        string
	DlPath          string
	UpPath          string
	MaxUploadSize   int64
	EnableUploads   bool
	EnableDownloads bool
	HiddenFiles     bool
}

// TmplData template parameters
type TmplData struct {
	Title           string
	Version         string
	UploadPath      string
	EnableUploads   bool
	EnableDownloads bool
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
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		rpath, err := filepath.Rel(base, path)
		if err != nil {
			log.Error(err)
			return nil
		}

		if !hiddenFiles {
			if strings.HasPrefix(rpath, ".") {
				//log.Debugf("skipping hidden file: %s", rpath)
				return nil
			}
		}

		wpath := filepath.Join(webBase, rpath)
		hfile := &HTMLFile{
			Name:     name,
			Size:     sizeToHuman(info.Size()),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			WPath:    wpath,
			RPath:    rpath,
		}
		log.Debugf("%#v", hfile)

		*hfiles = append(*hfiles, hfile)
		return nil
	}
}

// get list of files in upload dir
func getFiles(path string, webPath string, hidden bool) ([]*HTMLFile, error) {
	var hfiles []*HTMLFile
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

// handle /upload endpoint
func uploadHandler(param *Param) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		err = saveFile(file, name, param.FilePath)
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
func apiListFiles(param *Param) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}
		files, err := getFiles(param.FilePath, param.DlPath, param.HiddenFiles)
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
func viewHandler(param *Param) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		log.Debugf("access to /")
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
			UploadPath:      uploadPath,
			EnableUploads:   param.EnableUploads,
			EnableDownloads: param.EnableDownloads,
		}
		t.Execute(w, data)
	}
}

// setup and start http server
func startServer(param *Param) error {
	addr := fmt.Sprintf("%s:%d", param.Host, param.Port)

	// handle main page
	http.HandleFunc("/", viewHandler(param))

	// handle uploads
	if param.EnableUploads {
		http.HandleFunc("/upload", uploadHandler(param))
		http.HandleFunc("/upload/", uploadHandler(param))
	}

	// handle downloads
	if param.EnableDownloads {
		log.Debugf("serve files from %s under %s", param.FilePath, param.DlPath)
		fs := http.FileServer(http.Dir(param.FilePath))
		http.Handle("/files/", http.StripPrefix("/files/", fs))
		http.Handle("/files", http.StripPrefix("/files", fs))
	}

	// API file listing endpoint
	http.HandleFunc("/api/files", apiListFiles(param))
	http.HandleFunc("/api/files/", apiListFiles(param))

	// start the server
	fmt.Printf("listening on \"%s\"\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		return err
	}
	return nil
}

// print usage
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [<options>] [<work-directory>]\n", os.Args[0])
	flag.PrintDefaults()
}

// entry point
func main() {
	var filePath string

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

	if len(flag.Args()) < 1 {
		filePath = "."
	} else {
		filePath = flag.Args()[0]
	}

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

	// validate paths
	err := isValidDir(filePath)
	if err != nil {
		log.Fatal(err)
	}
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		log.Fatal(err)
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
		FilePath:        filePath,
		DlPath:          downloadPath,
		UpPath:          uploadPath,
		EnableUploads:   !*upArg,
		EnableDownloads: !*downArg,
		MaxUploadSize:   MaxUploadSize,
		HiddenFiles:     *hiddenArg,
	}
	log.Debugf("%#v", param)

	fmt.Printf("- version: %s\n", version)
	fmt.Printf("- path: \"%s\"\n", param.FilePath)
	fmt.Printf("- download path: %s\n", param.DlPath)
	fmt.Printf("- download enabled: %v\n", param.EnableDownloads)
	fmt.Printf("- upload path: %s\n", param.UpPath)
	fmt.Printf("- upload enabled: %v\n", param.EnableUploads)
	if param.EnableUploads {
		fmt.Printf("- upload max size: %v\n", sizeToHuman(param.MaxUploadSize))
	}
	fmt.Printf("- show hidden files: %v\n", param.HiddenFiles)

	err = startServer(param)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}
