package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var (
	port = flag.String("port", "1234", "access port")
)

type HttpRequest struct {
	url  string
	data url.Values
}

type FirewallRequest struct {
	iprequest *IPRequest
	rsp       chan *Response
}

func NginxApi(w http.ResponseWriter, req *http.Request) {
	pat := strings.Split(req.URL.Path, "/")
	if req.Method != "POST" {
		if pat[2] == "ip" {
			req.Form, _ = url.ParseQuery(
				"show_type=ip&show_list=all")
		} else {
			req.Form, _ = url.ParseQuery(
				"show_type=variable&show_list=all")
		}
	}
	rst := genrequest("/limit_interface_"+pat[2], pat[1], req.Form)
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rst))
}

func FirewallApi(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	parts := strings.Split(req.URL.Path, "/")
	if req.Method == "GET" {
		req.Form, _ = url.ParseQuery("action_type=list")
	}
	if len(parts) > 2 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("error"))
		return
	}
	w.Write([]byte(gen_protov1(parts[2], req)))
}

func ReadmeApi(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	req.ParseForm()
	if len(parts) > 0 {
		if parts[1] == "lvs" {
			FirewallApi(w, req)
		} else {
			NginxApi(w, req)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world!"))
	}
}
func main() {
	flag.Parse()
	http.HandleFunc("/", ReadmeApi)
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Println(err)
	}
}
