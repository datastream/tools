package main

import (
	"github.com/streadway/amqp"
	"labix.org/v2/mgo"
	"flag"
	"log"
	"strings"
	"regexp"
	"strconv"
)

var (
	uri          = flag.String("uri", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange     = flag.String("exchange", "test-exchange", "Durable AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	queue        = flag.String("queue", "test-queue", "Ephemeral AMQP queue name")
	bindingKey   = flag.String("key", "test-key", "AMQP binding key")
	consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	mongouri      = flag.String("mongouri", "mongodb://myuser:mypass@localhost:27017/mydatabase", "MONGODB RUI")
	user         = flag.String("user", "admin", "mongodb user")
	password     = flag.String("passwd", "admin", "mongodb password")
	dbname       = flag.String("db", "mydatabase", "mongodb database")
	collection   = flag.String("collection", "metrics", "mongodb collection")
)

func init() {
	flag.Parse()
}

type Metric struct {
	Retention string
	App string
	Name string
	Colo string
	Hostname string
	Value float64
	Timestamp int64
}

func NewMetric(s string) *Metric {
	this := new(Metric)
	splitstring := strings.Split(s," ")
	if len(splitstring) == 3 {
		splitname := strings.Split(splitstring[0],".")
		this.Hostname = splitname[len(splitname) -1]
		this.Colo = splitname[len(splitname) - 2]
		var p int
		if rst, _ :=regexp.MatchString("(1sec|10sec|1min|5min)",splitname[0]); rst {
			this.Retention = splitname[0]
			this.App = splitname[1]
			p = 2
		} else {
			this.App = splitname[0]
			p = 1
		}

		for i := p; i < len(splitname) -2 ; i ++ {
			if len(this.Name) > 0 {
				this.Name += "."
			}
			this.Name += splitname[i]
		}
		this.Value, _ = strconv.ParseFloat(splitstring[1], 64)
		this.Timestamp, _ = strconv.ParseInt(splitstring[2], 10, 64)
		return this
	} else {
		log.Fatal("metric not recognized: ", s)
	}
	return nil
}

func main() {
	c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
	go c.handle()
	if err != nil {
		log.Fatalf("%s", err)
	}
	m, err := NewMongo(*mongouri, *dbname, *collection, *user, *password)
	if err != nil {
		log.Fatalf("%s", err)
	}
	go m.handle(c.body_chan)

	go func() {
		for {
			select {
			case <- c.done:
				{
					c.channel.Close()
					c.conn.Close()
					c = nil
					c, err = NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
					go c.handle()
					if err != nil {
						log.Fatalf("%s", err)
					}

				}
			case <- m.done:
				{
					m, err = NewMongo(*mongouri, *dbname, *collection, *user, *password)
					if err != nil {
						log.Fatalf("%s", err)
					}
					go m.handle(c.body_chan)
				}
			}
		}
	}()
	select {}
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	body_chan chan string
	deliveries <-chan amqp.Delivery
	done    chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queue, key, ctag string) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		body_chan: make(chan string),
		done:    make(chan error),
	}

	var err error

	log.Printf("dialing %s", amqpURI)
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("Dial: %s", err)
		return nil, err
	}

	log.Printf("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		log.Fatalf("Channel: %s", err)
		return nil, err
	}

	log.Printf("got Channel, declaring Exchange (%s)", exchange)
	if err = c.channel.ExchangeDeclare(
		exchange,          // name of the exchange
		exchangeType,      // type
		amqp.UntilDeleted, // lifetime = durable
		false,             // internal
		false,             // noWait
		nil,               // arguments
	); err != nil {
		log.Fatalf("Exchange Declare: %s", err)
		return nil, err
	}

	log.Printf("declared Exchange, declaring Queue (%s)", queue)
	state, err := c.channel.QueueDeclare(
		queue,            // name of the queue
		amqp.UntilDeleted, // lifetime = auto-delete
		false,            // exclusive
		false,            // noWait
		nil,              // arguments
	)
	if err != nil {
		log.Fatalf("Queue Declare: %s", err)
		return nil, err
	}

	log.Printf("declared Queue (%d messages, %d consumers), binding to Exchange (key '%s')",
		state.Messages, state.Consumers, key)

	if err = c.channel.QueueBind(
		queue,    // name of the queue
		key,      // bindingKey
		exchange, // sourceExchange
		false,    // noWait
		nil,      // arguments
	); err != nil {
		log.Fatalf("Queue Bind: %s", err)
		return nil, err
	}

	log.Printf("Queue bound to Exchange, starting Consume (consumer tag '%s')", c.tag)
	c.deliveries, err = c.channel.Consume(
		queue, // name
		c.tag, // consumerTag,
		false, // noAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Queue Consume: %s", err)
		return nil, err
	}
	return c, nil
}

func (this *Consumer)handle() {
	for d := range this.deliveries {
		/* log.Printf(
		 "got %dB delivery: [%v] %s",
		 len(d.Body),
		 d.DeliveryTag,
		 d.Body,
		 )
		 */
		this.body_chan <- string(d.Body)
		d.Ack(false)
	}
	log.Printf("handle: deliveries channel closed")
	this.done <- nil
}

type Mongo struct {
	session *mgo.Session
	collection *mgo.Collection
	done chan error
}

func NewMongo(mongouri, dbname, collection, user, password string) (m *Mongo, err error) {
	m = new(Mongo)
	m.session, err =  mgo.Dial(mongouri)
	if err != nil {
		return
	}
	db := m.session.DB(dbname)
	err = db.Login(user,password)
	if err != nil {
		return
	}
	m.collection = db.C(collection)
	return
}

func (this *Mongo)handle(body_ch chan string) {
	for {
		body := <- body_ch
		metrics := strings.Split(strings.TrimSpace(body),"\n")
		for i := range metrics {
			err := this.collection.Insert(NewMetric(metrics[i]))
			if err != nil {
				log.Fatal("mongodb insert failed", body)
				this.done <- err
				this.session.Close()
				this.session = nil
				body_ch <- body
				break
			}
		}
		if this.session == nil {
			break
		}
	}
}
