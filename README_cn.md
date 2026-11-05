## Gome 高性能撮合引擎微服务

- 使用 Golang 做计算，gRPC 做服务，ProtoBuf 做数据交换，RabbitMQ 做队列，Redis 做缓存实现的高性能撮合引擎微服务

## 依赖

- 具体依赖信息可以查看 **[docker-composer 文件](https://github.com/stingbo/gome-docker/blob/master/docker-compose.example.yml)**

## 快速开始

1. **[使用 docker 一键部署运行环境](https://github.com/stingbo/gome-docker)**，进入 gome 容器，`docker exec -it gome bash`

2. 仓库已包含生成的 Go 代码。修改协议后，在 `api/` 下使用 `protoc-gen-go v1.4.3` 重新运行 `protoc --go_out=plugins=grpc:. *.proto`。

3. 进入项目目录，复制并修改配置: `cd /go/src/gome && cp config.example.yaml config.yaml`

4. 启动 gRPC 服务端：`go run ./cmd/server`

5. 启动脚本撮合消费 RabbitMQ 队列：`go run ./cmd/match symbol`，symbol 为交易对名称，如 btc2usdt，symbol 要与客户端调用时保持一致

6. 启动脚本消费撮合结果 RabbitMQ 队列：`go run ./cmd/notice symbol`.

可用 `GOME_CONFIG` 指定配置文件。Redis 启用密码时请填写 `redis.password`，否则留空。构建和测试可运行 `go build ./...` 和 `go test ./...`。设置 `GOME_TEST_REDIS_PORT` 可在隔离 Redis 实例中运行订单簿回归测试。

## 消息恢复与升级

- 每个交易对只允许一个撮合消费者。Redis 在同一事务写入订单簿、去重记录及待发布回报。撮合进程重连时先补发回报，每条回报在 RabbitMQ 确认后才从 Redis 移除。下单与撤单指令在处理完成后手动 ACK。
- 回报可能重复送达。下游应将 JSON 的 `EventID` 或 AMQP 的 `message_id` 作为幂等键，在业务写入成功后再 ACK。每个交易对的订单 ID 应全程唯一。Redis 中的去重记录会保留，请把增长量纳入容量规划。
- Redis 需使用持久磁盘，并配置 `appendonly yes`、`appendfsync always`、`no-appendfsync-on-rewrite no`、`maxmemory-policy noeviction`。撮合进程启动和重连时使用 `CONFIG GET` 校验这些选项。RabbitMQ 也需使用持久磁盘，订单队列和回报队列均声明为 durable，并发布持久消息。此配置覆盖使用原数据盘的服务重启，磁盘丢失或切换到未同步副本需要独立的容灾方案。
- 从旧版本升级时，先停止旧版生产者，用旧版消费者排空每个交易对的订单与回报队列。确认 ready 和 unacked 均为零后停止旧版消费者，仅删除已确认空的旧非持久队列。新版本会以原队列名重建持久队列。升级时保留 Redis 订单簿和去重数据，并禁止旧版撮合进程继续写入。

在隔离环境中运行故障测试时，使用启用了上述持久选项的 Redis 和独立 RabbitMQ 实例。给 RabbitMQ 测试实例配置用户名和密码 `gome-test`，然后运行：

```sh
GOME_TEST_REDIS_PORT=<Redis本机端口> GOME_TEST_RABBITMQ_PORT=<RabbitMQ本机端口> go test -race ./engine
```

## 撮合精度

- 下单价格和数量使用 `gome.accuracy` 转成整数 tick。例如配置为 8 时，最小单位是 `0.00000001`。小于最小单位、无法整除为 tick 或超过单笔范围的价格和数量会返回 `InvalidArgument`，避免静默舍入。单笔上限为 `2^53 - 1` tick，保证 Redis 价格排序能区分相邻 tick。
- gRPC 字段仍为 `double`，请求继续使用业务单位，深度价格和数量继续返回放大后的数值。深度汇总在 Redis 中保持整数精度，转换为 gRPC `double` 时受该类型的表示精度限制。
- 升级时保持原有 `gome.accuracy`。整数 tick 的旧订单和深度可直接读取，已生成的历史回报按原始消息重发。如果旧版接收过不足一 tick 的值，升级前请检查并对账现存订单、深度和待消费指令。新版遇到无法精确转换的旧指令会保留待处理消息，等待对账。

## 说明

* gome 目录说明：
    > api，RPC 接口定义目录，使用 ProtoBuf 3 版本

    > engine，撮合引擎实现逻辑目录

    > grpc，gRPC服务脚本

    > redis，redis客户端

    > utils，工具脚本目录

    > cmd/server 入口文件

    > cmd/match 撮合脚本

    > cmd/notice 撮合结果消费脚本

    > cmd/client 测试脚本，命令如下：

        1. 下单:`go run ./cmd/client doOrder`
        2. 撤单:`go run ./cmd/client deleteOrder`
        3. 获取交易对深度:`go run ./cmd/client getDepth symbol transaction`
        4. 查看命令帮助:`go run ./cmd/client help`

* gome 会使用 symbol 名作为下单队列，撮合引擎会消耗此队列，撮合成交结果会 push 到 notice:+symbol 作为名称的队列，如 notice:btc2usdt

* 目前成交结果消费脚本只打印成交与撤单回报，没有其它功能，使用者可以自行消费此队列，实现后续逻辑，如更新数据库，通知用户等，gome 后续会增加根据配置的地址推送功能，这样使用者只需要配置接收地址即可接收结果然后处理

* 本项目是在我之前的 PHP 项目基础上，把队列替换为 RabbitMQ，Redis 只作为缓存，再使用 Golang 与 gRPC 实现微服务化

* gome 的具体实现思想与数据结构设计可以查看 **[基于Laravel的撮合服务](https://github.com/stingbo/mengine)** 项目

* 本项目不用依赖其他环境，使用 docker 跑起环境后，其他项目对接调用即可，如：
    - [PHP 客户端](https://github.com/stingbo/php-gome)，composer 安装，开箱即用

* api 的 OrderRequest 里，uuid(用户标识)与 oid(订单标识)应该具有系统唯一性，话说回来，这两者在系统里也不应该重复，我定义的是 string 类型，方便主键是非自增整型数据库使用

## 总结

1. 如果使用的 docker 环境，需要进入 gome 容器执行对应的操作，或者使用 Supervisor 在启动容器时自动启动相关脚本

2. 进入 rabbitmq 容器，`docker exec -it rabbitmq bash`，查看现有队列：`rabbitmqctl list_queues`，删除队列：`rabbitmqctl delete_queue queuename`
