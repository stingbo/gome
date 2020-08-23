## Golang+gRPC+ProtoBuf+RabbitMQ+Redis撮合引擎微服务

### 快速开始

1. 使用docker部署环境，[库地址](https://github.com/stingbo/go_match_engine_docker)

2. 复制并修改配置 `cd /go/src/gome/gomengine && copy config.ini.example config.ini`

3. 生成gRPC接口定义文件 `protoc --go_out=plugins=grpc:. *.proto`

4. 启动gRPC服务端，`go run main.go`

5. 启动RabbitMQ消费端，`go run pushEngin.go`

6. 使用gRPC客户端脚本测试，`go run ordertest.go`

### 说明

此微服务的的具体实现逻辑可以查看[基于Laravel的撮合服务](https://github.com/stingbo/mengine)项目

本项目就是在我之前的项目基础上，把Redis队列使用RabbitMQ中间件代替，Redis只作缓存使用，再使用Golang与gRPC来实现微服务化

这样可以不用依赖其他环境，使用docker跑起环境后，其他项目对接使用就行
