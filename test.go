package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/context"
	"gome/api"
	rpc "gome/grpc"
	"gome/util"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "doOrder",
				Usage:   "下单",
				Action:  func(c *cli.Context) error {
					fmt.Println("开始下单-----------:")
					DoOrder()
					return nil
				},
			},
			{
				Name:    "deleteOrder",
				Usage:   "撤单",
				Action:  func(c *cli.Context) error {
					fmt.Println("开始撤单-----------:")
					DeleteOrder()
					return nil
				},
			},
			{
				Name:    "getDepth",
				Usage:   "获取深度",
				Action:  func(c *cli.Context) error {
					symbol := c.Args().First()
					if symbol == "" {
						return errors.New("请输入需要查询深度的交易对名称")
					}
					fmt.Println("交易对名称: ", symbol)

					transaction := c.Args().Get(1)
					tran, err := strconv.Atoi(transaction)
					if err != nil {
						return errors.New("请输入需要查询深度的交易方向[0-buy|1-sale]")
					}
					if transaction == "" || (tran != 0 && tran != 1) {
						return errors.New("请输入需要查询深度的交易方向[0-buy|1-sale]")
					}
					fmt.Println("交易方向: ", tran)

					// 直接调用方法
					//order := api.OrderRequest{Uuid: "", Oid: "", Symbol: symbol, Transaction: api.TransactionType(tran), Price: 0, Volume: 0}
					//node := engine.NewOrderNode(order)
					//pool := engine.Pool{Node: node}
					//depths := pool.GetDepth(0, 9)

					// 通过grpc客户端调用
					depths := GetDepth(symbol, tran, 0, 5)
					fmt.Printf("depths数据----------:%#v\n", depths)

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func GetDepth (symbol string, tran int, offset int64, count int64) api.DepthResponse {
	conn := rpc.NewRpcClient()
	defer conn.Close()
	client := api.NewPoolClient(conn)

	depth := api.DepthRequest{Symbol: symbol, Transaction: api.DepthRequest_DepthType(tran), Offset: offset, Count: count}

	// 调用 RPC
	resp, err := client.GetDepth(context.Background(), &depth)

	if err != nil {
		log.Fatalln("Do Format error:" + err.Error())
	}
	log.Println(resp.Message)

	return *resp
}

func DoOrder() {
	conn := rpc.NewRpcClient()
	defer conn.Close()
	client := api.NewOrderClient(conn)

	timens := int64(time.Now().Nanosecond())
	rand.Seed(timens)
	// 0-buy,1-sale
	for i := 1; i < 2000; i++ {
		t := rand.Intn(2)
		p := FloatRound(rand.Float64(), 2)
		if p == 0 {
			p = p + 0.1
		}
		//v := rand.Intn(50)
		v := FloatRound(rand.Float64(), 2)
		if v == 0 {
			v = v + 1
		}
		//order := api.OrderRequest{Uuid: "2", Oid: strconv.Itoa(i), Symbol: "eth2usdt", Transaction: api.TransactionType(t), Price: p, Volume: float64(v)}
		order := api.OrderRequest{Uuid: "2", Oid: strconv.Itoa(i), Symbol: "eth2usdt", Transaction: api.TransactionType(t), Price: p, Volume: v}
		fmt.Printf("下单--------%#v\n", order)
		util.Info.Printf("下单------：%#v\n", order)
		resp, err := client.DoOrder(context.Background(), &order)
		if err != nil {
			log.Fatalln("Do Format error:" + err.Error())
		}
		log.Println(resp.Message)
	}
}

func DeleteOrder() {
	conn := rpc.NewRpcClient()
	defer conn.Close()
	client := api.NewOrderClient(conn)

	order := api.OrderRequest{Uuid: "2", Oid: "11", Symbol: "eth2usdt", Transaction: 0, Price: 0.5, Volume: 11}
	// 调用简单 RPC
	resp, err := client.DeleteOrder(context.Background(), &order)
	// 如果调用没有返回错误，那么我们就可以从服务器返回的第一个返回值中读到响应信息
	if err != nil {
		log.Fatalln("Do Format error:" + err.Error())
	}
	log.Println(resp.Message)
}

// 截取小数位数
func FloatRound(f float64, n int) float64 {
	format := "%." + strconv.Itoa(n) + "f"
	res, _ := strconv.ParseFloat(fmt.Sprintf(format, f), 64)
	return res
}
