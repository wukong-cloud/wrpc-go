package logx

import (
    "encoding/json"
    "fmt"
    "log"
    "runtime"
    "strconv"
)

type Field struct {
    key string
    val interface{}
}

func Kv(k string, v interface{}) *Field {
    return &Field{key: k, val: v}
}

func (f *Field) String() string {
    if f == nil {
        return ""
    }
    if f.val == nil {
        return "\"" + f.key + "\":nil"
    }
    switch v := f.val.(type) {
    case uint:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(v), 10)
    case uint8:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(v), 10)
    case uint16:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(v), 10)
    case uint32:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(v), 10)
    case uint64:
        return "\"" + f.key + "\":" + strconv.FormatUint(v, 10)
    case int:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(v), 10)
    case int8:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(v), 10)
    case int16:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(v), 10)
    case int32:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(v), 10)
    case int64:
        return "\"" + f.key + "\":" + strconv.FormatInt(v, 10)
    case *uint:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(*v), 10)
    case *uint8:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(*v), 10)
    case *uint16:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(*v), 10)
    case *uint32:
        return "\"" + f.key + "\":" + strconv.FormatUint(uint64(*v), 10)
    case *uint64:
        return "\"" + f.key + "\":" + strconv.FormatUint(*v, 10)
    case *int:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(*v), 10)
    case *int8:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(*v), 10)
    case *int16:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(*v), 10)
    case *int32:
        return "\"" + f.key + "\":" + strconv.FormatInt(int64(*v), 10)
    case *int64:
        return "\"" + f.key + "\":" + strconv.FormatInt(*v, 10)
    case float32:
        return "\"" + f.key + "\":" + strconv.FormatFloat(float64(v), 'f', -1, 64)
    case float64:
        return "\"" + f.key + "\":" + strconv.FormatFloat(v, 'f', -1, 64)
    case *float32:
        return "\"" + f.key + "\":" + strconv.FormatFloat(float64(*v), 'f', -1, 64)
    case *float64:
        return "\"" + f.key + "\":" + strconv.FormatFloat(*v, 'f', -1, 64)
    case bool:
        return "\"" + f.key + "\":" + strconv.FormatBool(v)
    case *bool:
        return "\"" + f.key + "\":" + strconv.FormatBool(*v)
    case string:
        return "\"" + f.key + "\":\"" + v + "\""
    case *string:
        return "\"" + f.key + "\":\"" + *v + "\""
    case []byte:
        return "\"" + f.key + "\":\"" + string(v) + "\""
    default:
        bs, err := json.Marshal(v)
        if err != nil {
            return fmt.Sprintf("\"%s\":\"%+v\"", f.key, f.val)
        }
        return "\"" + f.key + "\":\"" + string(bs) + "\""
    }
}

type Logger interface {
    Log(args...interface{})
    Logf(format string, args...interface{})
}

var inLog Logger = &consoleLog{}

func Log(args...interface{}) {
    inLog.Log(args...)
}

func Logf(format string, args...interface{}) {
    inLog.Logf(format, args...)
}

func Recover() {
    if err := recover(); err != nil {
        const size = 64 << 10
        buf := make([]byte, size)
        buf = buf[:runtime.Stack(buf, false)]
        Logf("panic recover:\n%s", buf)
    }
}

type consoleLog struct {}
func (l *consoleLog)Log(args...interface{}) { log.Println(args...) }
func (l *consoleLog)Logf(format string, args...interface{}) { log.Printf(format, args...) }
