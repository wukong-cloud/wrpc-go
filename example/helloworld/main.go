package main

import (
    wrpcgo "github.com/wukong-cloud/wrpc-go"
    "github.com/wukong-cloud/wrpc-go/example/helloworld/handler"
    "github.com/wukong-cloud/wrpc-go/example/helloworld/protocol/pb"
    "github.com/wukong-cloud/wrpc-go/util/logx"
)

func main() {
    server := pb.NewHelloServer("HelloServer", &handler.HelloServerImpl{}, wrpcgo.WithServerOptionAddr(":9092"))
    app := wrpcgo.NewApp(wrpcgo.WithServer(server))
    logx.Log("start service")
    if err := app.Run(); err != nil {
        logx.Log("service start failed", err.Error())
    }
}
