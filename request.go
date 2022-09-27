package wrpc_go

type Request struct {
    RequestId int32
    Method    string
    Meta      Meta
    Body      []byte
}
