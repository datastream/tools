package main

import (
	"bufio"
	"bytes"
	"flag"
	"github.com/garyburd/twister/server"
	"github.com/garyburd/twister/web"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	port = flag.String("port", "1234", "access port")
)

type Request struct {
	url  string
	data url.Values
}

func genrequest(requrl string, host string, postdata map[string][]string) string {
	file, err := os.Open("./" + host)
	var rstbody string
	if err != nil {
		return "server list file open Error"
	} else {
		var hostlist []string
		f := bufio.NewReader(file)
		for {
			line, err := f.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("file readline error", err)
				break
			}
			hostlist = append(hostlist, line)
		}
		cmd := make(chan *Request, 100)
		rst := make(chan string, 100)
		for i := 0; i < len(hostlist); i++ {
			req := &Request{
				url:  "http://" + strings.Trim(hostlist[i], "\n") + requrl,
				data: postdata,
			}
			go send(cmd, rst)
			cmd <- req
		}
		for i := 0; i < len(hostlist); i++ {
			rstbody += <-rst
		}
	}
	return rstbody
}

func send(req chan *Request, rst chan string) {
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

func blockApi(req *web.Request) {
	pat := strings.Split(req.URL.Path, "/")
	if pat[1] != "lvs" {
		var params map[string][]string
		if req.Method == "POST" {
			params = req.Param
		} else {
			if pat[2] == "ip" {
				params, _ = url.ParseQuery("show_type=ip&show_list=all")
			} else {
				params, _ = url.ParseQuery("show_type=variable&show_list=all")
			}
		}
		rst := genrequest("/limit_interface_"+pat[2], pat[1], params)
		w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain; charset=\"utf-8\"")
		io.WriteString(w, rst)
	} else {
		lvsblockApi(req)
	}
}

func lvsblockApi(req *web.Request) {
	w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain; charset=\"utf-8\"")
	parts := strings.Split(req.URL.Path, "/")
	if req.Method == "GET" {
		param, _ := url.ParseQuery("action_type=read")
		io.WriteString(w, sendip(parts[2], param))
	} else {
		if len(parts) > 2 {
			io.WriteString(w, sendip(parts[2], req.Param))
		} else {
			io.WriteString(w, "error")
		}
	}
	return
}

func sendip(name string, params map[string][]string) string {
	file, err := os.Open("./" + name)
	if err != nil {
		return "server list file open Error" + name
	} else {
		f := bufio.NewReader(file)
		var rst string
		for {
			line, err := f.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("file readline error")
				break
			}
			erro := make(chan error)
			if len(params["action_type"]) == 0 {
				log.Println(params["action_type"])
				return "null action\n"
			}
			act := []byte(params["action_type"][0])
			if bytes.Compare(act, []byte("add")) != 0 &&
				bytes.Compare(act, []byte("del")) != 0 &&
				bytes.Compare(act, []byte("clear")) != 0 &&
				bytes.Compare(act, []byte("list")) != 0 {
				return "wrong action\n"
			}
			if bytes.Compare(act, []byte("list")) == 0 {
				rst += line + "\n" + read_list(line, "list") + "-----------\n"
			}
			if len(params["ip"]) > 0 {
				if len(params["timeout"]) != 0 && bytes.Compare(act, []byte("add")) == 0 {
					go handle(line, params["action_type"][0],
						params["ip"][0], params["timeout"][0], erro)
				} else {
					go handle(line, params["action_type"][0],
						params["ip"][0], "", erro)
				}
				go func() {
					err := <-erro
					if err != nil {
						time.Sleep(2 * time.Second)
						if len(params["timeout"]) != 0 &&
							bytes.Compare(act, []byte("add")) == 0 {
							go handle(line, params["action_type"][0],
								params["ip"][0], params["timeout"][0], erro)
						} else {
							go handle(line, params["action_type"][0],
								params["ip"][0], "", erro)
						}
						<-erro
					}
				}()
			}
		}
		return rst
	}
	return "pushed\n"
}

func read_list(host, action string) string {
	client, e := net.Dial("tcp", host[:len(host)-1])
	if e != nil {
		log.Println("dial error:", host, " ", e)
		return "connect error: " + host
	}
	defer client.Close()
	_, e = client.Write([]byte(action + "\n"))
	if e != nil {
		log.Println("send list command error", e)
		return "list failed"
	}
	body, ee := ioutil.ReadAll(client)
	if ee != nil {
		return "read failed" + host
	}
	return string(body)
}
func handle(host, action, data, timeout string, err chan error) {
	client, e := net.Dial("tcp", host[:len(host)-1])
	if e != nil {
		log.Println("dial error:", host, " ", e)
		err <- e
		return
	}
	defer client.Close()
	_, e = client.Write([]byte(action + "\n"))
	if timeout != "" {
		_, e = client.Write([]byte("timeout\n"))
		_, e = client.Write([]byte(timeout + "\n"))
	}
	_, e = client.Write([]byte(data + "\n"))
	if e != nil {
		log.Println("send error", data)
		err <- e
	}
	err <- nil
}

func main() {
	flag.Parse()
	h := web.NewRouter().
		Register("/<:.*>", "*", web.FormHandler(8148, false, web.HandlerFunc(blockApi)))
	server.Run(":"+*port, h)
}
