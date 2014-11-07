package main

import (
	"flag"
	"log"
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
	w.Stop()
}

type WebService struct {
	ListenAddress string
}

func (q *WebService) Run() error {
	fs := memfs.New()
	q.LevelDB, err := Open("", &db.Options{
		FileSystem: fs,
	})
	if err == nil {
		fmt.Println("type=convert,stat=2,msg=leveldb error")
	}
	r := mux.NewRouter()
	s := r.PathPrefix("/api/v1").Subrouter()
	s.HandleFunc("/collect", q.Collectd).
		Methods("POST").
		Headers("Content-Type", "application/json")
	http.Handle("/", r)
	http.ListenAndServe(q.ListenAddress, nil)
	defer db.Close()
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
		msgHeader := fmt.Sprintf("==%s\n",c.Type)
		var msg string
		for i := range c.Values {
			if len(msg) >0 {
				msg += ","
			}
			//rawValue, err := q.LevelDB.Get([]byte(key),nil)
			//keyValue, err := KeyValueDecode(rawValue)
			//var nValue float64
			if err == nil {
				//nValue = c.GetMetricRate(keyValue.Value, keyValue.Timestamp, i)
				msg += fmt.Sprintf("%s=%.2f",c.DataSetTypes, c.Values[i])
			}
			if err != nil {
				fmt.Println("type=covert,state=2,msg=leveldb get error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			//t := int64(c.Timestamp)
		}
		fmt.Println(msgHeader, msg)
	}
}
