package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	_ "github.com/heroku/x/hmetrics/onload"
)

// Main and Init
func main() {
	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start and init Webhook Component
	port := os.Getenv("PORT")
	directory := "static"
	fileServer := http.FileServer(FileSystem{http.Dir(directory)})
	http.Handle("/favicon.ico", fileServer)
	http.Handle("/", fileServer)
	http.HandleFunc("/echo", httpecho)
	http.ListenAndServe(":"+port, nil)
}

func httpecho(w http.ResponseWriter, r *http.Request) {
	// Test Code
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	w.Write([]byte(string(requestDump)))
}

// FileSystem custom file system handler
type FileSystem struct {
	fs http.FileSystem
}

// Handles http errors
func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		directory := "static"
		fileServer := http.FileServer(FileSystem{http.Dir(directory)})
		fileServer.ServeHTTP(w, r)
	}
}

// Open opens file
func (fs FileSystem) Open(path string) (http.File, error) {
	f, err := fs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if _, err := fs.fs.Open(index); err != nil {
			return nil, err
		}
	}

	return f, nil
}
