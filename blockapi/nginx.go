package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func genrequest(requrl string, name string, postdata map[string][]string) string {
	hosts := read_config(name)
	var rstbody string
	cmd := make(chan *HttpRequest, 100)
	rst := make(chan string, 100)
	for i := range hosts {
		req := &HttpRequest{
			url:  "http://" + strings.TrimSpace(hosts[i]) + requrl,
			data: postdata,
		}
		go sendrequest(cmd, rst)
		cmd <- req
	}
	for i := 0; i < len(hosts); i++ {
		rstbody += <-rst
	}
	return rstbody
}

func sendrequest(req chan *HttpRequest, rst chan string) {
	reqinfo := <-req
	client := &http.Client{}
	cal := time.Time{}
	timer := cal.UnixNano()
	log.Println(reqinfo.url, "expire_time:", reqinfo.data["ban_expire"], "ban_list:", reqinfo.data["ban_list"])
	resp, err := client.PostForm(reqinfo.url, reqinfo.data)
	if err != nil {
		log.Println("connect timeout", err)
		rst <- "error to post" + reqinfo.url + "\n"
	} else {
		if resp.StatusCode != 200 {
			log.Printf("unsuccessfull return %s\n", resp.Status)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		timer = cal.UnixNano() - timer
		rstbody := "----START----\n" + reqinfo.url + " completed at " + strconv.FormatFloat(float64(timer/1e9), 'f', 5, 64) + "\n" + string(body) + "\n----END----\n"
		rst <- rstbody
	}
}
