package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
)

var (
	port = flag.String("port", "8080", "access port")
)

func main() {
	flag.Parse()
	http.HandleFunc("/", show_ip)
	http.ListenAndServe(":"+*port, nil)
}

func show_ip(w http.ResponseWriter, r *http.Request) {
	addr := strings.Split(r.RemoteAddr, ":")
	io.WriteString(w, addr[0])
	log.Println(r.RemoteAddr, "get show_ip")
}
