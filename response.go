package wrpc_go

import "github.com/wukong-cloud/wrpc-go/util/uerror"

type Response struct {
    RequestId  int32
    Body       []byte
    Meta       Meta
    Code       int32
    CodeStatus string
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
