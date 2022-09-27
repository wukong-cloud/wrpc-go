package handler

import (
    "context"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/example/helloworld/protocol/pb"
)

type HelloServerImpl struct {
    pb.NopHelloServerImpl
}

func (this *HelloServerImpl)SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloResp, error) {
    fmt.Println(">>> say hello to ", req.Name)
    return &pb.HelloResp{Message: "hello " + req.Name}, nil
}
