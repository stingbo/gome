## Golang+gRPC+ProtoBuf+RabbitMQ+Redis 撮合引擎微服务

### 快速开始

1. **[使用 docker 一键部署运行环境](https://github.com/stingbo/gome-docker)**

2. 复制并修改配置: `cd /go/src/gome/gomengine && copy config.example.yaml config.yaml`

3. 生成 gRPC 接口定义文件: `protoc --go_out=plugins=grpc:. *.proto`

4. 启动 gRPC 服务端：`go run main.go`

5. 启动 RabbitMQ 撮合消费端，匹配并计算：`go run consume_new_order.go`

6. 使用 gRPC 客户端脚本测试，下单：`go run doorder.go`，撤单：`go run delorder.go`

7. 启动 RabbitMQ 撮合结果消费端，持久化、更新数据库使用：`go run consume_match_order.go`

### 说明

* doOrder 是下单队列，撮合引擎会消耗此队列，matchOrder 是撮合成交结果队列，消耗此队列并更新数据库，持久化

* 此微服务的的具体实现思想与数据结构设计可以查看 **[基于Laravel的撮合服务](https://github.com/stingbo/mengine)** 项目

* 本项目就是在我之前的项目基础上，队列使用 RabbitMQ 中间件代替，Redis 只作缓存使用，再使用 Golang 与 gRPC 来实现微服务化

* 这样可以不用依赖其他环境，使用 docker 跑起环境后，其他项目对接使用即可，如：
    - [PHP 客户端](https://github.com/stingbo/php-gome)，composer 安装，开箱即用

### 总结

1. 正常来说，api 的 OrderRequest 里，uuid 与 oid 应该是具有全系统唯一性的标识，话说回来，这两者在系统里也不应该重复，
我定义的是 string 类型，方便主键是非自增整型数据库使用
