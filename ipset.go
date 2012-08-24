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
const basename = "ddoshash"

var index int
var hashname string
var hashlist []string

type ipset struct {
	ip    string
	set   string
	timer *time.Timer
}
type request struct {
	ip      string
	action  string
	timeout int
}

func main() {
	flag.Parse()
	hashname = basename
	req := make(chan *request)
	done := make(chan int)
	expire_chan := make(chan *ipset)
	index = 0
	hashname = basename + strconv.Itoa(index)
	create_set(*blockset)
	create_hash(hashname)
	add_hashlist(hashname)
	for i := range hashlist {
		log.Println(hashlist[i])
	}
	go run_command(req, expire_chan)
	go expire_ip(expire_chan)
	go run_server(req, done)
	<-done
}

func run_command(req chan *request, expire_chan chan *ipset) {
	for {
		rq := <-req
		ip_list := strings.Split(rq.ip, ",")
		if rq.action == "add" {
			for i := range ip_list {
				if len(ip_list[i]) < 7 {
					continue
				}
				cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-A", hashname, ip_list[i])
				var output bytes.Buffer
				cmd.Stderr = &output
				err := cmd.Run()
				if err != nil {
					reg, e := regexp.Compile("set is full")
					if e == nil && reg.MatchString(output.String()) {
						log.Println("ipset ", hashname, " is full")
						index ++
						hashname = basename + strconv.Itoa(index)
						create_hash(hashname)
						add_hashlist(hashname)
					} else {
						//log.Println("add error:", ip_list[i])
						continue
					}
				}
				if rq.action == "add" {
					exp := &ipset{
						ip:  ip_list[i],
						set: hashname,
					}
					exp.timer = time.AfterFunc(time.Duration(rq.timeout)*time.Second,
						func() { expire_chan <- exp })
				}
			}
		}
		if rq.action == "del" {
			for i := range ip_list {
				for l := range hashlist {
					_, _ = exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", hashlist[l], ip_list[i]).Output()
				}
			}
		}
	}
}

func check_iphash() {
	output, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", *blockset).Output()
	if err == nil {
		buf := bytes.NewBuffer(output)
		i := false
		for {
			line, _ := buf.ReadString('\n')
			if line[:len(line)-1] == "Members:" {
				i = true
				continue
			}
			if line[:len(line)-1] == "Bindings:" {
				break
			}
			if i {
				hashlist = append(hashlist, line[:len(line)-1])
			}
		}
	}
}

func create_set(setname string) {
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", setname).Output()
	if err == nil {
		log.Println("setlist ", *blockset, " exist!")
		return
	}
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", setname, "setlist")
	if err = cmd.Run(); err != nil {
		log.Println("ipset create setlist failed:", err)
	}
}

func create_hash(name string) {
	check_iphash()
	full := false
	if len(hashlist) >= 8 {
		if index > 7 {
			index = 0
		}
		hashname = hashlist[index]
		name = hashname
		full = true
		log.Println("setlist full, reuse ", hashname)
	}
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", name).Output()
	if err == nil {
		log.Println("iphash ", name, " exist!")
		if full {
			_, _ = exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-F", name).Output()
		}
		return
	}
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", name, "iphash")
	hashlist = append(hashlist, name)
	if err := cmd.Run(); err != nil {
		log.Println("ipset create iphash ", name, " failed:", err)
	}
}

func add_hashlist(hash string) {
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", *blockset, hash).Output()
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-A", *blockset, hash)
	if err = cmd.Run(); err != nil {
		log.Println("ipset add ", hash, " to ", *blockset, " setlist failed:", err)
	}
}

func expire_ip(expire_chan chan *ipset) {
	for {
		item := <-expire_chan
		cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", item.set, item.ip)
		err := cmd.Run()
		if err != nil {
			log.Println("ipset delete error", err)
		} else {
			log.Println("auto expire:", "/usr/bin/sudo", "/usr/sbin/ipset", "-D", item.set, item.ip)
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
	if bytes.Compare(line, []byte("del")) == 0 {
		if line, _, err = reader.ReadLine(); err == nil {
			rst.ip = string(line)
		} else {
			log.Println("Read ip failed", line)
			return
		}
		req <- rst
	}
	if bytes.Compare(line, []byte("add")) == 0 {
		if line, _, err = reader.ReadLine(); err == nil {
			if bytes.Compare(line, []byte("timeout")) == 0 {
				if line, _, err = reader.ReadLine(); err == nil {
					rst.timeout, _ = strconv.Atoi(string(line))
				} else {
					log.Println("Read time failed", line)
					return
				}
				if line, _, err = reader.ReadLine(); err == nil {
					rst.ip = string(line)
				} else {
					log.Println("Read ip failed", line)
					return
				}
			} else {
				rst.timeout = 3600 * 8
				rst.ip = string(line)
			}
		} else {
			log.Println("Read ip failed", line)
			return
		}
		req <- rst
	}

	if bytes.Compare(line, []byte("list")) == 0 {
		cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L")
		if out, err := cmd.Output(); err != nil {
			log.Println("ipset list failed ")
			fd.Write([]byte("list failed\n"))
		} else {
			fd.Write(out)
		}
	}
	if bytes.Compare(line, []byte("clear")) == 0 {
		cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-F")
		if _, err := cmd.Output(); err != nil {
			log.Println("ipset clear failed ")
			fd.Write([]byte("list clear \n"))
		} else {
			fd.Write([]byte("all ip cleaned\n"))
		}
		hashname = basename
		index = 0
	}
}
