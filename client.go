package wrpc_go

import (
    "context"
    "fmt"
    "github.com/serialx/hashring"
    "github.com/wukong-cloud/wrpc-go/internal/discovery"
    "github.com/wukong-cloud/wrpc-go/util/uerror"
    "math"
    "net"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

var ErrConnectNotFound = fmt.Errorf("connect not found")

var requestId int64

func nextRequestId() int64 {
    if atomic.CompareAndSwapInt64(&requestId, math.MaxInt32, 1) {
        return 1
    }
    id := atomic.AddInt64(&requestId, 1)
    return id
}

type ClientOptions struct {
    addr           string
    maxConn        int
    requestTimeout time.Duration
    maxIdleTime    time.Duration
    readSize       int32
    encodeType     string
    reTry          int
    discover       discovery.Discover
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
        opt.encodeType = encodeType
    }
}

func WithClientOptionsDiscover(discover discovery.Discover) ClientOption {
    return func(opt *ClientOptions) {
        opt.discover = discover
    }
}

func loadClientOptions(opts ...ClientOption) *ClientOptions {
    cfg := GetClientConfig()
    options := &ClientOptions{
        requestTimeout: cfg.RequestTimeout,
        readSize: cfg.ReadBufferSize,
        maxConn: cfg.Thread,
        maxIdleTime: cfg.MaxIdleTime,
        encodeType: cfg.EncodeType,
        reTry: cfg.ReTry,
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
    reqMap map[int64]chan *Response
    rwLock sync.Mutex
    discover discovery.Discover
    hasher *hashring.HashRing
}

func NewClient(name string, opts ...ClientOption) *Client {
    conf := GetConfig()
    client := &Client{
        name: name,
        protocol: newWRPCProtocol(),
        connectors: make([]*connector, 0),
        reqMap: make(map[int64]chan *Response),
        discover: discovery.NewDiscover(conf.DiscoverConfig),
        hasher: hashring.New([]string{}),
    }
    client.opts = loadClientOptions(opts...)
    if client.opts.discover != nil {
        client.discover = client.opts.discover
    }
    client.initConnect()
    return client
}

func (client *Client)initConnect() {
    if client.opts.addr != "" {
        client.updateConnector(client.opts.addr, true)
    }
    endpoints := client.discover.Find(client.name)
    if len(endpoints) > 0 {
        client.updateConnector(strings.Join(endpoints, ";"), false)
    }
    go func() {
        timer := time.NewTicker(time.Second*10)
        for {
            select {
            case <- timer.C:
                endpoints := client.discover.Find(client.name)
                client.updateConnector(strings.Join(endpoints, ";"), false)
            case endpoints := <- client.discover.Watch(client.name):
                client.updateConnector(strings.Join(endpoints, ";"), false)
            }
        }
    }()
}

func (client *Client)updateConnector(addr string, isFixed bool) {
    client.mu.Lock()

    addrs := strings.Split(addr, ";")
    oldConnectors := client.connectors
    newConnectors := make([]*connector, 0)

    for _, addr := range addrs {
        addr := strings.TrimSpace(addr)
        if addr == "" {
            continue
        }
        connect := newConnector(client, addr, isFixed)
        newConnectors = append(newConnectors, connect)
    }

    delConnectoers := make([]*connector, 0)
    for _, oldConn := range oldConnectors {
        isFound := false
        for _, newConn := range newConnectors {
            if oldConn.addr == newConn.addr {
                isFound = true
                break
            }
        }
        if isFound {
            continue
        }
        if oldConn.isFixed {
            newConnectors = append(newConnectors, oldConn)
            continue
        }
        delConnectoers = append(delConnectoers, oldConn)
    }
    newNodes := map[string]int{}
    for _, node := range newConnectors {
        newNodes[node.addr] = 10
    }
    client.hasher = hashring.NewWithWeights(newNodes)
    client.connectors = newConnectors
    client.mu.Unlock()

    if len(delConnectoers) > 0 {
        go func() {
            for _, delConn := range delConnectoers {
                delConn.close()
            }
        }()
    }
}

func (client *Client)connector(key string, findType int) *connector {
    switch findType {
    case findType_addr:
        return client.findConnector(key)
    case findType_consistentHash:
        return client.consistentHashConnector(key)
    default:
        return client.nextConnector()
    }
}

func (client *Client)findConnector(addr string) *connector {
    var connect *connector
    client.mu.Lock()
    for _, c := range client.connectors {
        if c.addr == addr {
            connect = c
            break
        }
    }
    client.mu.Unlock()
    return connect
}

func (client *Client)consistentHashConnector(key string) *connector {
    var connect *connector
    client.mu.Lock()
    node, ok := client.hasher.GetNode(key)
    if !ok {
        client.mu.Unlock()
        return connect
    }
    client.mu.Unlock()
    return client.findConnector(node)
}

func (client *Client)nextConnector() *connector {
    var connect *connector
    client.mu.Lock()
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
    client.mu.Unlock()
    return connect
}

func (client *Client)GetAllEndpoints() []string {
    addrs := make([]string, 0)
    client.mu.Lock()
    for _, connect := range client.connectors {
        addrs = append(addrs, connect.addr)
    }
    client.mu.Unlock()
    return addrs
}

func (client *Client)Invoke(ctx context.Context, encName, addr, method string, in []byte, opt ...map[string]string) ([]byte, error) {
    var cancel context.CancelFunc
    if client.opts.requestTimeout > 0 {
        ctx, cancel = context.WithTimeout(ctx, client.opts.requestTimeout)
        defer cancel()
    }
    metadata, ok := FromOutgoingContext(ctx)
    if !ok {
        metadata = make(map[string]string)
    }
    if encName == "" {
        encName = client.opts.encodeType
    }
    metadata.Set(EncodeType, encName)
    req := &Request{
        RequestId: nextRequestId(),
        Method: method,
        Body: in,
        Meta: metadata,
    }

    respChan := make(chan *Response, 1)
    client.rwLock.Lock()
    client.reqMap[req.RequestId] = respChan
    client.rwLock.Unlock()

    err := client.sendRequest(addr, req)
    if err != nil {
        client.rwLock.Lock()
        delete(client.reqMap, req.RequestId)
        close(respChan)
        client.rwLock.Unlock()
        return nil, err
    }

    select {
    case <- ctx.Done():
        client.rwLock.Lock()
        delete(client.reqMap, req.RequestId)
        close(respChan)
        client.rwLock.Unlock()
        return nil, uerror.ErrRequestTimeout
    case resp, ok := <- respChan:
        if !ok {
            return nil, uerror.NewError(502, "chan is closed")
        }
        close(respChan)
        if resp.Code > 0 && resp.Code != 200 {
            return nil, uerror.NewError(resp.Code, resp.CodeStatus)
        }
        return resp.Body, nil
    }
}

const (
    findType_next = 1
    findType_addr = 2
    findType_consistentHash = 3
)

func (client *Client)sendRequest(addr string, req *Request) error {
    bs, err := client.protocol.PacketRequest(req)
    if err != nil {
        return  err
    }

    tryTime := client.opts.reTry
    if tryTime <= 0 {
        tryTime = 1
    }

    findType := findType_next
    key := ""
    if addr != "" {
        key = addr
        findType = findType_addr
    } else if hash, ok := req.Meta[ConsistentHashKey]; ok {
        key = hash
        findType = findType_consistentHash
    }

    for i := 0; i < tryTime; i++ {
        if i > 0 && findType != findType_addr {
            findType = findType_next
        }
        connect := client.connector(key, findType)
        if connect == nil {
            return ErrConnectNotFound
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
    return err
}

type connector struct {
    addr    string
    client  *Client
    nextId  int32
    idx     int
    conns   []*clientConn
    mu      sync.Mutex
    callNum int
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

func (c *connector)close() {
    for _, conn := range c.conns {
        conn.close()
    }
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
    if conn.connect.client.opts.maxIdleTime > 0 {
        return time.Now().Sub(conn.createAt) >= conn.connect.client.opts.maxIdleTime
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
        buf = make([]byte, 0, conn.connect.client.opts.readSize)
        readBuf = make([]byte, conn.connect.client.opts.readSize)
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
    conn.connect.client.rwLock.Lock()
    respChan, ok := conn.connect.client.reqMap[req.RequestId]
    if ok {
        delete(conn.connect.client.reqMap, req.RequestId)
        respChan <- req
    }
    conn.connect.client.rwLock.Unlock()
}

func (conn *clientConn)send(pkg []byte) error {
    if err := conn.reconnect(); err != nil {
        return err
    }
    _, err := conn.rw.Write(pkg)
    conn.useAt = time.Now()
    return err
}
