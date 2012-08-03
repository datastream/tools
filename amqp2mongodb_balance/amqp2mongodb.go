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

type Work struct {
	c chan *Consumer
	p chan *Producer
	nc chan *Consumer
	np chan *Producer
	message chan *string
	done chan int
}
func NewWork() *Work {
	 return &Work{
		c: make(chan *Consumer),
		p: make(chan *Producer),
		nc: make(chan *Consumer),
		np: make(chan *Producer),
		message: make(chan *string),
	}
}
func (this *Work)renew() {
	for {
		select {
		case op := <- this.p:
			{
				log.Fatalf("%s", <- op.done)
				p, err := NewProducer(*mongouri, *dbname, *collection, *user, *password)
				if err != nil {
					log.Printf("%s", err)
					time.Sleep(time.Duration(2*time.Second))
					this.p <- p
				} else {
					this.np <- p
				}
			}
		case oc := <- this.c:
			{
				log.Fatalf("%s", <- oc.done)
				c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
				if err != nil {
					log.Printf("%s", err)
					time.Sleep(time.Duration(2*time.Second))
					this.c <- c
				} else {
					this.nc <- c
				}
			}
		}
	}
}
func (this *Work) work() {
	for {
		select {
		case nc := <- this.nc:
			{
				go nc.handle(this)
			}
		case np := <- this.np:
			{
				go np.handle(this)
			}
		}
	}
}
func main() {
	flag.Parse()
	work := NewWork()
	go work.work()
	go work.renew()
	for i := 0; i < nWorker; i++ {
		c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
		if err != nil {
			log.Printf("%s", err)
			time.Sleep(time.Duration(2*time.Second))
			work.c <- c
		} else {
			work.nc <- c
		}
		p, err := NewProducer(*mongouri, *dbname, *collection, *user, *password)
		if err != nil {
			log.Printf("%s", err)
			time.Sleep(time.Duration(2*time.Second))
			work.p <- p
		} else {
			work.np <- p
		}
	}
	select {
	}
}
