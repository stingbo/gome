package grpc

import (
	"gome/utils"
	"net"
)

var Conf *utils.MeConfig

func NewRpcListener(config utils.GRPC) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(config.Host, config.Port))
}
