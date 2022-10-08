package wrpc_go

import (
    "context"
    "fmt"
    "github.com/wukong-cloud/wrpc-go/internal/register"
    "github.com/wukong-cloud/wrpc-go/util/logx"
    "sync"
    "time"
)

type AppOption interface {
    apply(app *App)
}

type AppOptionFunc func(app *App)

func (f AppOptionFunc)apply(app *App) {
    f(app)
}

func WithServer(server Server) AppOption {
    return AppOptionFunc(func(app *App) {
        app.serverMap[server.Name()] = server
    })
}

func WithRegister(register register.Register) AppOption {
    return AppOptionFunc(func(app *App) {
        app.register = register
    })
}

type App struct {
    serverMap map[string]Server
    stopChan chan struct{}
    wg sync.WaitGroup
    register register.Register
}

func NewApp(opts ...AppOption) *App {
    conf := GetConfig()
   app := &App{
       serverMap: make(map[string]Server),
       stopChan: make(chan struct{}),
       register: register.NewRegister(conf.RegisterConfig),
   }
   for _, opt := range opts {
       opt.apply(app)
   }
   return app
}

func (app *App)AddServer(server Server) {
    app.serverMap[server.Name()] = server
}

func (app *App)Run() error {
    if len(app.serverMap) == 0 {
        return fmt.Errorf("server not found")
    }
    for _, server := range app.serverMap {
        app.wg.Add(1)
        go func() {
            defer app.wg.Done()
            if err := server.Start(); err != nil {
                panic(fmt.Sprintf("start server %s failed:%+v", server.Name(), err))
                return
            }
        }()
    }
    time.Sleep(time.Second)
    for _, server := range app.serverMap {
        app.register.Register(*server.Target())
    }
    return app.loop()
}

func (app *App)loop() error {
    timer := time.NewTicker(time.Second*10)
    for {
        select {
        case <- timer.C:
            logx.Log("keep alive")
            for _, server := range app.serverMap {
                app.register.KeepAlive(*server.Target())
            }
        case <- app.stopChan:
            for _, server := range app.serverMap {
                app.register.UnRegister(*server.Target())
                server.Stop(context.TODO())
            }
            app.wg.Wait()
        }
    }
}
