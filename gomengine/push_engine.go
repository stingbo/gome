package main

import (
	memq "gome/gomengine/rabbitmq"
)

func main() {
	rabbitmq := memq.NewSimpleRabbitMQ("doOrder")
	rabbitmq.ConsumeSimple()
}
