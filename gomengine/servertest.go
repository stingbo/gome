package main

import (
	"context"
	"fmt"
	"gome/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"strings"
)

type FormatData struct{}

func (fd *FormatData) DoFormat(ctx context.Context, in *api.Data) (out *api.Data, err error) {
	str := in.Text
	out = &api.Data{Text: strings.ToUpper(str)}
	return out, nil
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	listener := getRpcListener()

	rpcServer := grpc.NewServer()
	api.RegisterFormatDataServer(rpcServer, &FormatData{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(listener); err != nil {
		log.Fatalln("服务启动失败")
	}
}
