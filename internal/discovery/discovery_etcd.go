package discovery

import (
    "context"
    "go.etcd.io/etcd/api/v3/mvccpb"
    clientv3 "go.etcd.io/etcd/client/v3"
    "strings"
    "sync"
    "time"
)

type EtcdDiscover struct {
    client  *clientv3.Client
    watchCh chan []string
    once    sync.Once
}

func NewEtcdDiscover(hosts string) (Discover, error) {
    endpoints := strings.Split(hosts, ";")
    cli, err := clientv3.New(clientv3.Config{
        Endpoints: endpoints,
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    discover := &EtcdDiscover{client: cli, watchCh: make(chan []string)}
    return discover, nil
}

func (cli *EtcdDiscover)Find(name string) []string {
    resp, err := cli.client.Get(context.TODO(), name, clientv3.WithPrefix())
    if err != nil {
        return nil
    }
    endpoints := make([]string, 0)
    for _, kv := range resp.Kvs {
        endpoints = append(endpoints, string(kv.Value))
    }
    return endpoints
}

func (cli *EtcdDiscover)Watch(name string) chan []string {
    cli.once.Do(func() {
        go func() {
            watchCh := cli.client.Watch(context.TODO(), name, clientv3.WithPrefix())
            for n := range watchCh {
                for _, env := range n.Events {
                    switch env.Type {
                    case mvccpb.DELETE:
                        newList := cli.Find(name)
                        cli.watchCh <- newList
                    case mvccpb.PUT:
                        newList := cli.Find(name)
                        cli.watchCh <- newList
                    }
                }
            }
        }()
    })
    return cli.watchCh
}
