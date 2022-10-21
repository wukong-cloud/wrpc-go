package wrpc_go

import (
    "context"
    "encoding/binary"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/internal/register"
    "github.com/wukong-cloud/wrpc-go/util/logx"
    "github.com/wukong-cloud/wrpc-go/util/uerror"
    "net"
    "sync"
    "time"
)

var (
    ErrServerIsRunning = fmt.Errorf("rpc: server is running")
    ErrServerIsClosed = fmt.Errorf("rpc: server is closed")
)

type TcpServer struct {
    name string
    opts *ServerOptions
    listen net.Listener
    conns  map[*tcpConn]struct{}
    mu sync.Mutex
    protocol Protocol

    target *register.Target

    impl interface{}
    dispatcher Dispatcher

    doneChan chan struct{}
    running bool
}

func NewRPCServer(name string, impl interface{}, dispatcher Dispatcher, opts ...ServerOption) *TcpServer {
    srv := &TcpServer{
        name: name,
        conns: make(map[*tcpConn]struct{}),
        protocol: newWRPCProtocol(),

        impl: impl,
        dispatcher: dispatcher,
    }
    srv.opts = loadServerOptions(name, opts...)
    srv.target = &register.Target{
        Name: name,
        IP: srv.opts.ip,
        Port: srv.opts.port,
    }
    return srv
}

func (srv *TcpServer)Start() error {
    listen, err := net.Listen("tcp", srv.opts.addr)
    if err != nil {
        return err
    }
    srv.mu.Lock()
    if srv.running {
        srv.mu.Unlock()
        return ErrServerIsRunning
    }

    logx.Logf("start rpc server %s listen %s", srv.name, srv.opts.addr)

    srv.listen = listen
    srv.running = true
    srv.mu.Unlock()

    var tempDelay time.Duration

    for {
        rw, err := srv.listen.Accept()
        if err != nil {
            select {
            case <- srv.getDoneChan():
                return nil
            default:
            }
            logx.Logf("server %s listen accept failed:%v", srv.Name(), err)
            if ne, ok := err.(net.Error); ok && ne.Temporary() {
                if tempDelay == 0 {
                    tempDelay = 5 * time.Millisecond
                } else {
                    tempDelay *= 2
                }
                if max := 1 * time.Second; tempDelay > max {
                    tempDelay = max
                }
                logx.Logf("http: Accept error: %v; retrying in %v\n", err, tempDelay)
                time.Sleep(tempDelay)
                continue
            }
            return err
        }
        conn := newConn(srv, rw)
        go conn.handle()
    }
}

func (srv *TcpServer)Stop(ctx context.Context) error {
    srv.mu.Lock()
    if srv.running == false {
        srv.mu.Unlock()
        return nil
    }
    srv.listen.Close()
    srv.running = false
    srv.closeDoneChanLocked()
    conns := srv.conns
    srv.conns = make(map[*tcpConn]struct{})
    srv.mu.Unlock()

    for conn := range conns {
        conn.close()
    }
    return nil
}

func (srv *TcpServer)Name() string {
    return srv.name
}

func (srv *TcpServer)Target() *register.Target {
    return srv.target
}

func (srv *TcpServer)getDoneChan() <-chan struct{} {
    srv.mu.Lock()
    defer srv.mu.Unlock()
    return srv.getDoneChanLocked()
}

func (srv *TcpServer)getDoneChanLocked() chan struct{} {
    if srv.doneChan == nil {
        srv.doneChan = make(chan struct{})
    }
    return srv.doneChan
}

func (srv *TcpServer)closeDoneChanLocked() {
    ch := srv.getDoneChanLocked()
    select {
    case <-ch:
        // Already closed. Don't close again.
    default:
        // Safe to close here. We're the only closer, guarded
        // by s.mu.
        close(ch)
    }
}

func (srv *TcpServer)addConn(conn *tcpConn) {
    srv.mu.Lock()
    srv.conns[conn] = struct{}{}
    srv.mu.Unlock()
}

func (srv *TcpServer)removeConn(conn *tcpConn) {
    srv.mu.Lock()
    delete(srv.conns, conn)
    srv.mu.Unlock()
}

type tcpConn struct {
    ip string
    port string
    rw net.Conn
    srv *TcpServer
    mu sync.Mutex
    invokeNum int32
}

func newConn(srv *TcpServer, rw net.Conn) *tcpConn {
    ip, port, _ := net.SplitHostPort(rw.RemoteAddr().String())
    conn := &tcpConn{
        rw: rw,
        srv: srv,
        ip: ip,
        port: port,
    }
    return conn
}

func (conn *tcpConn) handle() {
    defer logx.Recover()
    defer conn.close()

    var (
        buf = make([]byte, 0, conn.srv.opts.readSize)
        readBuf = make([]byte, conn.srv.opts.readSize)
    )

    for {
        n, err := conn.rw.Read(readBuf)
        if err != nil {
            return
        }
        buf = append(buf, readBuf[:n]...)
        for {
            body, n, state := readBody(buf)
            if state == state_full {
                buf = buf[n:]
                go conn.invoke(body)
                continue
            }
            if state == state_need_read {
                break
            }
            return
        }
    }
}

func (conn *tcpConn)close() {
    conn.srv.removeConn(conn)
    conn.rw.Close()
}

const state_full = 1
const state_need_read = 2
const state_err = -1

func readBody(bs []byte) ([]byte, int, int) {
    if len(bs) <= 4 {
        return nil, 0, state_need_read
    }
    n := int(binary.LittleEndian.Uint32(bs))
    if n <= 4 {
        return nil, 0, state_err
    }
    if n <= len(bs) {
        return bs[:n], n, state_full
    }
    return nil, 0, state_need_read
}

func (conn *tcpConn)invoke(body []byte) {
    defer logx.Recover()

    req, err := conn.srv.protocol.UnPacketRequest(body)
    if err != nil {
        logx.Log(logx.Kv("message", "unpacket failed"), logx.Kv("protocol", conn.srv.protocol.Name()), logx.Kv("error", err))
        return
    }

    var resp *Response
    meta := Meta(req.Meta)
    encName := meta.Get(EncodeType)
    start := time.Now()
    defer func() {
        interval := time.Now().Sub(start)
        desc := ""
        code := int32(0)
        if resp != nil {
            code = resp.Code
            desc = resp.CodeStatus
        }
        logx.Log("request call time", logx.Kv("protocol", conn.srv.protocol.Name()), logx.Kv("server", conn.srv.Name()), logx.Kv("method", req.Method), logx.Kv("interval", int32(interval/time.Millisecond)), logx.Kv("code", code), logx.Kv("status", desc), logx.Kv("encoder", encName), logx.Kv("spend", interval.String()))
    }()

    ctx := NewOutgoingContext(context.TODO(), req.Meta)
    var cancel context.CancelFunc
    if conn.srv.opts.invokeTimeout > 0 {
        ctx, cancel = context.WithTimeout(ctx, conn.srv.opts.invokeTimeout)
    }
    if cancel != nil {
        defer cancel()
    }

    select {
    case <- ctx.Done():
        resp = GetResponse(req, nil, uerror.ErrRequestFull)
    case conn.srv.opts.tick <- struct{}{}:
        defer func() {
            <- conn.srv.opts.tick
        }()
    }

    enc := GetEncoder(encName)
    if enc == nil && resp == nil {
        resp = GetResponse(req, nil, uerror.ErrEncoderNotFound)
    }

    if resp == nil {
        respChan := make(chan *Response)
        go func() {
            defer logx.Recover()
            bin, err := conn.srv.dispatcher(ctx, conn.srv.impl, req, enc)
            resp := GetResponse(req, bin, err)
            respChan <- resp
        }()

        select {
        case <- ctx.Done():
            resp = GetResponse(req, nil, uerror.ErrRequestFull)
        case resp = <- respChan:
        }
        close(respChan)
    }

    bs, err := conn.srv.protocol.PacketResponse(resp)
    if err != nil {
        return
    }
    err = conn.send(bs)
}

func (conn *tcpConn)send(body []byte) error {
    _, err := conn.rw.Write(body)
    return err
}

func GetResponse(req *Request, bs []byte, err error) *Response {
    resp := &Response{
        RequestId: req.RequestId,
        Body: bs,
        Meta: req.Meta,
        Code: 200,
        CodeStatus: "ok",
    }
    if err != nil {
        werr := uerror.ParseError(err)
        resp.Code = werr.Code
        resp.CodeStatus = werr.ErrMsg
    }
    return resp
}
