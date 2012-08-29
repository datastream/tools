package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var (
	port       = flag.String("port", "1234", "access port")
	mongouri   = flag.String("mongouri", "mongodb://myuser:mypass@localhost:27017/mydatabase", "MONGODB RUI")
	user       = flag.String("user", "admin", "mongodb user")
	password   = flag.String("passwd", "admin", "mongodb password")
	dbname     = flag.String("db", "mydatabase", "mongodb database")
	collection = flag.String("collection", "metrics", "mongodb collection")
)

type HttpRequest struct {
	url  string
	data url.Values
}

type WhiteListRequest struct {
	ip    string
	hosts []string
}

type FirewallRequest struct {
	iprequest *IPRequest
	rsp       chan *Response
}

var check_chan chan WhiteListRequest

func NginxApi(w http.ResponseWriter, req *http.Request) {
	pat := strings.Split(req.URL.Path, "/")
	if req.Method != "POST" {
		if pat[2] == "ip" {
			req.Form, _ = url.ParseQuery("show_type=ip&show_list=all")
		} else {
			req.Form, _ = url.ParseQuery("show_type=variable&show_list=all")
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
	check_chan = make(chan WhiteListRequest)
	go check_whitelist(check_chan)
	http.HandleFunc("/", ReadmeApi)
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Println(err)
	}
}
