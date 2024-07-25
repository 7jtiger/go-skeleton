package main

import (
	"go-skeleton/common/logger"
	"go-skeleton/conf"
	ctl "go-skeleton/controller"
	"go-skeleton/models"
	rt "go-skeleton/router"
	schd "go-skeleton/scheduler"

	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	g errgroup.Group
)
var configFlag = flag.String("config", "./conf/config.toml", "toml file to use for configuration")

func main() {
	flag.Parse()
	cf := conf.NewConfig(*configFlag)

	if err := logger.InitLogger(cf); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}

	logger.Debug("ready server....")

	//model 모듈 선언
	if mod, err := models.NewModel(cf); err != nil {
		panic(err)
	} else if controller, err := ctl.NewCTL(cf, mod); err != nil {
		panic(fmt.Errorf("controller.New > %v", err))
	} else if _, err := schd.NewScheduler(cf, mod); err != nil { // 스케쥴러 초기화 추가
		panic(fmt.Errorf("scheduler.New > %v", err))
	} else if rt, err := rt.NewRouter(cf, controller); err != nil {
		panic(fmt.Errorf("router.NewRouter > %v", err))
	} else {
		mapi := &http.Server{
			Addr:           cf.Server.Port,
			Handler:        rt.Idx(),
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		g.Go(func() error {
			return mapi.ListenAndServe()
		})

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Warn("Shutdown Server ...")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := mapi.Shutdown(ctx); err != nil {
			logger.Error("Server Shutdown:", err)
		}

		<-ctx.Done()
		// logger.Info("timeout of 5 seconds.")
		logger.Info("Server exiting")
	}

	if err := g.Wait(); err != nil {
		logger.Error(err)
	}
}
