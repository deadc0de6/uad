/*
author: deadc0de6 (https://github.com/deadc0de6)
Copyright (c) 2020, deadc0de6
*/

package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	MaxUploadSize   = 1 << 30
	FCreationRights = 0666
	Version         = "0.1"
	FileWebPath     = "/files/"
	Title           = "uad"
)

var (
	UploadDst       = "./uploads"
	Units           = []string{"B", "KB", "MB", "GB", "TB", "PB"}
	EnableUploads   = true
	EnableDownloads = true
)

type TmplData struct {
	Title           string
	Files           []HTMLFile
	EnableUploads   bool
	EnableDownloads bool
}

type HTMLFile struct {
	Name     string
	Size     string
	Modified string
	Path     string
}

// return size in human readable format
func humanSize(bytes int64) string {
	size := bytes
	for _, unit := range Units {
		if size < 1024 {
			return fmt.Sprintf("%d %s", size, unit)
		}
		size = size / 1024
	}
	return "??"
}

// walk a directory and return HTMLFiles list
func walker(hfiles *[]HTMLFile) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		hfile := HTMLFile{
			Name:     info.Name(),
			Size:     humanSize(info.Size()),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			Path:     filepath.Join(FileWebPath, info.Name()),
		}

		*hfiles = append(*hfiles, hfile)
		return nil
	}
}

// get list of files in upload dir
func getFiles(path string) ([]HTMLFile, error) {
	var hfiles []HTMLFile
	if !EnableDownloads {
		return nil, nil
	}
	err := filepath.Walk(path, walker(&hfiles))
	if err != nil {
		return nil, err
	}
	return hfiles, nil
}

// save file locally from upload form
func saveFile(file io.Reader, name string) error {
	mkdirp(UploadDst)
	dst := filepath.Join(UploadDst, name)
	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, FCreationRights)
	if err != nil {
		return err
	}
	io.Copy(dstf, file)
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
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	if !EnableUploads {
		http.NotFound(w, r)
		return
	}

	err := r.ParseMultipartForm(MaxUploadSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		http.Redirect(w, r, r.Header.Get("Referer"), 302)
		//http.Error(w, err.Error(), 500)
		return
	}
	file, fhandler, err := r.FormFile("file")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		http.Redirect(w, r, r.Header.Get("Referer"), 302)
		//http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	name := fhandler.Filename
	err = saveFile(file, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		http.Error(w, err.Error(), 500)
		return
	}
	// redirect to main page
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// handle / endpoint
func viewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	files, err := getFiles(UploadDst)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	t := template.New("t")
	t, err = t.Parse(Page)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		http.Error(w, err.Error(), 500)
		return
	}

	data := TmplData{
		Title:           Title,
		Files:           files,
		EnableUploads:   EnableUploads,
		EnableDownloads: EnableDownloads,
	}
	t.Execute(w, data)
}

// setup and start http server
func startServer(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	// handle uploads
	if EnableUploads {
		http.HandleFunc("/upload", uploadHandler)
	}

	// handle main page
	http.HandleFunc("/", viewHandler)

	// handle downloads
	if EnableDownloads {
		fs := http.FileServer(http.Dir(UploadDst))
		http.Handle(FileWebPath, http.StripPrefix(FileWebPath, fs))
	}

	// start the server
	fmt.Printf("listening on \"%s\"\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		return err
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [<options>]\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	hostArg := flag.String("host", "", "Host to listen to")
	portArg := flag.Int("port", 6969, "Port to listen to")
	dstArg := flag.String("path", UploadDst, "Destination path for uploaded files")
	upArg := flag.Bool("no-uploads", false, "Disable uploads")
	downArg := flag.Bool("no-downloads", false, "Disable downloads")
	helpArg := flag.Bool("help", false, "Show usage")
	versArg := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *helpArg {
		usage()
		os.Exit(0)
	}

	if *versArg {
		fmt.Printf("%s v%s\n", os.Args[0], Version)
		os.Exit(0)
	}

	UploadDst = *dstArg
	EnableUploads = !*upArg
	EnableDownloads = !*downArg
	fmt.Printf("upload destination: %s\n", UploadDst)
	fmt.Printf("uploads enabled: %v\n", EnableUploads)
	fmt.Printf("downloads enabled: %v\n", EnableDownloads)

	err := startServer(*hostArg, *portArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
