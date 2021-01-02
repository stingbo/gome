package main

import (
	"errors"
	"github.com/urfave/cli/v2"
	"gome/gomengine/engine"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name: "notice",
		Usage: "消费撮合成功后的结果",
		Action: func(c *cli.Context) error {
			symbol := c.Args().Get(0)
			if symbol == "" {
				return errors.New("请输入需要消费的队列名称")
			}
			mq := engine.NewSimpleRabbitMQ("notice:"+symbol)
			mq.ConsumeMatchOrder()

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
