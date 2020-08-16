#### Golang+gRPC+ProtoBuf+RabbitMQ+Redis撮合引擎

1. 使用docker部署环境，[库地址](https://github.com/stingbo/go_match_engine_docker)

2. 启动gRPC服务端，在容器里`cd /go/src/gome/gomengine && go run main.go gomerpc.go`

3. 启动RabbitMQ消费端，`cd /go/src/gome/gomengine && go run pushCache.go`

4. 使用gRPC客户端脚本测试，`cd /go/src/gome/gomengine && go run ordertest.go` 可以看到输出的交易对的内容

### 未完待续
