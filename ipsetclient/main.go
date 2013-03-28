package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"flag"
	"io"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var (
	port      = flag.String("port", "1234", "access port")
	blockset  = flag.String("blockset", "ddos", "ddos ipset")
	softlimit = flag.Int("softlimit", 14, "softlimit(MB)")
	hardlimit = flag.Int("hardlimit", 30, "hardlimit(MB)")
)

const basename = "ddoshash"

var indexlock *sync.Mutex
var index int

var hashname string
var hashlist []string

var currentspeed uint64

type ipset struct {
	ip    string
	set   string
	timer *time.Timer
}

func main() {
	flag.Parse()
	indexlock = new(sync.Mutex)
	hashname = basename
	req := make(chan *Request)
	done := make(chan int)
	expire_chan := make(chan *ipset)
	sleep_chan := make(chan int32)
	speed_chan := make(chan uint64)
	index = 0
	hashname = basename + strconv.Itoa(index)

	create_set(*blockset)
	create_hash(hashname)
	add_hashlist(hashname)
	for i := range hashlist {
		log.Println(hashlist[i])
	}
	go func() {
		currentspeed = <-speed_chan
	}()
	go run_command(req, expire_chan, sleep_chan)
	go read_speed(speed_chan, sleep_chan)
	go expire_ip(expire_chan, sleep_chan)
	go run_server(req, done)
	<-done
}

func run_command(req chan *Request, expire_chan chan *ipset, sleep_chan chan int32) {
	for {
		rq := <-req
		if rq == nil {
			continue
		}
		go func() {
			if *rq.iprequest.RequestType == REQUEST_TYPE_CREATE {
				for _, addr := range rq.iprequest.Ipaddresses {
					if len(addr) < 7 {
						continue
					}
					cmd := exec.Command("/usr/bin/sudo",
						"/usr/sbin/ipset", "-A",
						hashname, string(addr))
					var output bytes.Buffer
					cmd.Stderr = &output
					err := cmd.Run()
					if err != nil {
						reg, e := regexp.
							Compile("set is full")
						if e == nil &&
							reg.MatchString(
								output.String()) {
							log.Println("ipset ",
								hashname,
								" is full")
							indexlock.Lock()
							index++
							hashname = basename +
								strconv.Itoa(
									index)
							create_hash(hashname)
							indexlock.Unlock()
							add_hashlist(hashname)
						} else {
							continue
						}
					}
					exp := &ipset{
						ip:  string(addr),
						set: hashname,
					}
					exp.timer = time.AfterFunc(
						time.Duration(
							*rq.iprequest.Timeout)*
							time.Second,
						func() { expire_chan <- exp })
				}
			}

			if *rq.iprequest.RequestType == REQUEST_TYPE_DELTE {
				for _, addr := range rq.iprequest.Ipaddresses {
					for l := range hashlist {
						exec.Command("/usr/bin/sudo",
							"/usr/sbin/ipset",
							"-D", hashlist[l],
							string(addr)).Output()
					}
				}
			}

			if *rq.iprequest.RequestType == REQUEST_TYPE_UPDATE {
				for _, addr := range rq.iprequest.Ipaddresses {
					for l := range hashlist {
						exec.Command("/usr/bin/sudo",
							"/usr/sbin/ipset",
							"-D", hashlist[l],
							string(addr)).Output()
					}
				}
				rq.iprequest.RequestType =
					REQUEST_TYPE_CREATE.Enum()
				req <- rq
			}
			if *rq.iprequest.RequestType == REQUEST_TYPE_READ {
				cmd := exec.Command("/usr/bin/sudo",
					"/usr/sbin/ipset", "-L")
				response := &Response{}
				if out, err := cmd.Output(); err != nil {
					log.Println("ipset list failed ")
					response.StatCode =
						STATES_CODE_ERR.Enum()
					response.Msg = []byte("failed to list")
				} else {
					response.StatCode =
						STATES_CODE_OK.Enum()
					response.Msg = out
				}
				rq.rsp <- response
			}

			if *rq.iprequest.RequestType == REQUEST_TYPE_CLEAR {
				for i := range hashlist {
					cmd := exec.Command("/usr/bin/sudo",
						"/usr/sbin/ipset", "-F",
						hashlist[i])
					if _, err := cmd.Output(); err != nil {
						log.Println(
							"ipset clear failed")
					} else {
						log.Println("ipset cleared")
					}
				}
				hashname = basename
				indexlock.Lock()
				index = 0
				indexlock.Unlock()
			}
			if *rq.iprequest.RequestType == REQUEST_TYPE_STOP {
				sleep_chan <- *rq.iprequest.Timeout
			}
		}()
	}
}

func run_server(req chan *Request, done chan int) {
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

type Request struct {
	iprequest *IPRequest
	rsp       chan *Response
}

func handle(fd net.Conn, req chan *Request) {
	defer fd.Close()
	reader := bufio.NewReaderSize(fd, 1024*20)
	buf := make([]byte, 4)
	if _, err := reader.Read(buf); err != nil {
		return
	}
	data_length := int(decodefixed32(buf))
	data_record := make([]byte, data_length)
	var index = 0
	for {
		var size int
		var err error
		if size, err = reader.Read(data_record[index:]); err != nil {
			if err == io.EOF {
				break
			}
			log.Println("read socket data failed", err,
				"read size:", size, "data_length:", data_length)
			break
		}
		index += size
		if index == data_length {
			break
		}
	}
	request := &Request{
		iprequest: new(IPRequest),
		rsp:       make(chan *Response),
	}
	err := proto.Unmarshal(data_record, request.iprequest)
	if err != nil {
		log.Println("Unmarshal failed:", err)
		return
	}
	req <- request
	if *request.iprequest.RequestType == REQUEST_TYPE_READ {
		response := <-request.rsp
		data, err := proto.Marshal(response)
		fd.Write(encodefixed32(uint64(len(data))))
		if _, err = fd.Write(data); err != nil {
			log.Println("write socket data error", err)
			return
		}
	}
}
func encodefixed32(x uint64) []byte {
	var p []byte
	p = append(p,
		uint8(x),
		uint8(x>>8),
		uint8(x>>16),
		uint8(x>>24))
	return p
}
func decodefixed32(num []byte) (x uint64) {
	x = uint64(num[0])
	x |= uint64(num[1]) << 8
	x |= uint64(num[2]) << 16
	x |= uint64(num[3]) << 24
	return
}
