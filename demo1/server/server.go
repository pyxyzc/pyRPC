package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func main() {
	service := new(ServiceA)
	rpc.Register(service) // 注册RPC服务
	rpc.HandleHTTP()      // 基于HTTP协议
	l, e := net.Listen("tcp", ":9091")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
