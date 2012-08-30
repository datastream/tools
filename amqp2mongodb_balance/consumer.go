package main

import (
	"github.com/streadway/amqp"
	"log"
	"time"
)

type Consumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tag          string
	deliveries   <-chan amqp.Delivery
	amqpURI      string
	exchange     string
	exchangeType string
	queue        string
	key          string
	done         chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queue, key, ctag string) *Consumer {
	this := &Consumer{
		conn:         nil,
		channel:      nil,
		tag:          ctag,
		amqpURI:      amqpURI,
		exchange:     exchange,
		exchangeType: exchangeType,
		queue:        queue,
		key:          key,
		done:         make(chan error),
	}
	return this
}
func (this *Consumer) connect_mq() {
	for {
		var err error
		this.conn, err = amqp.Dial(this.amqpURI)
		if err != nil {
			log.Println("Dial: ", this.amqpURI, " err:", err)
			time.Sleep(time.Second * 2)
			continue
		}
		this.channel, err = this.conn.Channel()
		if err != nil {
			log.Println("Channel: ", err)
			time.Sleep(time.Second * 2)
			continue
		}

		if err = this.channel.ExchangeDeclare(
			this.exchange,     // name of the exchange
			this.exchangeType, // type
			true,              // durable
			false,             // delete when complete
			false,             // internal
			false,             // noWait
			nil,               // arguments
		); err != nil {
			log.Println("Exchange:", this.exchange, " Declare error: ", err)
			log.Println("Channel: ", this.channel, err)
			time.Sleep(time.Second * 2)
			continue
		}

		state, err := this.channel.QueueDeclare(
			this.queue, // name of the queue
			true,       // durable
			false,      // delete when usused
			false,      // exclusive
			false,      // noWait
			nil,        // arguments
		)
		if err != nil {
			log.Println("Queue Declare: ", this.queue, " error: ", err)
			time.Sleep(time.Second * 2)
			continue
		}

		if err = this.channel.QueueBind(
			this.queue,    // name of the queue
			this.key,      // bindingKey
			this.exchange, // sourceExchange
			false,         // noWait
			nil,           // arguments
		); err != nil {
			log.Println("Queue Bind: ", err)
			time.Sleep(time.Second * 2)
			continue
		}
		this.deliveries, err = this.channel.Consume(
			this.queue, // name
			this.tag,   // consumerTag,
			false,      // noAck
			false,      // exclusive
			false,      // noLocal
			false,      // noWait
			nil,        // arguments
		)
		if err != nil {
			log.Println("Queue Consume: ", err)
			time.Sleep(time.Second * 2)
			continue
		}
		break
	}
}

func (this *Consumer) read_record(message_chan chan *Message) {
	this.connect_mq()
	go this.handle(message_chan)
	for {
		<-this.done
		this.connect_mq()
		go this.handle(message_chan)
	}
}
func (this *Consumer) handle(message_chan chan *Message) {
	for {
		select {
		case d, ok := <-this.deliveries:
			{
				if !ok {
					break
				}
				rst := string(d.Body)
				msg := &Message{
					done:    make(chan int),
					content: rst,
				}
				message_chan <- msg
				go func() {
					<-msg.done
					d.Ack(false)
				}()
			}
		}
	}
	log.Printf("handle: deliveries channel closed")
	this.done <- nil
}
