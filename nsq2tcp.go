package main

import (
	"flag"
	"fmt"
	"github.com/bitly/nsq/nsq"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	port             = flag.String("port", ":1234", "log server port")
	topic            = flag.String("topic", "nginx_log", "nsq topic")
	logchan          = flag.String("chan", "nginx_log", "nsq chan")
	lookupdHTTPAddrs = flag.String("lookupd-http-address", "127.0.0.1:4161", "lookupd http")
)

func main() {
	flag.Parse()
	// get nsqd list
	lookupdlist := strings.Split(*lookupdHTTPAddrs, ",")
	//exit
	termchan := make(chan os.Signal, 1)
	exitChan := make(chan int)
	signal.Notify(termchan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-termchan
		exitChan <- 1
	}()
	lreader := newlogreader()
	for _, ld_addr := range lookupdlist {
		go read_nsq(ld_addr, *topic, *logchan, lreader)
	}
	go run_server(lreader, *port)
	<-exitChan
}

type Message struct {
	*nsq.Message
	returnChannel chan *nsq.FinishedMessage
}
type logreader struct {
	logchan chan *Message
}

//read log from nsq
func read_nsq(lookupd_addr string, topic string, logchan string, lreader *logreader) {
	reader, _ := nsq.NewReader(topic, logchan)
	reader.AddAsyncHandler(lreader)
	reader.ConnectToLookupd(lookupd_addr)
}

//logreader
func newlogreader() *logreader {
	r := &logreader{make(chan *Message)}
	return r
}

//message handler
func (this *logreader) HandleMessage(m *nsq.Message, responseChannel chan *nsq.FinishedMessage) {
	this.logchan <- &Message{m, responseChannel}
}

//tcp server
func run_server(lreader *logreader, port string) {
	lp, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("Bind 1234 failed")
		return
	}
	defer lp.Close()
	for {
		fd, error := lp.Accept()
		if error != nil {
			fmt.Printf("accpet error %s", error)
		}
		go send_log(fd, lreader)
	}
}

//send log via tcp
func send_log(fd net.Conn, lreader *logreader) {
	defer fd.Close()
	var err error
	for {
		msg := <-lreader.logchan
		if msg != nil {
			_, err = fd.Write(msg.Body)
		}
		_, err = fd.Write([]byte("\n"))
		if err != nil {
			fmt.Printf("TCP connect write error")
			break
		} else {
			msg.returnChannel <- &nsq.FinishedMessage{msg.Id, 0, true}
		}
	}
	fmt.Printf("TCP closed!\n")
}
