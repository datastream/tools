package main

import (
	"flag"
	"log"
	"time"
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
	collection   = flag.String("collection", "metrics", "mongodb collection")
)

const nWorker = 10

type Message struct {
	done    chan int
	content string
}

type Work struct {
	consumer chan *Consumer
	producer chan *Producer
	message  chan *Message
	done     chan int
}

func NewWork() *Work {
	return &Work{
		consumer: make(chan *Consumer),
		producer: make(chan *Producer),
		message:  make(chan *Message),
		done:     make(chan int),
	}
}
func (this *Work) work() {
	for {
		select {
		case op := <-this.producer:
			{
				if op.session != nil {
					go op.handle(this)
				}
				go func() {
					select {
					case <-op.done:
						{
							p, err := NewProducer(*mongouri, *dbname, *collection, *user, *password)
							if err != nil {
								log.Printf("create new producer error%s", err)
								time.Sleep(time.Duration(2 * time.Second))
								p.done <- nil
							}
							this.producer <- p

						}
					}
				}()
			}
		case oc := <-this.consumer:
			{
				go oc.handle(this)
				go func() {
					select {
					case <-oc.done:
						{
							c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
							if err != nil {
								log.Printf("create new consumer error%s", err)
								time.Sleep(time.Duration(2 * time.Second))
								c.done <- nil
							}
							this.consumer <- c
						}
					}
				}()
			}
		}
	}
}
func main() {
	flag.Parse()
	work := NewWork()
	go work.work()
	for i := 0; i < nWorker; i++ {
		c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
		if err != nil {
			log.Printf("create new consumer failed%s", err)
			c.done <- nil
			time.Sleep(time.Duration(2 * time.Second))
		}
		work.consumer <- c

		p, err := NewProducer(*mongouri, *dbname, *collection, *user, *password)
		if err != nil {
			p.done <- nil
			log.Printf("create new producer failed%s", err)
			time.Sleep(time.Duration(2 * time.Second))
		}
		work.producer <- p
	}
	select {}
}
