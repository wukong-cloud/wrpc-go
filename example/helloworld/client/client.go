package main

import (
    "context"
    "fmt"
    wrpcgo "github.com/wukong-cloud/wrpc-go"
    "github.com/wukong-cloud/wrpc-go/example/helloworld/protocol/pb"
    "strconv"
    "sync"
)

func main() {
    client := pb.NewHelloClient("HelloServer", wrpcgo.WithClientOptionMaxConn(2))
    var wg sync.WaitGroup
    for i := 0; i<10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            resp, err := client.SayHello(context.TODO(), &pb.HelloReq{Name: "world " + strconv.FormatInt(int64(i), 10)})
            if err != nil {
                fmt.Println(err.Error())
            } else {
                fmt.Println(i, resp.Message)
            }
        }(i)
    }
    wg.Wait()
}
