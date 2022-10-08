package discovery

import (
    "context"
    clientv3 "go.etcd.io/etcd/client/v3"
    "strings"
    "time"
)

type EtcdDiscover struct {
    client  *clientv3.Client
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
    discover := &EtcdDiscover{client: cli}
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
