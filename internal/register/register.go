package register

type Target struct {
    IP   string
    Port string
    Name string
}

func (t *Target)String() string {
    if t == nil {
        return ""
    }
    return t.Name + "/" + t.IP + ":" + t.Port
}

func (t *Target)Host() string {
    return t.IP + ":" + t.Port
}

type RegisterConfig struct {
    Name  string `yaml:"name"`
    Hosts string `yaml:"hosts"`
}

type Register interface {
    Register(target Target) error
    UnRegister(target Target) error
    KeepAlive(target Target) error
}

type nopRegister struct {}

func (*nopRegister)Register(target Target) error { return nil }
func (*nopRegister)UnRegister(target Target) error { return nil }
func (*nopRegister)KeepAlive(target Target) error { return nil }

func NewRegister(conf *RegisterConfig) Register {
    var regist Register = &nopRegister{}
    if conf == nil {
        return regist
    }
    switch conf.Name {
    case "etcd":
        register, err := NewEtcdRegister(conf.Hosts)
        if err == nil {
            regist = register
        }
    }
    return regist
}
