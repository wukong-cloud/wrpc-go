package wrpc_go

import (
    "context"
    "encoding/json"
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
    srv.opts = loadServerOptions(opts...)
    srv.Server.Addr = srv.opts.Addr
    return srv
}

func NewHttpRPCServer(name string, dispatch Dispatcher, opts...ServerOption) *HttpServer {
    srv := &HttpServer{
        Server: &http.Server{},
        name: name,
    }

    handler := &HttpHandlerRPC{
        srv: srv,
        dispatch: dispatch,
    }
    srv.Server.Handler = handler

    srv.opts = loadServerOptions(opts...)
    srv.Server.Addr = srv.opts.Addr
    return srv
}

func (srv *HttpServer)Start() error {
    listen, err := net.Listen("tpc", srv.opts.Addr)
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

type HttpHandlerRPC struct {
    route    string
    dispatch Dispatcher
    srv      *HttpServer
}

func (mux *HttpHandlerRPC)ServeHTTP(rw http.ResponseWriter, req *http.Request) {
    defer logx.Recover()

    if err := mux.checkRequest(req); err != nil {
        mux.ReturnError(rw, http.StatusNotFound, err)
        return
    }

}

func (mux *HttpHandlerRPC)checkRequest(req *http.Request) error {
    if req.Method != http.MethodPost {
        return ErrMethodNotFound
    }
    uri := req.RequestURI
    if uri != mux.route {
        return ErrPageNotFound
    }
    return nil
}

func (mux *HttpHandlerRPC)ReturnError(rw http.ResponseWriter, code int, err error) {
    rw.WriteHeader(code)
    rw.Write([]byte(err.Error()))
}

func (mux *HttpHandlerRPC)ReturnJson(rw http.ResponseWriter, code int, v interface{}) {
    rw.Header().Set("Content-Type", "application/json;charset=UTF-8")
    rw.WriteHeader(code)
    if v == nil {
        rw.Write([]byte("{}"))
    } else {
        bs, _ := json.Marshal(v)
        rw.Write(bs)
    }
}
