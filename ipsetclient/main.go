package main

import (
	"flag"
	"github.com/bitly/go-nsq"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	bind   = flag.String("port", "0.0.0.0:1234", "http port")
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string]string
var ipSet *IPSet

// IPSet store info about ipset hash/list name
type IPSet struct {
	HashSetName string
	HashName    string
	HashList    []string
	maxsize     int
	ipreader    *nsq.Reader
	timeout     string
}

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	ipSet = &IPSet{
		HashName:    setting["hashname"],
		HashSetName: setting["hashsetname"],
		maxsize:     8,
		timeout:     setting["timeout"],
	}
	ipSet.setup()
	ddosChannel, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	ipSet.ipreader, err = nsq.NewReader(setting["topic"], ddosChannel)
	ipSet.ipreader.AddHandler(ipSet)
	lookupdlist := strings.Split(setting["lookupdaddresses"], ",")
	for _, addr := range lookupdlist {
		log.Printf("lookupd addr %s", addr)
		err := ipSet.ipreader.ConnectToLookupd(addr)
		if err != nil {
			log.Fatal(err)
		}
	}
	http.HandleFunc("/", showIP)
	err = http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func showIP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "list")
	if out, err := cmd.Output(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}
}
