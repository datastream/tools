package main

import (
	"flag"
	"github.com/garyburd/twister/server"
	"github.com/garyburd/twister/web"
	"io"
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

type Request struct {
	iprequest *IPRequest
	rsp chan *Response
}

var check_chan chan WhiteListRequest

func NginxApi(req *web.Request) {
	pat := strings.Split(req.URL.Path, "/")
	if pat[1] != "lvs" {
		var params map[string][]string
		if req.Method == "POST" {
			params = req.Param
		} else {
			if pat[2] == "ip" {
				params, _ = url.ParseQuery("show_type=ip&show_list=all")
			} else {
				params, _ = url.ParseQuery("show_type=variable&show_list=all")
			}
		}
		rst := genrequest("/limit_interface_"+pat[2], pat[1], params)
		w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain; charset=\"utf-8\"")
		io.WriteString(w, rst)
	} else {
		FirewallApi(req)
	}
}

func FirewallApi(req *web.Request) {
	w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain; charset=\"utf-8\"")
	parts := strings.Split(req.URL.Path, "/")
	if req.Method == "GET" {
		param, _ := url.ParseQuery("action_type=list")
		io.WriteString(w, gen_protov1(parts[2], param))
	} else {
		if len(parts) > 2 {
			io.WriteString(w, gen_protov1(parts[2], req.Param))
		} else {
			io.WriteString(w, "error")
		}
	}
	return
}

func main() {
	flag.Parse()
	check_chan = make(chan WhiteListRequest)
	go check_whitelist(check_chan)
	h := web.NewRouter().
		Register("/<:.*>", "*", web.FormHandler(8148, false, web.HandlerFunc(NginxApi)))
	server.Run(":"+*port, h)
}
