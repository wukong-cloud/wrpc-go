package wrpc_go

import (
    "flag"
    "github.com/wukong-cloud/wrpc-go/internal/discovery"
    "github.com/wukong-cloud/wrpc-go/internal/register"
    "github.com/wukong-cloud/wrpc-go/util/logx"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os"
    "sync"
    "time"
)

type Config struct {
    DiscoverConfig *discovery.DiscoverConfig `yaml:"discover"`
    RegisterConfig *register.RegisterConfig `yaml:"register"`
    ServerConfigs []*ServerConfig `yaml:"server-config"`
    ClientConfig  *ClientConfig   `yaml:"client-config"`
}

type ServerConfig struct {
    Name           string `yaml:"name"`
    IP             string `yaml:"ip"`
    Port           string `yaml:"port"`
    MaxInvoke      int32  `yaml:"max-invoke"`
    ReadBufferSize int32  `yaml:"read-buffer-size"`
}

type ClientConfig struct {
    RequestTimeout time.Duration `yaml:"request-timeout"`
    ReadBufferSize int32         `yaml:"read-buffer-size"`
    Thread         int           `yaml:"thread"`
    MaxIdleTime    time.Duration `yaml:"max-idle-time"`
    EncodeType     string        `yaml:"encode-type"`
    ReTry          int           `yaml:"retry"`
}

var (
    _cfg *Config = nil

    initOnce sync.Once
)

func GetConfig() *Config {
    initOnce.Do(initConfig)
    return _cfg
}

func GetServerConfig(name string) *ServerConfig {
    cfg := GetConfig()
    for _, c := range cfg.ServerConfigs {
        if c.Name == name {
            return c
        }
    }
    return nil
}

func GetClientConfig() *ClientConfig {
    cfg := GetConfig()
    return cfg.ClientConfig
}

func initConfig() {
    var configFile string
    flag.StringVar(&configFile, "config", "config.yaml", " -config config.yaml")
    flag.Parse()
    file, err := os.Open(configFile)
    if file != nil {
        defer func() {
            file.Close()
        }()
    }
    if err != nil {
        panic(err)
    }
    data, err := ioutil.ReadAll(file)
    if err != nil {
        panic(err)
    }
    cfg := &Config{
        ClientConfig: defaultClientConfig(),
    }
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        panic(err)
    }

    for i := range cfg.ServerConfigs {
        if cfg.ServerConfigs[i].MaxInvoke <= 0 {
            cfg.ServerConfigs[i].MaxInvoke = defaultMaxInvoke
        }
        if cfg.ServerConfigs[i].ReadBufferSize <= 0 {
            cfg.ServerConfigs[i].ReadBufferSize = defaultReadBufSize
        }
    }

    cfg.ClientConfig.RequestTimeout = parseTimeout(int32(cfg.ClientConfig.RequestTimeout))

    logx.Log(logx.Kv("config", cfg))
    _cfg = cfg
}

func defaultClientConfig() *ClientConfig {
    return &ClientConfig{
        RequestTimeout: 60000,
        ReadBufferSize: defaultReadBufSize,
        MaxIdleTime:    7200000,
        Thread:         1,
        EncodeType:     "json",
        ReTry:          1,
    }
}

func parseTimeout(timeout int32) (ret time.Duration) {
    return time.Duration(timeout) * time.Millisecond
}

func GetRequestTimeout() time.Duration {
    cfg := GetConfig()
    if cfg == nil || cfg.ClientConfig == nil {
        return 0
    }
    return cfg.ClientConfig.RequestTimeout
}
