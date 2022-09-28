package wrpc_go

import (
    "context"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/util/logx"
    "net"
    "net/http"
    "sync"
)

var (
	ErrPageNotFound = fmt.Errorf("404 page not found")
	ErrMethodNotFound = fmt.Errorf("method not found")
)

type HttpServer struct {
    *http.Server
    opts *ServerOptions
    mu sync.Mutex
    name string
}

func NewHttpServer(name string, handler http.Handler, opts...ServerOption) *HttpServer {
    srv := &HttpServer{
        Server: &http.Server{
            Handler: withHttpHandlerRecover(handler),
        },
        name: name,
    }
    srv.opts = loadServerOptions(name, opts...)
    srv.Server.Addr = srv.opts.addr
    return srv
}

func (srv *HttpServer)Start() error {
    listen, err := net.Listen("tpc", srv.opts.addr)
    if err != nil {
        return err
    }
    return srv.Serve(listen)
}

func (srv *HttpServer)Stop(ctx context.Context) error {
    return srv.Shutdown(ctx)
}

func (srv *HttpServer)Name() string {
    return srv.name
}

func (srv *HttpServer)String() string {
    return srv.name
}

type httpHandlerRecover struct {
    http.Handler
}

func withHttpHandlerRecover(parent http.Handler) http.Handler {
    handler := &httpHandlerRecover{Handler: parent}
    return handler
}

func (mux *httpHandlerRecover)ServeHTTP(rw http.ResponseWriter, req *http.Request) {
    defer logx.Recover()
    mux.Handler.ServeHTTP(rw, req)
}
