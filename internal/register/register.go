package register

type Target struct {
    IP   string
    Port string
    Name string
}

type Register interface {
    Register(target Target) error
    UnRegister(target Target) error
    KeepAlive(target Target) error
}
