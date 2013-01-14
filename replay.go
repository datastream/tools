package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var (
	file = flag.String("file", "/var/log/apache/access.log", "filenema")
	host = flag.String("host", "upload-9", "hostname")
	max  = flag.Int("max", 128, "max connection")
)

type Request struct {
	Url   string
	Host  string
	Ua    string
	Refer string
}

func main() {
	flag.Parse()
	done := make(chan int)
	go read_file(*file)
	//for i := 0; i < *max; i++ {
	//	go sendrequest(req_chan, *host)
	//}
	<-done
}

func sendrequest(req_chan chan *Request, host string) {
	connect_count := 0
	var client *http.Client
	for {
		connect_count++
		if connect_count < 9 {
			client = &http.Client{}
			connect_count = 0
		}
		v, ok := <-req_chan
		if !ok {
			log.Println("error request")
			break
		}
		req, err := http.NewRequest("GET", "http://"+host+v.Url, nil)

		if err != nil {
			log.Printf("Error req")
			continue
		}
		req.Host = v.Host
		if len(v.Refer) > 1 {
			req.Header.Set("Referer", v.Refer)
		}
		if len(v.Ua) > 1 {
			req.Header.Set("User-Agent", v.Ua)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("connet timeout:", host, err)
			break
		}
		_, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func sendrequest2(v *Request, host string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://"+host+v.Url, nil)

	if err != nil {
		log.Printf("Error req")
	}
	req.Host = v.Host
	if len(v.Refer) > 1 {
		req.Header.Set("Referer", v.Refer)
	}
	if len(v.Ua) > 1 {
		req.Header.Set("User-Agent", v.Ua)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("connet timeout:", host, err)
		return
	}
	_, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}

func read_file(file_name string) {
	cmd := exec.Command("/bin/cat", file_name)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Read Pipe error", err)
	}
	f := bufio.NewReaderSize(stdout, 10240)
	err = cmd.Start()
	if err != nil {
		log.Println("run error", err)
	}
	for {
		line, err := f.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("file readline error")
			break
		}
		parsed_line := strings.Split(strings.TrimSpace(line), "\"")
		if len(parsed_line) < 12 {
			continue
		}
		url := strings.Split(parsed_line[1], " ")
		if len(url) < 2 {
			continue
		}
		req := &Request{
			Url:   url[1],
			Host:  parsed_line[7],
			Refer: parsed_line[3],
			Ua:    parsed_line[11],
		}
		go sendrequest2(req, *host)
		//req_chan <- req
	}
	log.Println(err)
	//close(req_chan)
}
