package wrpc_go

import (
    "encoding/binary"
    "github.com/golang/protobuf/proto"
)

type Protocol interface {
    PacketRequest(request *Request) ([]byte, error)
    UnPacketRequest(body []byte) (*Request, error)

    PacketResponse(response *Response) ([]byte, error)
    UnPacketResponse(body []byte) (*Response, error)

    Name() string
}

type wrpcProtocol struct{}

func newWRPCProtocol() Protocol {
    return &wrpcProtocol{}
}

func (jp *wrpcProtocol) PacketRequest(request *Request) ([]byte, error) {
    bs, err := proto.Marshal(request)
    if err != nil {
        return nil, err
    }
    pkg := make([]byte, len(bs)+4)
    binary.LittleEndian.PutUint32(pkg, uint32(len(bs)+4))
    copy(pkg[4:], bs)
    return pkg, nil
}

func (jp *wrpcProtocol) UnPacketRequest(body []byte) (*Request, error) {
    req := Request{}
    err := proto.Unmarshal(body[4:], &req)
    if err != nil {
        return nil, err
    }
    return &req, nil
}

func (jp *wrpcProtocol) PacketResponse(response *Response) ([]byte, error) {
    bs, err := proto.Marshal(response)
    if err != nil {
        return nil, err
    }
    pkg := make([]byte, len(bs)+4)
    binary.LittleEndian.PutUint32(pkg, uint32(len(bs)+4))
    copy(pkg[4:], bs)
    return pkg, nil
}

func (jp *wrpcProtocol) UnPacketResponse(body []byte) (*Response, error) {
    resp := Response{}
    err := proto.Unmarshal(body[4:], &resp)
    if err != nil {
        return nil, err
    }
    return &resp, nil
}

func (jp *wrpcProtocol) Name() string {
    return "proto-protocol"
}
