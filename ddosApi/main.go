package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	bind   = flag.String("port", "0.0.0.0:1234", "http port")
)

func main() {
	flag.Parse()
	setting, err := readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	API := &DDoSAPI{}
	API.Setting = setting
	API.Run()
	http.HandleFunc("/", API.APIHandle)
	err = http.ListenAndServe(setting["port"], nil)
	if err != nil {
		log.Println(err)
	}
	API.Stop()
}
