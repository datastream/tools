package main

import (
	"github.com/streadway/amqp"
	"log"
)

type Consumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	tag        string
	deliveries <-chan amqp.Delivery
	done       chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queue, key, ctag string) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	var err error

	log.Printf("dialing %s", amqpURI)
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		log.Printf("Dial: %s", err)
		return nil, err
	}

	log.Printf("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		log.Printf("Channel: %s", err)
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
		log.Printf("Exchange Declare: %s", err)
		return nil, err
	}

	log.Printf("declared Exchange, declaring Queue (%s)", queue)
	state, err := c.channel.QueueDeclare(
		queue,             // name of the queue
		amqp.UntilDeleted, // lifetime = auto-delete
		false,             // exclusive
		false,             // noWait
		nil,               // arguments
	)
	if err != nil {
		log.Printf("Queue Declare: %s", err)
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
		log.Printf("Queue Bind: %s", err)
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
		log.Printf("Queue Consume: %s", err)
		return nil, err
	}
	return c, nil
}

func (this *Consumer) handle(work *Work) {
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
				go func() {
					work.message <- msg
					<-msg.done
					d.Ack(false)
				}()
			}
		}
	}
	log.Printf("handle: deliveries channel closed")
	this.conn.Close()
	this.channel.Close()
	this.conn = nil
	this.done <- nil
	work.consumer <- this
}
