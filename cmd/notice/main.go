package main

import (
	"errors"
	"github.com/urfave/cli/v2"
	"github.com/stingbo/gome/engine"
	"github.com/stingbo/gome/utils"
	"log"
	"os"
)

func main() {
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	engine.Configure(config)
	app := &cli.App{
		Name:  "notice",
		Usage: "消费撮合成功后的结果",
		Action: func(c *cli.Context) error {
			symbol := c.Args().Get(0)
			if symbol == "" {
				return errors.New("请输入需要消费的队列名称")
			}
			return engine.ConsumeMatchOrder(symbol)
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
