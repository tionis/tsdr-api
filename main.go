package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/UniPassauTestBot", UniPassauTestBot)
	http.ListenAndServe(":"+port, nil)
}

func UniPassauTestBot(w http.ResponseWriter, r *http.Request) {
	// Test Code
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
	w.Write([]byte("OK"))
}
