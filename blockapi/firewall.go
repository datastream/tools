package main

import (
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func gen_protov1(configname string, rq *http.Request) string {
	hosts := read_config(configname)
	if len(rq.FormValue("action_type")) == 0 {
		return "null action\n"
	}
	var rst string
	req := &FirewallRequest{
		iprequest: new(IPRequest),
		rsp:       make(chan *Response),
	}
	switch strings.TrimSpace(rq.FormValue("action_type")) {
	case "add":
		{
			req.iprequest.RequestType = REQUEST_TYPE_CREATE.Enum()
			req.iprequest.Ipaddresses = get_ipaddresses(rq.FormValue("ip"))
			req.iprequest.Timeout = proto.Int32(get_timeout(rq.FormValue("timeout")))
		}
	case "del":
		{
			req.iprequest.RequestType = REQUEST_TYPE_DELTE.Enum()
			req.iprequest.Ipaddresses = get_ipaddresses(rq.FormValue("ip"))
		}
	case "clear":
		{
			req.iprequest.RequestType = REQUEST_TYPE_CLEAR.Enum()
		}
	case "list":
		{
			req.iprequest.RequestType = REQUEST_TYPE_READ.Enum()
		}
		// update not work, client donot support now
	case "update":
		{
			req.iprequest.RequestType = REQUEST_TYPE_UPDATE.Enum()
			req.iprequest.Ipaddresses = get_ipaddresses(rq.FormValue("ip"))
			req.iprequest.Timeout = proto.Int32(get_timeout(rq.FormValue("timeout")))
		}
	case "stop":
		{
			req.iprequest.RequestType = REQUEST_TYPE_STOP.Enum()
			req.iprequest.Timeout = proto.Int32(get_timeout(rq.FormValue("timeout")))
		}
	default:
		{
			return "wrong action\n"
		}
	}
	if *req.iprequest.RequestType == REQUEST_TYPE_CREATE {
		go func() {
			rq := &WhiteListRequest{
				hosts: hosts,
			}
			for i := range req.iprequest.Ipaddresses {
				rq.ip += string(req.iprequest.Ipaddresses[i]) + ","
			}
			time.Sleep(time.Second * 10)
			check_chan <- *rq
		}()
	}
	for i := range hosts {
		go sendtohost(hosts[i], req)
		if *req.iprequest.RequestType == REQUEST_TYPE_READ {
			rp := <-req.rsp
			if rp != nil {
				rst += hosts[i] + "\n" + string(rp.Msg) + "\n\n"
			}
		}
	}
	return rst
}

func get_ipaddresses(ips string) [][]byte {
	var rst [][]byte
	ip_list := strings.Split(ips, ",")
	for i := range ip_list {
		if len(ip_list[i]) > 3 {
			rst = append(rst, []byte(ip_list[i]))
		}
	}
	return rst
}

func get_timeout(t string) int32 {
	to, err := strconv.Atoi(t)
	if err != nil {
		to = 8 * 3600
	}
	return int32(to)
}

func sendtohost(host string, req *FirewallRequest) {
	fd, e := net.Dial("tcp", host)
	if e != nil {
		log.Println("dial error:", host, " ", e)
		return
	}
	defer fd.Close()
	data, err := proto.Marshal(req.iprequest)
	fd.Write(encodefixed32(uint64(len(data))))
	if _, err = fd.Write(data); err != nil {
		log.Println("write socket data error", err)
		if *req.iprequest.RequestType == REQUEST_TYPE_READ {
			req.rsp <- nil
		}
		return
	}
	if *req.iprequest.RequestType == REQUEST_TYPE_READ {
		reader := bufio.NewReader(fd)
		buf := make([]byte, 4)
		if _, err := reader.Read(buf); err != nil {
			req.rsp <- nil
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
				log.Println("read socket data failed", err, "read size:", size, "data_length:", data_length)
				req.rsp <- nil
				return
			}
			index += size
			if index == data_length {
				break
			}
		}

		rsp := &Response{}
		proto.Unmarshal(data_record, rsp)
		req.rsp <- rsp
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
