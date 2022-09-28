package wrpc_go

import (
    "context"
    "time"
)

const (
	defaultReadBufSize = 8192
	defaultMaxInvoke   = 10000
)

type Dispatcher func(ctx context.Context, impl interface{}, req *Request, enc Encoder) ([]byte, error)

type Server interface {
    Start() error
    Stop(ctx context.Context) error
    Name() string
}

type ServerOptions struct {
    addr          string
    maxInvoke     int32
    invokeTimeout time.Duration
    readSize      int32
    tick          chan struct{}
}

func loadServerOptions(name string, opts ...ServerOption) *ServerOptions {
    cfg := GetServerConfig(name)
    option := &ServerOptions{
        readSize: cfg.ReadBufferSize,
        maxInvoke: cfg.MaxInvoke,
        addr: ":"+cfg.Port,
    }

    for _, opt := range opts {
        opt(option)
    }

    option.tick = make(chan struct{}, option.maxInvoke)
    return option
}

type ServerOption func(opt *ServerOptions)

func WithServerOptionReadSize(size int32) ServerOption {
    return func(opt *ServerOptions) {
        opt.readSize = size
    }
}

func WithServerOptionAddr(addr string) ServerOption {
    return func(opt *ServerOptions) {
        opt.addr = addr
    }
}
