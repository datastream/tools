# logsend
> listen a port, when someon connect the port, read logfile and send.

## Useage
`./logsend -filename /var/log/apache/access.log`

# amqp2mongodb
> read [metrics](http://code.google.com/p/rocksteady/wiki/MetricFormat) from rabbitmq, send to mongodb

## Useage
`./amqp2mongodb -uri amqp://guest:guest@127.0.0.1:5672/ -exchange graphite -exchange-type topic -key "" -queue amqp2mongodb -mongouri localhost -user admin -passwd admin -db collectd -collection monitor`

# Update
> cleanup amqp2mongodb
