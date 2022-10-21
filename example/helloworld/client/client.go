package main

import (
    "context"
    "fmt"
    wrpcgo "github.com/wukong-cloud/wrpc-go"
    "github.com/wukong-cloud/wrpc-go/example/helloworld/protocol/pb"
    "github.com/wukong-cloud/wrpc-go/util/logx"
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
            resps, errs := client.BroadcastSayHello(ctx, &pb.HelloReq{Name: "world " + strconv.FormatInt(int64(i), 10)})
            if len(errs) > 0 {
                logx.Log(logx.Kv("errss", errs))
            } else {
                logx.Log(logx.Kv("resps", resps))
            }
        }(i)
    }
    wg.Wait()
}
