package main

import (
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	_ "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"gome/api"
	"gome/engine"
	rpc "gome/grpc"
	"gome/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	rpcListener := rpc.NewRpcListener()
	listener := rpcListener.Listener

	//rpcServer := grpc.NewServer()
	rpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_opentracing.StreamServerInterceptor(),
			//grpc_prometheus.StreamServerInterceptor,
			//grpc_zap.StreamServerInterceptor(ZapInterceptor()),
			grpc_zap.StreamServerInterceptor(utils.ZapFileInterceptor()),
			//grpc_auth.StreamServerInterceptor(myAuthFunction),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			//grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(utils.ZapFileInterceptor()),
			//grpc_auth.UnaryServerInterceptor(myAuthFunction),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	api.RegisterOrderServer(rpcServer, &engine.Order{})
	api.RegisterPoolServer(rpcServer, &engine.Pool{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}