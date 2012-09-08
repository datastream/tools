package main

import (
	"flag"
)

var (
	uri          = flag.String("uri", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange     = flag.String("exchange", "test-exchange", "Durable AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	queue        = flag.String("queue", "test-queue", "Ephemeral AMQP queue name")
	bindingKey   = flag.String("key", "test-key", "AMQP binding key")
	consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	mongouri     = flag.String("mongouri", "mongodb://myuser:mypass@localhost:27017/mydatabase", "MONGODB RUI")
	user         = flag.String("user", "admin", "mongodb user")
	password     = flag.String("passwd", "admin", "mongodb password")
	dbname       = flag.String("db", "mydatabase", "mongodb database")
)

const nWorker = 10

type Message struct {
	done    chan int
	content string
}

func main() {
	flag.Parse()
	producer := NewMongo(*mongouri, *dbname, *user, *password)
	for i := 0; i < nWorker; i++ {
		message_chan := make(chan *Message)
		consumer := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
		go consumer.read_record(message_chan)
		go producer.insert_record(message_chan)
	}
	select {}
}
