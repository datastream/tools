package main

import (
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

var (
	bind   = flag.String("port", "0.0.0.0:1234", "http port")
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string][]string

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/{endpoint}/{api}", APIHandle)
	http.Handle("/", r)
	err = http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Fatal(err)
	}
}
