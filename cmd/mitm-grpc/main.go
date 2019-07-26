package main

import (
	"net"

	"github.com/realityone/mitm-grpc/pkg/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	lis, err := net.Listen("tcp", "0.0.0.0:9000")
	if err != nil {
		panic(err)
	}

	server := server.NewServer(nil)
	logrus.Infof("Start mitm-grpc server on %+v...", lis.Addr())
	logrus.Fatal(server.Serve(lis))
}
