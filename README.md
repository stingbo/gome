## Gome 高性能撮合引擎微服务

- 使用 Golang 做计算，gRPC 做服务，ProtoBuf 做数据交换，RabbitMQ 做队列，Redis 做缓存实现的高性能撮合引擎微服务

## 依赖

- 具体依赖信息可以查看 **[docker-composer 文件](https://github.com/stingbo/gome-docker/blob/master/docker-compose.example.yml)**

## 快速开始

1. **[使用 docker 一键部署运行环境](https://github.com/stingbo/gome-docker)**，进入 gome 容器，`docker exec -it gome bash`

2. 进入 api 接口定义目录，生成 gRPC 接口定义文件: `cd /go/src/gome/api && protoc --go_out=plugins=grpc:. *.proto`

3. 进入项目目录，复制并修改配置: `cd /go/src/gome && copy config.example.yaml config.yaml`

4. 启动 gRPC 服务端：`go run main.go`

5. 启动脚本撮合消费 RabbitMQ 队列：`go run match.go symbol`，symbol 为交易对名称，如 btc2usdt，symbol 要与客户端调用时保持一致

6. 启动脚本消费撮合结果 RabbitMQ 队列：`go run match_notice.go symbol`.

## 说明

* gome 目录说明：
    > api，RPC 接口定义目录，使用 ProtoBuf 3 版本

    > engine，撮合引擎实现逻辑目录

    > grpc，gRPC服务脚本

    > redis，redis客户端

    > utils，工具脚本目录

    > main.go 入口文件

    > match.go 撮合脚本

    > match_notice.go 撮合结果消费脚本

    > test.go 测试脚本，命令如下：

        1. 下单:`go run test.go doOrder`
        2. 撤单:`go run test.go delOrder`
        3. 获取交易对深度:`go run test.go getDepth symbol transaction`
        4. 查看命令帮助:`go run test.go help`

* gome 会使用 symbol 名作为下单队列，撮合引擎会消耗此队列，撮合成交结果会 push 到 notice:+symbol 作为名称的队列，如 notice:btc2usdt

* 目前消费消费成交结果队列时只打印了数据，没有其它功能，使用者可以自行消费此队列，实现后续逻辑，如更新数据库，通知用户等，gome 后续会增加根据配置的地址推送功能，这样使用者只需要配置接收地址即可接收结果然后处理

* 本项目是在我之前的 PHP 项目基础上，把队列替换为 RabbitMQ，Redis 只作为缓存，再使用 Golang 与 gRPC 实现微服务化

* gome 的具体实现思想与数据结构设计可以查看 **[基于Laravel的撮合服务](https://github.com/stingbo/mengine)** 项目

* 本项目不用依赖其他环境，使用 docker 跑起环境后，其他项目对接调用即可，如：
    - [PHP 客户端](https://github.com/stingbo/php-gome)，composer 安装，开箱即用

* api 的 OrderRequest 里，uuid(用户标识)与 oid(订单标识)应该具有系统唯一性，话说回来，这两者在系统里也不应该重复，我定义的是 string 类型，方便主键是非自增整型数据库使用

## 总结

1. 如果使用的 docker 环境，需要进入 gome 容器执行对应的操作，或者使用 Supervisor 在启动容器时自动启动相关脚本

1. 进入 rabbitmq 容器，`docker exec -it rabbitmq bash`，查看现有队列：`rabbitmqctl list_queues`，删除队列：`rabbitmqctl delete_queue queuename`
