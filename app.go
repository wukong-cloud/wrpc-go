package wrpc_go

import (
    "context"
    "fmt"
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

type App struct {
    serverMap map[string]Server
    stopChan chan struct{}
    wg sync.WaitGroup
}

func NewApp(opts ...AppOption) *App {
   app := &App{
       serverMap: make(map[string]Server),
       stopChan: make(chan struct{}),
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
                logx.Log(logx.Kv("message", "server stop"), logx.Kv("server", server.Name()), logx.Kv("error", err.Error()))
                return
            }
        }()
    }
    return app.loop()
}

func (app *App)loop() error {
    timer := time.NewTicker(time.Second*10)
    for {
        select {
        case <- timer.C:
            logx.Log(logx.Kv("message", "keepAlive"))
        case <- app.stopChan:
            for _, server := range app.serverMap {
                server.Stop(context.TODO())
            }
            app.wg.Wait()
        }
    }
}
