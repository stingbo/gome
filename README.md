## Gome-High performance matchmaking engine microservice

- Using golang as computing, grpc as service, protobuf as data exchange, rabbitmq as queue, redis as cache to realize high-performance matchmaking engine micro service

- [中文文档](https://github.com/stingbo/gome/blob/master/README_cn.md)

## Requirement

- Specific dependency information can be viewed **[docker-composer file](https://github.com/stingbo/gome-docker/blob/master/docker-compose.example.yml)**

## Usage

Install the executables with Go 1.16 or later.

```bash
go install github.com/stingbo/gome/cmd/server@v1.0.1
go install github.com/stingbo/gome/cmd/match@v1.0.1
go install github.com/stingbo/gome/cmd/notice@v1.0.1
go install github.com/stingbo/gome/cmd/client@v1.0.1
```

The commands install `server`, `match`, `notice`, and `client` in `GOBIN` (by default, `$(go env GOPATH)/bin`). To run the services, copy the example configuration from the matching source tag and configure Redis and RabbitMQ as described below.

1. **[Use docker to deploy the operating environment with one click](https://github.com/stingbo/gome-docker)**，Enter the gome container，`docker exec -it gome bash`

2. Generated Go sources are checked in. After editing a protocol, run `protoc --go_out=plugins=grpc,paths=source_relative:. *.proto` in `api/` with `protoc-gen-go v1.4.3`.

3. Enter the project directory, copy and modify the configuration: `cd /go/src/gome && cp config.example.yaml config.yaml`

4. Start the gRPC server：`go run ./cmd/server`

5. Start script to match consumption RabbitMQ queue：`go run ./cmd/match symbol`，symbol is the name of the trading pair，such as btc2usdt，the symbol should be the same as when called by the client

6. Start script consumption match result RabbitMQ queue：`go run ./cmd/notice symbol`.

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
