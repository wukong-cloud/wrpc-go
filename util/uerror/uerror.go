package uerror

import "encoding/json"

type Error struct {
    Code   int32
    ErrMsg string
}

var (
    ErrRequestTimeout  = NewError(405, "request timeout")
    ErrRequestFull     = NewError(502, "request full")
    ErrEncoderNotFound = NewError(404, "encoder not found")
)

func NewError(code int32, errMsg string) error {
    err := &Error{Code: code, ErrMsg: errMsg}
    return err
}

func (e *Error)Error() string {
    if e == nil {
        return "nil"
    }
    bin, _ := json.Marshal(e)
    return string(bin)
}

func ParseError(e error) *Error {
    if e == nil {
        return nil
    }
    err := &Error{}
    if je := json.Unmarshal([]byte(e.Error()), err); je != nil {
        err.Code = 502
        err.ErrMsg = e.Error()
    }
    return err
}
