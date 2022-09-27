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
    Addr          string
    MaxInvoke     int32
    InvokeTimeout time.Duration
    ReadSize      int
    Tick          chan struct{}
}

func loadServerOptions(opts ...ServerOption) *ServerOptions {
    option := &ServerOptions{
        ReadSize: defaultReadBufSize,
        MaxInvoke: defaultMaxInvoke,
    }

    for _, opt := range opts {
        opt(option)
    }

    option.Tick = make(chan struct{}, option.MaxInvoke)
    return option
}

type ServerOption func(opt *ServerOptions)

func WithServerOptionReadSize(size int) ServerOption {
    return func(opt *ServerOptions) {
        opt.ReadSize = size
    }
}

func WithServerOptionAddr(addr string) ServerOption {
    return func(opt *ServerOptions) {
        opt.Addr = addr
    }
}
