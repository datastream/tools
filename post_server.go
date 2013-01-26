package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	port = flag.String("port", "1234", "access port")
)

func PostApi(w http.ResponseWriter, req *http.Request) {
	_, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello world!"))
}

func main() {
	flag.Parse()
	http.HandleFunc("/", PostApi)
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Println(err)
	}
}
