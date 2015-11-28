package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/datastream/aws"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	reqMethod     = flag.String("X", "GET", "Request method")
	reqBodyFile   = flag.String("f", "", "file include Post body")
	config        = flag.String("c", "s4.json", "aws signv4 config")
	reqBodyString = flag.String("b", "", "request body")
	debug         = flag.String("v", "", "debug")
)

type stringsFlag []string

func (v *stringsFlag) Set(value string) error {
	*v = append(*v, value)
	return nil
}
func (v *stringsFlag) String() string {
	return fmt.Sprintf("%s", *v)
}

var headers stringsFlag

func main() {
	flag.Var(&headers, "H", "request Headers")
	flag.Parse()
	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		return
	}
	S4setting, err := ReadConfig(*config)
	var s *sign4.Signature
	s = nil
	if err == nil {
		s = &sign4.Signature{
			AccessKey: S4setting["access_keys"],
			SecretKey: S4setting["secret_keys"],
			Region:    S4setting["region"],
			Service:   S4setting["service"],
		}
	}
	var bodyReader io.Reader
	bodyReader = nil
	if len(*reqBodyFile) != 0 {
		bodyReader, err = os.Open(*reqBodyFile)
		if err != nil {
			log.Fatal(err)
		}
	} else if len(*reqBodyString) > 0 {
		bodyReader = bytes.NewBufferString(*reqBodyString)
	}
	args := flag.Args()
	url := args[len(args)-1]
	req, _ := http.NewRequest(*reqMethod, url, bodyReader)
	req.Header.Add("date", time.Now().Format("Mon, 09 Sep 2011 23:36:00 GMT"))
	if s != nil {
		s.SignRequest(req)
	}
	if len(*debug) > 0 {
		fmt.Printf("%s %s %s\n", req.Method, req.URL.RequestURI(), req.Proto)
		fmt.Println("Host: ", req.Host)
		fmt.Println("Content-Length: ", req.ContentLength)
		PrintHeader(req.Header)
		fmt.Println("-----")
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("request failed:", err)
	}
	defer resp.Body.Close()
	if err != nil {
		log.Fatal("request error ", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if len(*debug) > 0 {
		fmt.Println(resp.Status)
		PrintHeader(resp.Header)
		fmt.Println("-----")
	}
	fmt.Println(string(body))
}

func PrintHeader(headers map[string][]string) {
	for header, values := range headers {
		fmt.Printf("%s: %s\n", header, strings.Join(values, ","))
	}
}

func ReadConfig(file string) (map[string]string, error) {
	configFile, err := os.Open(file)
	config, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	configFile.Close()
	setting := make(map[string]string)
	if err = json.Unmarshal(config, &setting); err != nil {
		return nil, err
	}
	return setting, err
}
