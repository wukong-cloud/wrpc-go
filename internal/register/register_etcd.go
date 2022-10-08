package register

import (
    "context"
    "go.etcd.io/etcd/client/v3"
    "strings"
    "sync"
    "time"
)

type EtcdRegister struct {
    mu      sync.Mutex
    client  *clientv3.Client
    leaseId clientv3.LeaseID
}

func NewEtcdRegister(hosts string) (Register, error) {
    endpoints := strings.Split(hosts, ";")
    cli, err := clientv3.New(clientv3.Config{
        Endpoints: endpoints,
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    register := &EtcdRegister{client: cli}
    return register, nil
}

func (cli *EtcdRegister)Register(target Target) error {
    cli.mu.Lock()
    defer cli.mu.Unlock()
    if cli.leaseId == 0 {
        lease, err := cli.client.Grant(context.TODO(), 30)
        if err != nil {
            return err
        }
        cli.leaseId = lease.ID
    }
    key := target.String()
    val := target.Host()
    _, err := cli.client.Put(context.TODO(), key, val, clientv3.WithLease(cli.leaseId))
    return err
}

func (cli *EtcdRegister)UnRegister(target Target) error {
    cli.mu.Lock()
    defer cli.mu.Unlock()
    key := target.String()
    _, err := cli.client.Delete(context.TODO(), key)
    return err
}

func (cli *EtcdRegister)KeepAlive(target Target) error {
    cli.mu.Lock()
    defer cli.mu.Unlock()
    if cli.leaseId == 0 {
        lease, err := cli.client.Grant(context.TODO(), 30)
        if err != nil {
            return err
        }
        cli.leaseId = lease.ID
    }
    cli.client.KeepAliveOnce(context.TODO(), cli.leaseId)
    return nil
}
