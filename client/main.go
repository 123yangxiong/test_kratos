package main

import (
	"context"
	"crypto/tls"
	"github.com/go-chassis/sc-client"
	"github.com/go-kratos/kratos/contrib/registry/servicecomb/v2"
	"log"
	"time"

	"github.com/go-kratos/examples/helloworld/helloworld"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	srcgrpc "google.golang.org/grpc"
)

func main() {

	c, err := sc.NewClient(sc.Options{
		// EndPoints填写servicecomb中service center的地址
		Endpoints: []string{"139.9.248.107:30100"},
		EnableSSL: true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		log.Panic(err)
	}
	r := servicecomb.NewRegistry(c)

	connHTTP, err := http.NewClient(
		context.Background(),
		http.WithEndpoint("discovery:///helloworld"),
		http.WithDiscovery(r),
		http.WithBlock(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer connHTTP.Close()

	connGRPC, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///helloworld"),
		grpc.WithDiscovery(r),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer connGRPC.Close()

	for {
		callHTTP(connHTTP)
		callGRPC(connGRPC)
		time.Sleep(time.Second)
	}
}

func callGRPC(conn *srcgrpc.ClientConn) {
	client := helloworld.NewGreeterClient(conn)
	reply, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "kratos"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[grpc] SayHello %+v\n", reply)
}

func callHTTP(conn *http.Client) {
	client := helloworld.NewGreeterHTTPClient(conn)
	reply, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "kratos"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[http] SayHello %+v\n", reply)
}
