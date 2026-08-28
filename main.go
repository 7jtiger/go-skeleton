package main

// @title ms-gateway API
// @version 1.0
// @description ms-gateway API server

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	haredis "ms-gateway/common/ha-redis"
	"ms-gateway/common/logger"
	"ms-gateway/conf"
	ctl "ms-gateway/controller"
	"ms-gateway/models"
	rt "ms-gateway/router"
	schd "ms-gateway/scheduler"

)

const (
	// 서버 타임아웃 설정 (동영상 최대 100MB 업로드 고려)
	serverReadTimeout    = 5 * time.Minute
	serverWriteTimeout   = 5 * time.Minute
	serverIdleTimeout    = 120 * time.Second
	serverMaxHeaderBytes = 1 << 20 // 1MB

	// Graceful shutdown 타임아웃
	shutdownTimeout = 15 * time.Second
)

var (
	configFlag = flag.String("config", "./conf/config.toml", "toml file to use for configuration")
)

func main() {
	// 설정 파일 로드
	flag.Parse()
	cf, err := loadConfig(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 로거 초기화
	if err := initLogger(cf); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting ms-gateway server...", "mode", cf.Server.Mode, "port", cf.Server.Port)

	// 서버 실행
	if err := runServer(cf); err != nil {
		logger.Error("Server failed to start:", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown completed")
}

// loadConfig 설정 파일을 로드하고 검증합니다
func loadConfig(configPath string) (*conf.Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	// 설정 파일 존재 확인
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// 설정 로드 (panic 방지를 위해 recover 사용)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic while loading config: %v", r)
			panic(err)
		}
	}()

	cf := conf.NewConfig(configPath)
	if cf == nil {
		return nil, fmt.Errorf("failed to load config from %s", configPath)
	}

	// 필수 설정 검증
	if cf.Server.Port == "" {
		return nil, fmt.Errorf("server port is not configured")
	}

	return cf, nil
}

// initializeLogger 로거를 초기화합니다
func initLogger(cf *conf.Config) error {
	if cf.Server.Name == "" {
		return fmt.Errorf("server name is not configured")
	}

	if err := logger.InitLogger(
		cf.Server.Name,
		cf.Server.Mode,
		cf.LogInfo.MaxAgeHour,
		cf.LogInfo.RotateHour,
	); err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}

	return nil
}

// runServer 서버를 초기화하고 실행합니다
func runServer(cf *conf.Config) error {

	// 모델 초기화
	mod, err := models.NewModel(cf)
	if err != nil {
		return fmt.Errorf("failed to initialize models: %w", err)
	}
	logger.Info("Models initialized successfully")

	hch, err := haredis.NewHAChecker(cf, mod)
	if hch == nil {
		fmt.Printf("hachecker.NewHAChecker failed")
		os.Exit(1)
	}

	// 컨트롤러 초기화
	controller, err := ctl.NewCTL(cf, hch, mod)
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %w", err)
	}
	logger.Info("Controller initialized successfully")

	// 스케줄러 초기화
	scheduler, err := schd.NewScheduler(cf, hch, mod)
	if err != nil {
		return fmt.Errorf("failed to initialize scheduler: %w", err)
	}
	logger.Info("Scheduler initialized successfully", "jobs", len(cf.Works))

	// 라우터 초기화
	router, err := rt.NewRouter(cf, controller)
	if err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}
	logger.Info("Router initialized successfully")

	// Context 생성 (graceful shutdown용)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// HTTP 서버 설정
	server := &http.Server{
		Addr:           cf.Server.Port,
		Handler:        router.Idx(),
		ReadTimeout:    serverReadTimeout,
		WriteTimeout:   serverWriteTimeout,
		IdleTimeout:    serverIdleTimeout,
		MaxHeaderBytes: serverMaxHeaderBytes,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("HTTP server error: %w", err)
			return
		}
		serverErr <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-quit:
		logger.Warn("Shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			return err
		}
		logger.Warn("HTTP server stopped unexpectedly")
	}

	return gracefulShutdown(&shutdownDeps{
		server:     server,
		cancel:     cancel,
		scheduler:  scheduler,
		controller: controller,
		hch:        hch,
		mod:        mod,
	})
}

type shutdownDeps struct {
	server     *http.Server
	cancel     context.CancelFunc
	scheduler  interface{}
	controller *ctl.Controller
	hch        *haredis.HAChecker
	mod        *models.Repositories
}

// gracefulShutdown 서버를 안전하게 종료합니다
func gracefulShutdown(deps *shutdownDeps) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	logger.Info("Initiating graceful shutdown...")

	if deps.cancel != nil {
		deps.cancel()
	}

	if deps.controller != nil {
		logger.Info("Closing WebSocket connections...")
		deps.controller.Shutdown()
	}

	if deps.scheduler != nil {
		if s, ok := deps.scheduler.(interface{ Stop() }); ok {
			logger.Info("Stopping scheduler...")
			s.Stop()
		}
	}

	if deps.hch != nil {
		logger.Info("Stopping HA checker...")
		deps.hch.Shutdown()
	}

	logger.Info("Shutting down HTTP server...")
	if err := deps.server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error:", err)
	} else {
		logger.Info("HTTP server shutdown successfully")
	}

	if deps.mod != nil {
		logger.Info("Stopping repositories...")
		deps.mod.Shutdown()
	}

	return nil
}
