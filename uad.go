/*
author: deadc0de6 (https://github.com/deadc0de6)
Copyright (c) 2020, deadc0de6
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const (
	FCreationRights = 0666
	Version         = "0.2.1"
	FileWebPath     = "/files/"
	Title           = "uad"
)

var (
	DfltUploadDst     = "./uploads"
	DfltMaxUploadSize = int64(1 << 30)
	Units             = []string{"B", "K", "M", "G", "T", "P"}
)

type Param struct {
	Host            string
	Port            int
	UploadDst       string
	MaxUploadSize   int64
	EnableUploads   bool
	EnableDownloads bool
}

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
func SizeToHuman(bytes int64) string {
	size := bytes
	rest := int64(0)
	for _, unit := range Units {
		if size < 1024 {
			return fmt.Sprintf("%d.%d%s", size, rest, unit)
		}
		rest = size % 1024
		size = size / 1024
	}
	return "??"
}

// return human size in bytes
func HumanToSize(size string) (int64, error) {
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
			Size:     SizeToHuman(info.Size()),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			Path:     filepath.Join(FileWebPath, info.Name()),
		}

		*hfiles = append(*hfiles, hfile)
		return nil
	}
}

// get list of files in upload dir
func getFiles(path string, enabled bool) ([]HTMLFile, error) {
	var hfiles []HTMLFile
	if !enabled {
		return nil, nil
	}
	err := filepath.Walk(path, walker(&hfiles))
	if err != nil {
		return nil, err
	}
	return hfiles, nil
}

// save file locally from upload form
func saveFile(file io.Reader, name string, dstdir string) error {
	mkdirp(dstdir)
	dst := filepath.Join(dstdir, name)
	fmt.Printf("saving file to \"%s\" ...\n", dst)
	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, FCreationRights)
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
func uploadHandler(param Param) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		if !param.EnableUploads {
			http.NotFound(w, r)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, param.MaxUploadSize)

		fmt.Printf("parsing multipart form ...\n")
		err := r.ParseMultipartForm(param.MaxUploadSize)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("%v bytes received\n", r.ContentLength)

		file, fhandler, err := r.FormFile("file")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		name := fhandler.Filename
		err = saveFile(file, name, param.UploadDst)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), 500)
			return
		}

		// redirect to main page
		http.Redirect(w, r, r.Header.Get("Referer"), 302)
	}
	return http.HandlerFunc(fn)
}

// handle / endpoint
func viewHandler(param Param) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		files, err := getFiles(param.UploadDst, param.EnableDownloads)
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
			EnableUploads:   param.EnableUploads,
			EnableDownloads: param.EnableDownloads,
		}
		t.Execute(w, data)
	}
	return http.HandlerFunc(fn)
}

// setup and start http server
func startServer(param Param) error {
	addr := fmt.Sprintf("%s:%d", param.Host, param.Port)

	// handle uploads
	if param.EnableUploads {
		http.Handle("/upload", uploadHandler(param))
	}

	// handle main page
	http.Handle("/", viewHandler(param))

	// handle downloads
	if param.EnableDownloads {
		fs := http.FileServer(http.Dir(param.UploadDst))
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
	dstArg := flag.String("path", DfltUploadDst, "Destination path for uploaded files")
	MaxUploadSizeArg := flag.String("max-upload", SizeToHuman(DfltMaxUploadSize), "Max upload size in bytes")
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

	MaxUploadSize, err := HumanToSize(*MaxUploadSizeArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	param := Param{
		Host:            *hostArg,
		Port:            *portArg,
		UploadDst:       *dstArg,
		EnableUploads:   !*upArg,
		EnableDownloads: !*downArg,
		MaxUploadSize:   MaxUploadSize,
	}

	fmt.Printf("download enabled: %v\n", param.EnableDownloads)
	fmt.Printf("upload enabled: %v\n", param.EnableUploads)
	fmt.Printf("upload destination: %s\n", param.UploadDst)
	fmt.Printf("upload max size: %v\n", SizeToHuman(param.MaxUploadSize))

	err = startServer(param)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
