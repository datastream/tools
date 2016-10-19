package main

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	w := &WebService{
		ListenAddress: "127.0.0.1:8765",
	}
	go w.Run()
	termchan := make(chan os.Signal, 1)
	signal.Notify(termchan, syscall.SIGINT, syscall.SIGTERM)
	<-termchan
}

type WebService struct {
	ListenAddress string
	LevelDB       *leveldb.DB
}

func (q *WebService) Run() {
	var err error
	q.LevelDB, err = leveldb.OpenFile("", nil)
	if err == nil {
		fmt.Println("type=convert,stat=2,msg=leveldb error")
	}
	go q.PeriodTask()
	r := mux.NewRouter()
	s := r.PathPrefix("/api/v1").Subrouter()
	s.HandleFunc("/collect", q.Collectd).
		Methods("POST").
		Headers("Content-Type", "application/json")
	http.Handle("/", r)
	http.ListenAndServe(q.ListenAddress, nil)
	defer q.LevelDB.Close()
}

func (q *WebService) Collectd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=\"utf-8\"")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var dataset []CollectdJSON
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&dataset)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, c := range dataset {
		for i := range c.Values {
			key := c.GetMetricName(i)
			rawValue, err := q.LevelDB.Get([]byte(key), nil)
			data, err := KeyValueEncode(int64(c.Timestamp), c.Values[i])
			if err == leveldb.ErrNotFound {
				q.LevelDB.Put([]byte(key), data, nil)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			keyValue, err := KeyValueDecode(rawValue)
			var nValue float64
			if err == nil {
				nValue = c.GetMetricRate(keyValue.GetValue(), keyValue.GetTimestamp(), i)
				q.LevelDB.Put([]byte("rate"+key), []byte(fmt.Sprintf("%f", nValue)), nil)
				q.LevelDB.Put([]byte(key), data, nil)
			}
			if err != nil {
				fmt.Println("type=covert,state=2,msg=leveldb get error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
}

func (q *WebService) PeriodTask() {
	ticker := time.Tick(time.Minute)
	for {
		<-ticker
		go q.GetCPU()
		//go q.GetMem()
		//go q.GetLoad()
		//go q.GetNet()
		//go q.GetSwap()
		//go q.GetDisk()
	}
}

func (q *WebService) GetCPU() {
	iter := q.LevelDB.NewIterator(util.BytesPrefix([]byte("cpu")), nil)
	for iter.Next() {
		// Use key/value.
	}
	iter.Release()
	iter.Error()
}
