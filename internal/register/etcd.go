package register

import (
    "context"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/util/logx"
    "go.etcd.io/etcd/client/v3"
    "strings"
    "time"
)

type EtcdRegister struct {
    client *clientv3.Client
    leaseId clientv3.LeaseID
}

func NewEtcdRegister(endpoint string) Register {
   points := strings.Split(endpoint, ";")
   endpoints := make([]string, 0)
   for _, point := range points {
       point = strings.TrimSpace(point)
       if point == "" {
           continue
       }
       endpoints = append(endpoints, point)
   }
   if len(endpoints) == 0 {
       return nil
   }
   cli, err := clientv3.New(clientv3.Config{
       Endpoints: endpoints,
       DialTimeout: 6 * time.Second,
   })
   if err != nil {
       logx.Log(logx.Kv("message", "new etcd register failed"), logx.Kv("error", err))
       return nil
   }
   lease, err := cli.Grant(context.TODO(), 30)
   if err != nil {
       logx.Log(logx.Kv("message", "new etcd lease failed"), logx.Kv("error", err))
       return nil
   }
   reg := &EtcdRegister{client: cli, leaseId: lease.ID}
   return reg
}

func (cli *EtcdRegister)Register(target Target) error {
    key := fmt.Sprintf("%s_%s_%s", target.Name, target.IP, target.Port)
    val := fmt.Sprintf("%s:%s", target.IP, target.Port)
    _, err := cli.client.Put(context.TODO(), key, val, clientv3.WithLease(cli.leaseId))
    if err != nil {
        logx.Log(logx.Kv("message", "Register failed"), logx.Kv("server", target.Name), logx.Kv("ip", target.IP), logx.Kv("port", target.Port), logx.Kv("error", err))
        return err
    }
    return nil
}

func (cli *EtcdRegister)UnRegister(target Target) error {
    key := fmt.Sprintf("%s_%s_%s", target.Name, target.IP, target.Port)
    _, err := cli.client.Delete(context.TODO(), key, clientv3.WithLease(cli.leaseId))
    return err
}

func (cli *EtcdRegister)KeepAlive(target Target) error {
    _, err := cli.client.KeepAliveOnce(context.TODO(), cli.leaseId)
    return err
}
