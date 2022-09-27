package wrpc_go

import (
    "context"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/util/uerror"
    "math"
    "net"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

var ErrConnectNotFound = fmt.Errorf("connect not found")

var requestId int32

func nextRequestId() int32 {
    if atomic.CompareAndSwapInt32(&requestId, math.MaxInt32, 1) {
        return 1
    }
    id := atomic.AddInt32(&requestId, 1)
    return id
}

type ClientOptions struct {
    addr           string
    maxConn        int
    RequestTimeout time.Duration
    maxLiveTime    time.Duration
    ReadSize       int32
    EncodeType     string
}

type ClientOption func(opt *ClientOptions)

func WithClientOptionAddr(addr string) ClientOption {
    return func(opt *ClientOptions) {
        opt.addr = addr
    }
}

func WithClientOptionMaxConn(max int) ClientOption {
    return func(opt *ClientOptions) {
        opt.maxConn = max
    }
}

func WithClientOptionEncodeType(encodeType string) ClientOption {
    return func(opt *ClientOptions) {
        opt.EncodeType = encodeType
    }
}

func loadClientOptions(opts ...ClientOption) *ClientOptions {
    options := &ClientOptions{
        RequestTimeout: time.Second*3,
        ReadSize: defaultReadBufSize,
        maxConn: 1,
        maxLiveTime: time.Hour*2,
        EncodeType: _encoder_json,
    }
    for _, opt := range opts {
        opt(options)
    }
    return options
}

type Client struct {
    name string
    opts *ClientOptions
    mu   sync.Mutex
    protocol Protocol
    idx int
    connectors []*connector
    reqMap sync.Map
}

func NewClient(name string, opts ...ClientOption) *Client {
    client := &Client{
        name: name,
        protocol: newWRPCProtocol(),
        connectors: make([]*connector, 0),
    }
    client.opts = loadClientOptions(opts...)
    client.initConnect()
    return client
}

func (client *Client)initConnect() {
    if client.opts.addr != "" {
        addrs := strings.Split(client.opts.addr, ";")
        for _, addr := range addrs {
            addr := strings.TrimSpace(addr)
            if addr == "" {
                continue
            }
            connect := newConnector(client, addr, true)
            client.connectors = append(client.connectors, connect)
        }
    }
}

func (client *Client)connector(addr string) *connector {
    var connect *connector
    client.mu.Lock()
    if addr != "" {
        for _, c := range client.connectors {
            if c.addr == addr {
                connect = c
                break
            }
        }
        client.mu.Unlock()
        return connect
    } else {
        connectNum := len(client.connectors)
        if connectNum == 0 {
            client.mu.Unlock()
            return nil
        }
        if client.idx >= connectNum {
            client.idx = 0
        }
        connect = client.connectors[client.idx]
        client.idx++
    }
    client.mu.Unlock()
    return connect
}

func (client *Client)Invoke(ctx context.Context, encName, addr, method string, in []byte, opt ...map[string]string) ([]byte, error) {
    var cancel context.CancelFunc
    if client.opts.RequestTimeout > 0 {
        ctx, cancel = context.WithTimeout(ctx, client.opts.RequestTimeout)
        defer cancel()
    }
    metadata, ok := FromOutgoingContext(ctx)
    if !ok {
        metadata = make(map[string]string)
    }
    if encName == "" {
        encName = client.opts.EncodeType
    }
    metadata.Set(EncodeType, encName)
    req := &Request{
        RequestId: nextRequestId(),
        Method: method,
        Body: in,
        Meta: metadata,
    }
    bs, err := client.protocol.PacketRequest(req)
    if err != nil {
        return nil, err
    }

    respChan := make(chan *Response)
    client.reqMap.Store(req.RequestId, respChan)

    for i := 0; i < 2; i++ {
        connect := client.connector(addr)
        if connect == nil {
            return nil, ErrConnectNotFound
        }
        conn, cerr := connect.getConn()
        if cerr != nil {
            time.Sleep(time.Millisecond*10)
            err = cerr
            continue
        }
        if serr := conn.send(bs); serr != nil {
            time.Sleep(time.Millisecond*10)
            err = cerr
            continue
        }
        err = nil
        break
    }
    if err != nil {
        client.reqMap.Delete(req.RequestId)
        return nil, err
    }

    select {
    case <- ctx.Done():
        client.reqMap.Delete(req.RequestId)
        return nil, uerror.ErrRequestTimeout
    case resp := <- respChan:
        if resp.Code > 0 {
            return nil, uerror.NewError(resp.Code, resp.ErrMsg)
        }
        return resp.Body, nil
    }
}

type connector struct {
    addr   string
    client *Client
    nextId int32
    idx   int
    conns []*clientConn
    mu    sync.Mutex
    isFixed bool
}

func newConnector(client *Client, addr string, isFixed bool) *connector {
    c := &connector{
        client: client,
        addr: addr,
        isFixed: isFixed,
        conns: make([]*clientConn, 0, client.opts.maxConn),
    }
    return c
}

func (c *connector)getConn() (*clientConn, error) {
    c.mu.Lock()
    connNum := len(c.conns)
    if connNum == 0 {
        conn, err := net.Dial("tcp", c.addr)
        if err != nil {
            c.mu.Unlock()
            return nil, err
        }
        clientConn := newClientConn(c, conn)
        c.conns = append(c.conns, clientConn)
        connNum++
    }
    if c.idx >= connNum {
        if connNum < c.client.opts.maxConn && c.idx < c.client.opts.maxConn {
            conn, err := net.Dial("tcp", c.addr)
            if err == nil {
                clientConn := newClientConn(c, conn)
                c.conns = append(c.conns, clientConn)
                connNum++
            }
        }
    }
    if c.idx >= connNum {
        c.idx = 0
    }
    conn := c.conns[c.idx]
    c.idx++
    c.mu.Unlock()
    return conn, nil
}

func (c *connector)nextConnId() int32 {
    id := atomic.AddInt32(&c.nextId, 1)
    return id
}

func (c *connector)removeConn(connId int32) {
    conns := make([]*clientConn, 0)
    c.mu.Lock()
    for _, conn := range c.conns {
        if conn.connId == connId {
            continue
        }
        conns = append(conns, conn)
    }
    c.conns = conns
    c.mu.Unlock()
}

type clientConn struct {
    connId   int32
    connect  *connector
    rw       net.Conn
    running  bool
    callNum  int32
    createAt time.Time
    useAt    time.Time
    mu       sync.Mutex
}

func newClientConn(c *connector, rw net.Conn) *clientConn {
    conn := &clientConn{
        connId: c.nextConnId(),
        connect: c,
        running: true,
        rw: rw,
        createAt: time.Now(),
    }
    go conn.recv(rw)
    return conn
}

func (conn *clientConn)reconnect() error {
    conn.mu.Lock()
    if conn.running && !conn.expired() {
        conn.mu.Unlock()
        return nil
    }
    rw, err := net.Dial("tcp", conn.connect.addr)
    if err != nil {
        conn.mu.Unlock()
        return err
    }
    conn.rw = rw
    conn.callNum = 0
    conn.running = true
    conn.createAt = time.Now()
    conn.mu.Unlock()
    go conn.recv(rw)
    return nil
}

func (conn *clientConn)expired() bool {
    if conn.connect.client.opts.maxLiveTime > 0 {
        return time.Now().Sub(conn.createAt) >= conn.connect.client.opts.maxLiveTime
    }
    return false
}

func (conn *clientConn)close() {
    conn.mu.Lock()
    if conn.closed() {
        conn.mu.Unlock()
        return
    }
    conn.running = false
    conn.mu.Unlock()
    conn.rw.Close()
    conn.connect.removeConn(conn.connId)
}

func (conn *clientConn)closed() bool {
    return conn.running == false
}

func (conn *clientConn)recv(rw net.Conn) {
    defer rw.Close()
    defer conn.close()

    var (
        buf = make([]byte, 0, 8192)
        readBuf = make([]byte, conn.connect.client.opts.ReadSize)
    )

    for {
        if conn.closed() {
            return
        }
        n, err := rw.Read(readBuf)
        if err != nil {
            return
        }
        buf = append(buf, readBuf[:n]...)
        for {
            body, n, state := readBody(buf)
            if state == state_full {
                buf = buf[n:]
                conn.invoke(body)
                continue
            }
            if state == state_need_read {
                break
            }
            return
        }
    }
}

func (conn *clientConn)invoke(body []byte) {
    protocol := conn.connect.client.protocol
    req, err := protocol.UnPacketResponse(body)
    if err != nil {
        return
    }
    val, ok := conn.connect.client.reqMap.Load(req.RequestId)
    if ok {
        conn.connect.client.reqMap.Delete(req.RequestId)
        respChan, rok := val.(chan *Response)
        if rok {
            respChan <- req
        }
    }
}

func (conn *clientConn)send(pkg []byte) error {
    if err := conn.reconnect(); err != nil {
        return err
    }
    _, err := conn.rw.Write(pkg)
    conn.useAt = time.Now()
    return err
}