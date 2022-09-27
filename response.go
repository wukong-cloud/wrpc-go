package wrpc_go

import "github.com/wukong-cloud/wrpc-go/util/uerror"

type Response struct {
    RequestId int32
    Body []byte
    Code int32
    ErrMsg string
}

func GetResponse(req *Request, bs []byte, err error) *Response {
    resp := &Response{
        RequestId: req.RequestId,
        Body: bs,
    }
    if err != nil {
        werr := uerror.ParseError(err)
        resp.Code = werr.Code
        resp.ErrMsg = werr.ErrMsg
    }
    return resp
}
