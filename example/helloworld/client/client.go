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
            meta := map[string]string{}
            meta[wrpcgo.ConsistentHashKey] = fmt.Sprint(i)
            ctx := wrpcgo.NewOutgoingContext(context.TODO(), meta)
            resp, err := client.SayHello(ctx, &pb.HelloReq{Name: "world " + strconv.FormatInt(int64(i), 10)})
            if err != nil {
                fmt.Println(err.Error())
            } else {
                fmt.Println(i, resp.Message)
            }
        }(i)
    }
    wg.Wait()
}
