package main

import (
	"gome/gomengine/engine"
)

func main() {
	rabbitmq := engine.NewSimpleRabbitMQ("matchOrder")
	rabbitmq.ConsumeMatchOrder()
}
