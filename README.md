## Gome-High performance matchmaking engine microservice

- Using golang as computing, grpc as service, protobuf as data exchange, rabbitmq as queue, redis as cache to realize high-performance matchmaking engine micro service

- [中文文档](https://github.com/stingbo/gome/blob/master/README_cn.md)

## Requirement

- Specific dependency information can be viewed **[docker-composer file](https://github.com/stingbo/gome-docker/blob/master/docker-compose.example.yml)**

## Usage

1. **[Use docker to deploy the operating environment with one click](https://github.com/stingbo/gome-docker)**，Enter the gome container，`docker exec -it gome bash`

2. Enter the api interface definition directory and generate a gRPC interface definition file: `cd /go/src/gome/api && protoc --go_out=plugins=grpc:. *.proto`

3. Enter the project directory, copy and modify the configuration: `cd /go/src/gome && copy config.example.yaml config.yaml`

4. Start the gRPC server：`go run main.go`

5. Start script to match consumption RabbitMQ queue：`go run match.go symbol`，symbol is the name of the trading pair，such as btc2usdt，the symbol should be the same as when called by the client

6. Start script consumption match result RabbitMQ queue：`go run match_notice.go symbol`.

## Description

* gome Catalog description：
    > api, RPC interface definition directory, using ProtoBuf 3 version

    > engine, matching engine to realize logical catalog

    > grpc, gRPC service script

    > redis, redis client

    > utils, tool script directory

    > main.go, entry file

    > match.go, matchmaking script

    > match_notice.go, match result consumption script

    > test.go, test script, the command is as follows:

        1. Place an order:`go run test.go doOrder`
        2. Cancel order:`go run test.go delOrder`
        3. Get symbol depth:`go run test.go getDepth symbol transaction`
        4. View command help:`go run test.go help`

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
