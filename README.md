## Gome-High performance matchmaking engine microservice

- Using golang as computing, grpc as service, protobuf as data exchange, rabbitmq as queue, redis as cache to realize high-performance matchmaking engine micro service

- [中文文档](https://github.com/stingbo/gome/blob/master/README_cn.md)

## Requirement

Deploy Gome v1.0.2 from its published container image using [gome-docker](https://github.com/stingbo/gome-docker). The Compose setup starts Gome, Redis, and RabbitMQ together.

## Install and run v1.0.2

Browse the versioned module and its packages on [pkg.go.dev](https://pkg.go.dev/github.com/stingbo/gome@v1.0.2). Release notes and source archives are on [GitHub Releases](https://github.com/stingbo/gome/releases/tag/v1.0.2).

With Go 1.16 or later, run these commands from any directory.

```bash
go install github.com/stingbo/gome/cmd/server@v1.0.2
go install github.com/stingbo/gome/cmd/match@v1.0.2
go install github.com/stingbo/gome/cmd/notice@v1.0.2
go install github.com/stingbo/gome/cmd/client@v1.0.2
```

The binaries are installed in `GOBIN`, or in `$(go env GOPATH)/bin` if `GOBIN` is unset. Add that directory to your `PATH` to run the commands by name.

### Docker Compose quickstart

Clone the deployment repository, copy its `.env.example` to `.env`, and set `RABBITMQ_PASSWORD`. Run `docker compose pull` and `docker compose up -d --no-build --wait`, then query depth with `docker compose run --rm client`. The default image is `ghcr.io/stingbo/gome:v1.0.2`. If GHCR is unreachable from your host, see the [mirror configuration in gome-docker](https://github.com/stingbo/gome-docker#中国大陆网络拉取).

### Local quickstart

Start Redis and RabbitMQ with Docker. These ports and credentials are for a local demo. Wait for both readiness checks to pass before starting Gome. If RabbitMQ is still initializing, retry its check after a few seconds.

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

Create `config.yaml` in your working directory. If you already have Redis and RabbitMQ, use their connection details and make sure Redis has the settings listed under Delivery and recovery.

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

From the directory containing `config.yaml`, run `server` and `match btc2usdt` in separate terminals. To use a config file elsewhere, set `GOME_CONFIG` to its absolute path in both terminals. Run one matching worker per trading pair and use that same symbol in client requests.

### Submit and query an order with grpcurl

Install [grpcurl](https://github.com/fullstorydev/grpcurl) and run these commands in a third terminal. The server enables gRPC reflection, so grpcurl can discover the RPCs directly.

```bash
grpcurl -plaintext \
  -d '{"uuid":"user-1","oid":"order-1001","symbol":"btc2usdt","transaction":"BUY","price":65000,"volume":0.01}' \
  127.0.0.1:8088 Order/DoOrder

grpcurl -plaintext \
  -d '{"uuid":"user-1","oid":"order-1001","symbol":"btc2usdt"}' \
  127.0.0.1:8088 Order/GetOrderStatus
```

A successful submission means the order was queued. If the query returns `ORDER_STATUS_QUEUED`, query again after a moment. In an empty order book, this buy order becomes `ORDER_STATUS_OPEN`. Its `remainingVolumeTicks` is `"1000000"`, which represents 0.01 at `accuracy: 8`. To try another order, use a fresh `oid` in both commands. To retry a timed-out submission, reuse the original ID and parameters.

### Integrate an application

Give callers the deployed gRPC address and the [order](api/order.proto) and [depth](api/pool.proto) definitions. They can generate a client in their language and call `Order/DoOrder` to submit, `Order/DeleteOrder` to cancel, `Order/GetOrderStatus` to query status, and `Pool/GetDepth` to read the order book. For public access, expose the service through a gateway with TLS and authentication.

To process fills and cancellations, consume `notice:btc2usdt` from RabbitMQ, deduplicate by `EventID`, and ACK after committing the business update. For a terminal-only demo, run `notice btc2usdt` from the config directory. Use the demo and business consumer separately because they compete for messages on the same queue.

### Run from source

1. Start Redis and RabbitMQ and copy `config.example.yaml` to `config.yaml` in the Gome source directory. Update the connection details before starting Gome.

2. The repository includes generated Go sources. After editing the protocol, regenerate them in `api/` with `protoc --go_out=plugins=grpc,paths=source_relative:. *.proto` and `protoc-gen-go v1.4.3`.

3. Start the gRPC server：`go run ./cmd/server`

4. Start script to match consumption RabbitMQ queue：`go run ./cmd/match symbol`，symbol is the name of the trading pair，such as btc2usdt，the symbol should be the same as when called by the client

5. Start script consumption match result RabbitMQ queue：`go run ./cmd/notice symbol`.

Set `GOME_CONFIG` to override the configuration path. Set `redis.password` if Redis requires authentication. Run `go build ./...` and `go test ./...` locally. Set `GOME_TEST_REDIS_PORT` to run the order-book regression test against an isolated Redis instance.

## Delivery and recovery

The worker commits book changes, replay markers, and pending notices in one Redis transaction. It retries pending notices on reconnect and ACKs order commands only after RabbitMQ confirms their notices. Notice consumers should deduplicate using the JSON `EventID` or AMQP `message_id` and ACK after committing their own side effects. Use unique order IDs per symbol.

Use persistent storage for both services. Redis requires `appendonly yes`, `appendfsync always`, `no-appendfsync-on-rewrite no`, and `maxmemory-policy noeviction`. The worker verifies these settings using `CONFIG GET`. Both RabbitMQ queues use durable declarations and persistent messages.

When upgrading from the old non-durable queues, stop old publishers and drain both queues with the old consumers. Verify that ready and unacknowledged counts are zero before stopping the consumers and removing the empty queues. Keep the Redis data and let the new version recreate the queues with the same names.

To run the broker recovery test, set both `GOME_TEST_REDIS_PORT` and `GOME_TEST_RABBITMQ_PORT` to local ports for isolated instances. Enable the Redis settings above and set the RabbitMQ test username and password to `gome-test`. Run `go test -race ./engine`.

## Order status and retries

Call `Order.GetOrderStatus` with `symbol`, `oid`, and `uuid` to query publication, open, partially filled, filled, or cancelled states. Quantities and price are returned as integer ticks with their scale in `accuracy`. Cancellation remains asynchronous.

Retry an ambiguous submission with the original ID and parameters. A matching retry returns `duplicate=true` on success and republishes only when publication is still unconfirmed. Reusing the same ID in a symbol with different normalized parameters returns gRPC `AlreadyExists`.

Drain old order commands before upgrading the gRPC server and matching workers together. Historical order IDs remain reserved. Orders processed before the upgrade may return `NotFound` because their lifecycle was not recorded.

## Matching precision

Prices and quantities must be positive whole ticks at the configured `gome.accuracy`. For example, a scale of 8 means a tick of `0.00000001`. Off-tick values and values above `2^53 - 1` ticks per order field return `InvalidArgument`. The gRPC fields remain doubles, with order inputs in business units and depth output in scaled units.

Keep the existing scale during upgrades. Existing integral tick values can be read directly, and historical notices are replayed as stored. Reconcile any fractional tick values in open orders, depth, or queued commands before switching workers.

## Description

* gome Catalog description：
    > api, RPC interface definition directory, using ProtoBuf 3 version

    > engine, matching engine to realize logical catalog

    > grpc, gRPC service script

    > redis, redis client

    > utils, tool script directory

    > cmd/server, entry file

    > cmd/match, matchmaking script

    > cmd/notice, match result consumption script

    > cmd/client, test script, the command is as follows:

        1. Place an order:`go run ./cmd/client doOrder`
        2. Cancel order:`go run ./cmd/client deleteOrder`
        3. Get symbol depth:`go run ./cmd/client getDepth symbol transaction`
        4. View command help:`go run ./cmd/client help`

* Gome will use the symbol name as the order queue, the matching engine will consume this queue, and the matching result will be pushed to notice:+symbol as the name queue, such as notice:btc2usdt

* At present, only the data is printed when consuming the transaction result queue, and there is no other function. Users can consume this queue by themselves to implement subsequent logic, such as updating the database, notifying users, etc. Gome will add the address push function according to the configuration in the future, so use Users only need to configure the receiving address to receive the results and then process


* This project is based on my previous PHP project, replacing the queue with RabbitMQ, Redis only as a cache, and then using Golang and gRPC to achieve microservices

* The specific implementation ideas and data structure design of gome can be viewed **[Laravel-based matching service](https://github.com/stingbo/mengine)** project

* This project does not need to rely on other environments. After running the environment with docker, other projects can be connected and called, such as:
    - [PHP client](https://github.com/stingbo/php-gome), composer installation, ready to use out of the box

* In the OrderRequest of the api, uuid (user ID) and oid (order ID) should be unique to the system. In other words, the two should not be duplicated in the system. I defined the string type to facilitate the primary key to be non-incremental. Database usage

## Summary

1. If you are using a docker environment, you need to enter the gome container to perform the corresponding operation, or use Supervisor to automatically start the relevant script when the container is started

2. Enter the rabbitmq container, `docker exec -it rabbitmq bash`, view the existing queue: `rabbitmqctl list_queues`, delete the queue: `rabbitmqctl delete_queue queuename`
