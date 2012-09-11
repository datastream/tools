package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	file = flag.String("file", "/var/log/apache/access.log", "filenema")
	host = flag.String("host", "upload-9", "hostname")
)

func init() {
	flag.Parse()
}

type Request struct {
	Url  string
	Host string
}

func main() {
	req_chan := make(chan *Request)
	rst_chan := make(chan int)
	go read_file(*file, req_chan)
	go sendrequest(req_chan, *host, rst_chan)
	rst := <-rst_chan
	log.Println(rst)
}

func sendrequest(req_chan chan *Request, host string, done chan int) {
	client := &http.Client{}
	good_count := 0
	bad_count := 0
	for {
		v, ok := <-req_chan
		if !ok {
			break
		}
		req, err := http.NewRequest("GET", "http://"+host+v.Url, nil)

		if err != nil {
			log.Printf("Error req")
			continue
		}
		req.Host = v.Host
		resp, err := client.Do(req)
		if err != nil {
			log.Println("connet timeout:", host, err)
			break
		}
		if resp.StatusCode != 200 {
			log.Println("return ", resp.Status, v.Url, v.Host)
			bad_count++
		} else {
			good_count++
		}
		_, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	done <- bad_count
}

func read_file(name string, req_chan chan *Request) {
	fd, err := os.Open("./" + name)
	if err != nil {
		log.Println("server list file open Error:", name)
	} else {
		f := bufio.NewReader(fd)
		for {
			line, err := f.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("file readline error")
				break
			}
			log := strings.Split(strings.TrimSpace(line), " ")
			if len(log) < 2 {
				continue
			}
			req := &Request{
				Url:  log[0],
				Host: log[1],
			}
			req_chan <- req
		}
	}
	close(req_chan)
}
