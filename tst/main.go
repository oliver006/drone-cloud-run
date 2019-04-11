package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func versionHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("versionHandler() received a request.")
	version := os.Getenv("VERSION")
	if version == "" {
		version = "NO-VERSION"
	}
	fmt.Fprintf(w, "version: [%s]", version)
}

func main() {
	log.Print("Test app started.")

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/version", versionHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
