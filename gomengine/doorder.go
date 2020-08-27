package main

import (
	"fmt"
	"golang.org/x/net/context"
	"gome/api"
	"gome/gomengine/util"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"time"
)

// main 方法实现对 gRPC 接口的请求
func main() {
	// 为了调用服务方法，我们首先创建一个 gRPC channel 和服务器交互。
	// 可以使用 DialOptions 在 grpc.Dial 中设置授权认证（如， TLS，GCE认证，JWT认证）
	conf := &util.MeConfig{}
	yamlFile, err := ioutil.ReadFile("config.yaml")
	yaml.Unmarshal(yamlFile, conf)
	host := conf.GRPCConf.Host
	port := conf.GRPCConf.Port
	conn, err := grpc.Dial(host+":"+port, grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Can't connect: " + host + ":" + port)
	}
	defer conn.Close()
	// 一旦 gRPC channel 建立起来，我们需要一个客户端 存根 去执行 RPC
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
		// 调用简单 RPC
		resp, err := client.DoOrder(context.Background(), &order)
		// 如果调用没有返回错误，那么我们就可以从服务器返回的第一个返回值中读到响应信息
		if err != nil {
			log.Fatalln("Do Format error:" + err.Error())
		}
		log.Println(resp.Message)
	}
}

// 截取小数位数
func FloatRound(f float64, n int) float64 {
	format := "%." + strconv.Itoa(n) + "f"
	res, _ := strconv.ParseFloat(fmt.Sprintf(format, f), 64)
	return res
}
