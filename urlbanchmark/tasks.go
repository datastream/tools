package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type TaskPool struct {
	*Setting
	exitChannel chan int
	msgChannel  chan []byte
}

func (t *TaskPool) Run() {
	for _, task := range t.Tasks {
		go task.Run(t.exitChannel, t.msgChannel)
	}
}

func (t *TaskPool) Stop() {
	close(t.exitChannel)
}

func (t *TaskPool) Notify() {
	for {
		select {
		case <-t.exitChannel:
			return
		case msg := <-t.msgChannel:
			mail := make(map[string]string)
			if err := json.Unmarshal(msg, &mail); err != nil {
				log.Println("message error", err)
			} else {
				sendNotifyMail(mail)
			}
		}
	}
}

func sendNotifyMail(mail map[string]string) error {
	body := mail["Body"]
	from := mail["From"]
	delete(mail, "Body")
	delete(mail, "From")
	var msg string
	for k, v := range mail {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	to := strings.Split(mail["To"], ", ")
	msg += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))
	err := smtp.SendMail(
		"localhost:25",
		nil,
		from,
		to,
		[]byte(msg),
	)
	return err
}

func (t *Task) Run(exitChannel chan int, msgChannel chan []byte) {
	ticker := time.Tick(time.Minute)
	t.State = true
	for {
		select {
		case <-exitChannel:
			return
		case <-ticker:
			t.FailedCount = 0
			t.SuccessCount = 0
			state := true
			if !sendRequest(t.Url) {
				for i := 0; i < 10; i++ {
					time.Sleep(time.Second * 10)
					if sendRequest(t.Url) {
						t.SuccessCount++
					} else {
						t.FailedCount++
					}
					if t.FailedCount > t.Fails {
						state = false
					}
					if t.SuccessCount > t.Success {
						state = true
					}
				}
			}
			if t.State != state {
				hostname, _ := os.Hostname()
				mail := make(map[string]string)
				mail["From"] = t.NotifyEmailAddress
				mail["To"] = t.EmailAddresses
				mail["Subject"] = hostname + ":" + t.Name
				mail["MIME-Version"] = "1.0"
				mail["Content-Type"] = "text/plain; charset=\"utf-8\""
				mail["Content-Transfer-Encoding"] = "base64"
				if state {
					mail["Body"] = t.Url + " check ok"
				} else {
					mail["Body"] = t.Url + " check failed"
				}
				t.State = state
				msg, _ := json.Marshal(mail)
				msgChannel <- msg
			}
		}
	}
}

func sendRequest(url string) bool {
	client := http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		log.Println("connect timeout", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("unsuccessfull return %s\n", resp.Status)
		return false
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read response", err)
		return false
	}
	return true
}
