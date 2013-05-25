package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func APIHandle(w http.ResponseWriter, r *http.Request) {
	endpoint := mux.Vars(r)["endpoint"]
	api := mux.Vars(r)["api"]
	var item string
	r.ParseForm()
	if r.Method == "GET" {
		if api == "ip" {
			r.Form, _ = url.ParseQuery(
				"show_type=ip&show_list=all")
		} else {
			r.Form, _ = url.ParseQuery(
				"show_type=variable&show_list=all")
		}
		item = endpoint + "_" + api + "_list"
	} else {
		item = endpoint + "_" + api
	}
	endpoints, ok := setting[item]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	resp := make(chan string)
	count := 0
	for _, h := range endpoints {
		go sendrequest(h, r.Form, resp)
		count++
	}
	rst := ""
	for i := 0; i < count; i++ {
		rst += <-resp
	}
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rst))
}

func sendrequest(url string, data map[string][]string, rst chan string) {
	client := http.Client{}
	cal := time.Time{}
	timer := cal.UnixNano()
	resp, err := client.PostForm(url, data)
	if err != nil {
		log.Println("connect timeout", err)
		rst <- "error to post" + url + "\n"
	} else {
		if resp.StatusCode != 200 {
			log.Printf("unsuccessfull return %s\n", resp.Status)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("read response",err)
		}
		resp.Body.Close()
		timer = cal.UnixNano() - timer
		rstbody := "----START----\n" + url + " completed at " +
			strconv.FormatFloat(float64(timer/1e9), 'f', 5, 64) +
			"\n" + string(body) + "\n----END----\n"
		rst <- rstbody
	}
}
