package main

import (
	"gome/gomengine/engine"
)

func main() {
	rabbitmq := engine.NewSimpleRabbitMQ("doOrder")
	rabbitmq.ConsumeNewOrder()
}
