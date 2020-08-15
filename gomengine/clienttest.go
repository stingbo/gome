package main
import (
	"google.golang.org/grpc"
	"log"
	"gome/api"
	"golang.org/x/net/context"
)
// 定义请求地址
const (
	//ADDRESS string = "gome:8088" //容器名称，容器间通讯使用
	ADDRESS string = "172.22.0.2:8088" //容器ip，在宿主机上使用
)
// main 方法实现对 gRPC 接口的请求
func main() {
	// 为了调用服务方法，我们首先创建一个 gRPC channel 和服务器交互。
	// 可以使用 DialOptions 在 grpc.Dial 中设置授权认证（如， TLS，GCE认证，JWT认证）
	conn, err := grpc.Dial(ADDRESS, grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Can't connect: " + ADDRESS)
	}
	defer conn.Close()
	// 一旦 gRPC channel 建立起来，我们需要一个客户端 存根 去执行 RPC
	client := api.NewFormatDataClient(conn)
	// 调用简单 RPC
	resp,err := client.DoFormat(context.Background(), &api.Data{Text:"hello,world!"})
	// 如果调用没有返回错误，那么我们就可以从服务器返回的第一个返回值中读到响应信息
	if err != nil {
		log.Fatalln("Do Format error:" + err.Error())
	}
	log.Println(resp.Text)
}
