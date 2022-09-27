package wrpc_go

import (
    "encoding/json"
    "fmt"
    "github.com/golang/protobuf/proto"
)

type Encoder interface {
    Encode(v interface{}) ([]byte, error)
    Decode(bs []byte, addr interface{}) error
    Name() string
}

var encMap map[string]Encoder

func init() {
    encMap = make(map[string]Encoder)
    RegisterEncoder(&jsonEncoder{})
    RegisterEncoder(&protoEncoder{})
}

func RegisterEncoder(enc Encoder) {
    if enc == nil || enc.Name() == "" {
        return
    }
    encMap[enc.Name()] = enc
}

func GetEncoder(name string) Encoder {
    if enc, ok := encMap[name]; ok {
        return enc
    }
    return nil
}

const _encoder_json = "json"
const _encoder_proto = "proto"

type jsonEncoder struct{}

func (*jsonEncoder)Encode(v interface{}) ([]byte, error) { return json.Marshal(v) }
func (*jsonEncoder)Decode(bs []byte, addr interface{}) error { return json.Unmarshal(bs, addr) }
func (*jsonEncoder)Name() string { return _encoder_json }

type protoEncoder struct {}

func (*protoEncoder)Encode(v interface{}) ([]byte, error) {
    msg, ok := v.(proto.Message)
    if !ok {
        return nil, fmt.Errorf("type %T not proto.Message", v)
    }
    return proto.Marshal(msg)
}

func (*protoEncoder)Decode(bs []byte, addr interface{}) error {
    msg, ok := addr.(proto.Message)
    if !ok {
        return fmt.Errorf("type %T not proto.Message", addr)
    }
    return proto.Unmarshal(bs, msg)
}

func (*protoEncoder)Name() string {
    return _encoder_proto
}
