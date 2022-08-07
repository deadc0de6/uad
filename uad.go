/*
author: deadc0de6 (https://github.com/deadc0de6)
Copyright (c) 2020, deadc0de6
*/

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	fCreationRights = 0666
	version         = "0.8.0"
	// web endpoint for upload
	webUploadPath = "upload/"
	webFilePath   = "files/"
	name          = "uad"
	pathName      = "path"
)

var (
	dfltMaxUploadSize = "1G"
	units             = []string{"B", "K", "M", "G", "T", "P"}
	//go:embed "page.html"
	page string
)

var (
	// cli parameters
	rootCmd   *cobra.Command
	cliParams = Param{}
)

type Settings struct {
	Host            string
	Port            int
	NPaths          []*NamedPath
	MaxUploadSize   int64
	EnableUploads   bool
	EnableDownloads bool
	ShowHiddenFiles bool
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

// NamedPath a named path
type NamedPath struct {
	Path string
	Name string
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

// Param global parameters
type Param struct {
	Host                string
	Port                int
	Debug               bool
	MaxUploadSizeString string
	EnableUploads       bool
	EnableDownloads     bool
	ShowHiddenFiles     bool
	NameFromParent      bool
	ServeSubs           bool
}

// init
func init() {
	// env variables
	viper.SetEnvPrefix(strings.ToUpper(name))
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// cli configs
	rootCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s [flags] <path>...", name),
		Version: version,
		Short:   "Upload And Download",
		Long:    `Very tiny web server allowing to upload and download files.`,
		Run: func(cmd *cobra.Command, args []string) {
			process(args)
		},
	}

	// set flags
	rootCmd.PersistentFlags().StringVarP(&cliParams.Host, "host", "A", viper.GetString("HOST"), "Host to listen to")
	defPort := viper.GetInt("PORT")
	if defPort == 0 {
		defPort = 6969
	}
	rootCmd.PersistentFlags().IntVarP(&cliParams.Port, "port", "p", defPort, "Port to listen to")
	defMax := viper.GetString("MAX_UPLOAD")
	if len(defMax) < 1 {
		defMax = string(dfltMaxUploadSize)
	}
	rootCmd.PersistentFlags().StringVarP(&cliParams.MaxUploadSizeString, "max-upload", "m", defMax, "Max upload size")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.EnableUploads, "no-uploads", "U", viper.GetBool("NO_UPLOADS"), "Disable uploads")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.EnableDownloads, "no-downloads", "D", viper.GetBool("NO_DOWNLOADS"), "Disable downloads")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.ShowHiddenFiles, "show-hidden", "H", viper.GetBool("SHOW_HIDDEN"), "Show hidden files")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.ServeSubs, "serve-subs", "s", viper.GetBool("SERVE_SUBS"), "Serve all directories found in <path>")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.NameFromParent, "from-parent", "P", viper.GetBool("FROM_PARENT"), "Paths get their names from parent dir")
	rootCmd.PersistentFlags().BoolVarP(&cliParams.Debug, "debug", "d", viper.GetBool("DEBUG"), "Debug mode")
}

// cli parsing
func parseCli() error {
	return rootCmd.Execute()
}

// is path a valid directory
func isValidDir(path string) bool {
	dir, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !dir.IsDir() {
		return false
	}
	return true
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

func isHidden(path string) bool {
	entries := strings.Split(path, string(os.PathSeparator))
	for _, entry := range entries {
		if strings.HasPrefix(entry, ".") {
			log.Debugf("skipping hidden file: %s", path)
			return true
		}
	}
	return false
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
			if isHidden(rpath) {
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
			Title:           name,
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
func startServer(settings *Settings) error {
	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)

	// construct all endpoints
	var subs []string
	for _, n := range settings.NPaths {
		subs = append(subs, n.Name)
	}

	// handle main page
	log.Debugf("serving / to %s", subs[0])
	http.HandleFunc("/", redirectorHandler(subs[0]))

	// setup all endpoints
	for _, p := range settings.NPaths {
		pparam := &PathParam{
			WebPath:         p.Name,
			FSPath:          p.Path,
			EnableUploads:   settings.EnableUploads,
			EnableDownloads: settings.EnableDownloads,
			MaxUploadSize:   settings.MaxUploadSize,
			HiddenFiles:     settings.ShowHiddenFiles,
			Others:          subs,
		}

		// handle main page
		webp := fmt.Sprintf("/%s", p.Name)
		log.Debugf("serving for %s: %s", pparam.FSPath, webp)
		http.HandleFunc(webp, viewHandler(pparam))

		// handle uploads
		if settings.EnableUploads {
			webp = fmt.Sprintf("/%s/upload", p.Name)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.HandleFunc(webp, uploadHandler(pparam))
			webp = fmt.Sprintf("/%s/upload/", p.Name)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.HandleFunc(webp, uploadHandler(pparam))
		}

		// handle downloads
		if settings.EnableDownloads {
			fs := http.FileServer(http.Dir(pparam.FSPath))
			webp = fmt.Sprintf("/%s/files", p.Name)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.Handle(webp, http.StripPrefix(webp, fs))
			webp = fmt.Sprintf("/%s/files/", p.Name)
			log.Debugf("serving for %s: %s", pparam.FSPath, webp)
			http.Handle(webp, http.StripPrefix(webp, fs))
		}

		// API file listing endpoint
		webp = fmt.Sprintf("/%s/api/files", p.Name)
		log.Debugf("serving for %s: %s", pparam.FSPath, webp)
		http.HandleFunc(webp, apiListFiles(pparam))
		webp = fmt.Sprintf("/%s/api/files/", p.Name)
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

// resolve path
func resolvePath(path string) (string, error) {
	var err error
	path = resolveSymlink(path)
	path, err = filepath.Abs(path)
	return path, err
}

// validate and normalize path
func checkPath(path string) string {
	log.Debugf("check valid path for \"%s\"", path)

	if !isValidDir(path) {
		log.Error(fmt.Errorf("\"%s\" is not a valid directory", path))
		return ""
	}

	return path
}

// get a named path
func getNamedPath(str string, fromParent bool, idx int) *NamedPath {
	var name, path string

	if strings.Contains(str, ":") {
		fields := strings.Split(str, ":")
		name = fields[0]
		path = fields[1]
	} else {
		path = str
	}

	path, err := resolvePath(path)
	if err != nil {
		log.Fatal(err)
	}

	if fromParent {
		name = filepath.Base(path)
		name = strings.Replace(name, " ", "-", -1)
	}

	if len(name) < 1 {
		name = fmt.Sprintf("%s%d", pathName, idx)
	}

	log.Debugf("from arg \"%s\" to name:%s and path:%s", str, name, path)
	path = checkPath(path)
	if len(path) < 1 {
		return nil
	}
	np := NamedPath{
		Path: path,
		Name: name,
	}
	return &np
}

func process(args []string) {
	if cliParams.Debug {
		log.SetLevel(log.DebugLevel)
		log.Info("debug mode enabled")
	}

	log.Debugf("path from parent: %v", cliParams.NameFromParent)

	log.Debugf("paths args: %#v", args)
	var namedpaths []*NamedPath
	if len(args) < 1 {
		np := getNamedPath(".", cliParams.NameFromParent, 0)
		if np != nil {
			namedpaths = append(namedpaths, np)
		}
	} else {
		for idx, p := range args {
			np := getNamedPath(p, cliParams.NameFromParent, idx)
			if np != nil {
				namedpaths = append(namedpaths, np)
			}
		}
	}

	// serve directory subs
	if cliParams.ServeSubs {
		var subs []*NamedPath
		var cnt int
		for _, p := range namedpaths {
			// read entry in path
			files, err := os.ReadDir(p.Path)
			if err != nil {
				log.Error(err)
				continue
			}
			for _, f := range files {
				subPath := path.Join(p.Path, f.Name())
				if !isValidDir(subPath) {
					// skip non-dir
					continue
				}
				if !cliParams.ShowHiddenFiles {
					// skip hidden files
					if isHidden(subPath) {
						continue
					}
				}
				np := getNamedPath(subPath, cliParams.NameFromParent, cnt)
				if np != nil {
					subs = append(subs, np)
					cnt++
				}
			}
		}
		namedpaths = subs
	}

	if log.GetLevel() == log.DebugLevel {
		// debug print parsed paths
		for _, np := range namedpaths {
			log.Debugf("named path %s: %s", np.Name, np.Path)
		}
	}

	// max upload size parsing
	MaxUploadSize, err := humanToSize(cliParams.MaxUploadSizeString)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// settings
	settings := &Settings{
		Host:            cliParams.Host,
		Port:            cliParams.Port,
		NPaths:          namedpaths,
		EnableUploads:   !cliParams.EnableUploads,
		EnableDownloads: !cliParams.EnableDownloads,
		MaxUploadSize:   MaxUploadSize,
		ShowHiddenFiles: cliParams.ShowHiddenFiles,
	}

	// print info
	fmt.Println("settings:")
	fmt.Printf("- version: %s\n", version)
	fmt.Printf("- paths:\n")
	for _, p := range settings.NPaths {
		fmt.Printf("  %s: \"%s\"\n", p.Name, p.Path)
	}
	fmt.Printf("- download enabled: %v\n", settings.EnableDownloads)
	fmt.Printf("- upload enabled: %v\n", settings.EnableUploads)
	if settings.EnableUploads {
		fmt.Printf("- upload max size: %v\n", sizeToHuman(settings.MaxUploadSize))
	}
	fmt.Printf("- show hidden files: %v\n", settings.ShowHiddenFiles)

	if len(settings.NPaths) < 1 {
		log.Fatal("no path provided")
	}

	err = startServer(settings)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

// entry point
func main() {
	err := parseCli()
	if err != nil {
		log.Fatal(err)
	}
}
