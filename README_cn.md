## Gome 高性能撮合引擎微服务

[![Release](https://img.shields.io/github/v/release/stingbo/gome)](https://github.com/stingbo/gome/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/stingbo/gome.svg)](https://pkg.go.dev/github.com/stingbo/gome)
[![Docker](https://img.shields.io/badge/Docker-GHCR-2496ED?logo=docker&logoColor=white)](https://github.com/stingbo/gome/pkgs/container/gome)
[![MIT License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

- 使用 Golang 做计算，gRPC 做服务，ProtoBuf 做数据交换，RabbitMQ 做队列，Redis 做缓存实现的高性能撮合引擎微服务

## 依赖

使用 [gome-docker](https://github.com/stingbo/gome-docker) 拉取 Gome v1.0.2 的发布镜像，并通过 Compose 一起启动 Gome、Redis 和 RabbitMQ。

## 安装并运行 v1.0.2

在 [pkg.go.dev 版本页](https://pkg.go.dev/github.com/stingbo/gome@v1.0.2) 查看模块和包文档，在 [GitHub Release](https://github.com/stingbo/gome/releases/tag/v1.0.2) 查看发布说明和源码归档。

使用 Go 1.16 或更新版本，可在任意目录执行以下命令。

```bash
go install github.com/stingbo/gome/cmd/server@v1.0.2
go install github.com/stingbo/gome/cmd/match@v1.0.2
go install github.com/stingbo/gome/cmd/notice@v1.0.2
go install github.com/stingbo/gome/cmd/client@v1.0.2
```

可执行文件安装到 `GOBIN`。未设置时，默认安装到 `$(go env GOPATH)/bin`。将该目录加入 `PATH` 后，可以直接用命令名启动。

### 用 Docker Compose 快速部署

克隆部署仓库，将 `.env.example` 复制为 `.env` 并填写 `RABBITMQ_PASSWORD`。运行 `docker compose pull` 和 `docker compose up -d --no-build --wait`，再用 `docker compose run --rm client` 查询深度。默认镜像为 `ghcr.io/stingbo/gome:v1.0.2`。部署主机无法访问 GHCR 时，参照 [gome-docker 的镜像加速配置](https://github.com/stingbo/gome-docker#中国大陆网络拉取)。

### 本机快速开始

用 Docker 启动 Redis 和 RabbitMQ。以下端口和密码用于本机演示。等两个就绪检查通过后再启动 Gome。如果 RabbitMQ 仍在初始化，稍等几秒再执行检查。

```bash
docker run -d --name gome-redis -p 127.0.0.1:16379:6379 \
  -v gome-redis-data:/data redis:7.4-alpine \
  redis-server --appendonly yes --appendfsync always \
  --no-appendfsync-on-rewrite no --maxmemory-policy noeviction

docker run -d --name gome-rabbitmq -p 127.0.0.1:15673:5672 \
  -v gome-rabbitmq-data:/var/lib/rabbitmq --hostname gome-rabbitmq \
  -e RABBITMQ_DEFAULT_USER=gome -e RABBITMQ_DEFAULT_PASS=gome-dev rabbitmq:4

docker exec gome-redis redis-cli ping
docker exec gome-rabbitmq rabbitmq-diagnostics -q check_port_connectivity
```

在工作目录创建 `config.yaml`，内容如下。已有 Redis 和 RabbitMQ 时，可跳过容器启动，并填写实际连接信息。Redis 需要开启下文「消息恢复与升级」列出的持久化设置。

```yaml
grpc:
  host: 127.0.0.1
  port: 8088
redis:
  host: 127.0.0.1
  port: 16379
  password: ""
rabbitmq:
  host: 127.0.0.1
  port: 15673
  username: gome
  password: gome-dev
gome:
  accuracy: 8
```

在配置文件所在目录打开两个终端，分别运行 `server` 和 `match btc2usdt`。配置文件放在其他目录时，在两个终端都将 `GOME_CONFIG` 设为该文件的绝对路径。每个交易对运行一个撮合进程，客户端请求的 `symbol` 要和撮合进程一致。

### 用 grpcurl 下单和查单

安装 [grpcurl](https://github.com/fullstorydev/grpcurl) 后，在第三个终端运行下面两条命令。服务端启用了 gRPC 反射，命令可以直接发现接口。

```bash
grpcurl -plaintext \
  -d '{"uuid":"user-1","oid":"order-1001","symbol":"btc2usdt","transaction":"BUY","price":65000,"volume":0.01}' \
  127.0.0.1:8088 Order/DoOrder

grpcurl -plaintext \
  -d '{"uuid":"user-1","oid":"order-1001","symbol":"btc2usdt"}' \
  127.0.0.1:8088 Order/GetOrderStatus
```

下单成功表示请求已入队。查询若返回 `ORDER_STATUS_QUEUED`，稍后再查一次。空订单簿中的这笔买单会变成 `ORDER_STATUS_OPEN`，剩余数量字段 `remainingVolumeTicks` 为 `"1000000"`，按 `accuracy: 8` 换算就是 0.01。再次试一笔新订单时，同时替换两条命令里的 `oid`。如果是下单超时后重试，保留原订单 ID 和原参数。

### 业务系统接入

把部署后的 gRPC 地址和 [订单接口](api/order.proto)、[深度接口](api/pool.proto) 提供给调用方。调用方可用对应语言生成客户端，调用 `Order/DoOrder` 下单、`Order/DeleteOrder` 撤单、`Order/GetOrderStatus` 查单和 `Pool/GetDepth` 查盘口。跨公网接入时，通过带 TLS 和鉴权的网关提供入口。

需要自动处理成交和撤单回报时，让业务消费者读取 `notice:btc2usdt`，按 `EventID` 去重，业务写入成功后再 ACK。只想在终端查看回报时，可在配置目录运行 `notice btc2usdt`。业务消费者和演示进程应分开使用，以免争抢同一队列的消息。

### 从源码运行

1. 启动 Redis 和 RabbitMQ，在 Gome 源码目录将 `config.example.yaml` 复制为 `config.yaml`，并填写连接信息。

2. 仓库已包含生成的 Go 代码。修改协议后，在 `api/` 下使用 `protoc-gen-go v1.4.3` 运行 `protoc --go_out=plugins=grpc,paths=source_relative:. *.proto` 重新生成。

3. 启动 gRPC 服务端：`go run ./cmd/server`

4. 启动脚本撮合消费 RabbitMQ 队列：`go run ./cmd/match symbol`，symbol 为交易对名称，如 btc2usdt，symbol 要与客户端调用时保持一致

5. 启动脚本消费撮合结果 RabbitMQ 队列：`go run ./cmd/notice symbol`.

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

## 订单状态与下单重试

- `Order.GetOrderStatus` 使用 `symbol`、`oid`、`uuid` 查询。状态包括发布待确认（`PENDING`）、已入队（`QUEUED`）、已挂单（`OPEN`）、部分成交（`PARTIALLY_FILLED`）、完全成交（`FILLED`）和已撤销（`CANCELLED`）。价格、原始数量、成交量、剩余量和撤销量用整数 tick 返回，`accuracy` 是小数位数。撤单仍是异步操作，只有订单簿实际撤销后，状态才变为 `CANCELLED`。
- 在同一交易对内，用相同订单 ID、用户、方向、价格和数量重试 `DoOrder`，成功响应包含 `duplicate=true`。同一 ID 对应不同参数时返回 gRPC `AlreadyExists`。现有下单和撤单的 RPC 路径及字段编号保留。
- 下单超时或返回 `Unavailable` 时，使用原订单 ID 和原参数重试。如果状态仍是 `PENDING`，重试会再次发布，撮合端会去重。如果已经入队或撮合完成，重试直接返回已有结果。首次发布前服务中断留下的 `PENDING` 记录需要客户端重试。
- 升级时先停止旧版生产者并排空订单指令，再切换新版 gRPC 服务和撮合进程。升级前已处理的订单没有完整的原始请求和状态历史，因此查询可能返回 `NotFound`。已使用的历史订单 ID 仍受保护，重新提交会返回 `AlreadyExists`。

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
