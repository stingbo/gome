package main
import (
	"google.golang.org/grpc"
	"log"
	"gome/api"
	"golang.org/x/net/context"
)

// main 方法实现对 gRPC 接口的请求
func main() {
	// 为了调用服务方法，我们首先创建一个 gRPC channel 和服务器交互。
	// 可以使用 DialOptions 在 grpc.Dial 中设置授权认证（如， TLS，GCE认证，JWT认证）
	conn, err := grpc.Dial("172.22.0.2:8088", grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Can't connect: 172.22.0.2:8088")
	}
	defer conn.Close()
	// 一旦 gRPC channel 建立起来，我们需要一个客户端 存根 去执行 RPC
	client := api.NewOrderClient(conn)
	// 调用简单 RPC
	resp,err := client.DoOrder(context.Background(), &api.OrderRequest{Symbol:"hello"})
	// 如果调用没有返回错误，那么我们就可以从服务器返回的第一个返回值中读到响应信息
	if err != nil {
		log.Fatalln("Do Format error:" + err.Error())
	}
	log.Println(resp.Message)
}
