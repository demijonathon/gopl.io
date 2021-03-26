package main

import (
	"fmt"
	"log"
	"net/http"
)

type database struct {
	sourceUrl string
	destUrl   string
}

func main() {
	db := database{"http://101receteas.es/", "https://cookpad.com/es/misc/101recetas"}
	log.Fatal(http.ListenAndServe("localhost:8080", db))
}

func (db database) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Host {
	case db.sourceUrl:
		fmt.Fprintf(w, "301 Moved Permanently.\nLocation: %s", db.destUrl)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "no such redirection for %s/%s\n", req.URL)
	}
	//switch req.URL.Path {
	//}
}
