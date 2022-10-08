package discovery

import "github.com/wukong-cloud/wrpc-go/util/logx"

type Discover interface {
    Find(name string) []string
}

type DiscoverConfig struct {
    Name  string `yaml:"name"`
    Hosts string `yaml:"hosts"`
}

var defaultDiscover = &nopDiscover{}

var discover Discover = defaultDiscover

func NewDiscover(conf *DiscoverConfig) Discover {
    if conf == nil {
        return discover
    }
    logx.Log(logx.Kv("isDefaultDiscover", discover == defaultDiscover))
    if discover != defaultDiscover {
        return discover
    }
    switch conf.Name {
    case "etcd":
        discove, err := NewEtcdDiscover(conf.Hosts)
        if err == nil {
            discover = discove
        }
    }
    return discover
}

type nopDiscover struct {}
func (*nopDiscover)Find(name string) []string { return nil }
