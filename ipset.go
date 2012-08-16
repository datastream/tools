package main

import (
	"bufio"
	"bytes"
	"flag"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	port     = flag.String("port", "1234", "access port")
	blockset = flag.String("blockset", "ddos", "ddos ipset")
)
var num int
var hashname string

type ipset struct {
	ip    string
	set   string
	timer *time.Timer
}
type request struct {
	ip     string
	action string
}

func main() {
	flag.Parse()
	hashname = "ddoshash"
	req := make(chan *request)
	done := make(chan int)
	expire_chan := make(chan *ipset)
	num = 0
	create_set()
	create_hash(hashname)
	add_hashlist(hashname)
	go run_command(req, expire_chan)
	go expire_ip(expire_chan)
	go run_server(req, done)
	<-done
}

func run_command(req chan *request, expire_chan chan *ipset) {
	for {
		rq := <-req
		ip_list := strings.Split(rq.ip, ",")
		var act string
		if rq.action == "add" {
			act = "-A"
		}
		if rq.action == "del" {
			act = "-D"
		}
		for i := range ip_list {
			cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", act, hashname, ip_list[i])
			var output bytes.Buffer
			cmd.Stderr = &output
			err := cmd.Run()
			if err != nil {
				log.Println("ipset ", rq.action, " error:", ip_list[i])
				continue
			}

			reg, e := regexp.Compile("set is full")
			if e == nil && reg.MatchString(output.String()) {
				log.Println("ipset ", hashname, " is full")
				num += 1
				hashname = hashname + strconv.Itoa(num)
				i--
				create_hash(hashname)
				add_hashlist(hashname)
			}
			exp := &ipset{
				ip:  ip_list[i],
				set: hashname,
			}
			exp.timer = time.AfterFunc(12*time.Hour, func() { expire_chan <- exp })
		}
	}
}

func create_set() {
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", *blockset, "setlist")
	if err := cmd.Run(); err != nil {
		log.Println("ipset create setlist failed:", err)
	}
}

func create_hash(name string) {
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", name, "iphash")
	if err := cmd.Run(); err != nil {
		log.Println("ipset create iphash ", name, " failed:", err)
	}
}

func add_hashlist(hash string) {
	_ , err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", *blockset, hash).Output()
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-A", *blockset, hash)
	if err = cmd.Run(); err != nil {
		log.Println("ipset add ", hash, " to ", *blockset, " setlist failed:", err)
	}
}

func expire_ip(expire_chan chan *ipset) {
	for {
		item := <- expire_chan
		cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", item.set, item.ip)
		err := cmd.Run()
		if err != nil {
			log.Println("ipset delete error", err)
		} else {
			log.Println("/usr/bin/sudo", "/usr/sbin/ipset", "-D", item.set, item.ip)
		}
	}
}

func run_server(req chan *request, done chan int) {
	server, err := net.Listen("tcp", "0.0.0.0:"+*port)
	if err != nil {
		log.Fatal("server bind failed:", err)
	}
	defer server.Close()
	for {
		fd, err := server.Accept()
		if err != nil {
			log.Println("accept error", err)
		}
		go handle(fd, req)
	}
	done <- 1
}
func handle(fd net.Conn, req chan *request) {
	defer fd.Close()
	rst := &request{}
	reader := bufio.NewReader(fd)
	var line []byte
	var err error
	if line, _, err = reader.ReadLine(); err == nil {
		rst.action = string(line)
	} else {
		log.Println("Read action failed", line)
		return
	}
	if bytes.Compare(line, []byte("del")) == 0 || bytes.Compare(line, []byte("add")) == 0 {
		if line, _, err = reader.ReadLine(); err == nil {
			rst.ip = string(line)
		} else {
			log.Println("Read ip failed", line)
			return
		}
		req <- rst
	} else {
		cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L")
		if out, err := cmd.Output(); err != nil {
			log.Println("ipset list failed ")
			fd.Write([]byte("list failed\n"))
		} else {
			fd.Write(out)
		}
	}
}
