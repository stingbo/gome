package main

import (
	"gome/gomengine/RabbitMQ"
)

func main() {
	rabbitmq := RabbitMQ.NewSimpleRabbitMQ("doOrder")
	rabbitmq.ConsumeSimple()
}
