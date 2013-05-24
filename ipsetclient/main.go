package main

import (
	"flag"
	"github.com/bitly/nsq/nsq"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	bind   = flag.String("port", "0.0.0.0:1234", "http port")
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string]string
var ip_set *IPSet

type IPSet struct {
	HashSetName string
	HashName    string
	HashList    []string
	index       int
	maxsize     int
	ipreader    *nsq.Reader
	ipList      map[string]string
	iplock      sync.Mutex
	timeout     string
	sync.Mutex
}

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	ip_set = &IPSet{
		HashName:    setting["hashname"],
		HashSetName: setting["hashsetname"],
		index:       0,
		maxsize:     8,
		ipList:      make(map[string]string),
		timeout:     setting["timeout"],
	}
	ip_set.setup()
	ddos_channel, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	ip_set.ipreader, err = nsq.NewReader(setting["topic"], ddos_channel)
	ip_set.ipreader.AddHandler(ip_set)
	lookupdlist := strings.Split(setting["lookupdaddresses"], ",")
	for _, addr := range lookupdlist {
		log.Printf("lookupd addr %s", addr)
		err := ip_set.ipreader.ConnectToLookupd(addr)
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
	cmd := exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "-L")
	if out, err := cmd.Output(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}
}
