package wrpc_go

import "context"

type wrpcContextkey struct {}

type Meta map[string]string

func (m Meta)Set(k, v string) {
    if m == nil {
        return
    }
    m[k] = v
}

func (m Meta)Get(k string) string {
    if m == nil {
        return ""
    }
    return m[k]
}

func FromOutgoingContext(ctx context.Context) (Meta, bool) {
    value := ctx.Value(wrpcContextkey{})
    md, ok := value.(Meta)
    return md, ok
}

func NewOutgoingContext(ctx context.Context, meta Meta) context.Context {
    return context.WithValue(ctx, wrpcContextkey{}, meta)
}

const (
    EncodeType = "encode-type"
)
