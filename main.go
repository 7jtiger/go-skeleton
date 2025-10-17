package main

import (
	"ms-gateway/common/logger"
	"ms-gateway/conf"
	ctl "ms-gateway/controller"
	"ms-gateway/models"
	rt "ms-gateway/router"
	schd "ms-gateway/scheduler"
	"net"
	"strings"

	"github.com/pion/turn/v2"

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

func startTurnServer(cf *conf.Config) *turn.Server {
	var turnServer *turn.Server

	// TURN 서버 설정 (config.toml 설정에 따라 활성화)
	if cf.Turn.Enabled {
		logger.Info("Initializing TURN server...")

		// TURN 서버 리스너 생성
		listenAddr := cf.Turn.ListenAddr
		if listenAddr == "" {
			listenAddr = "0.0.0.0:3478" // 기본값
		}

		udpListener, err := net.ListenPacket("udp4", listenAddr)
		if err != nil {
			panic(fmt.Errorf("failed to create UDP listener on %s: %v", listenAddr, err))
		}

		// RelayAddressGenerator 설정 (환경별 최적화)
		var relayGenerator turn.RelayAddressGenerator
		if cf.Server.Mode == "prod" {
			// 프로덕션: 포트 범위 제한으로 보안 강화
			minPort := cf.Turn.Relay.MinPort
			maxPort := cf.Turn.Relay.MaxPort
			maxRetries := cf.Turn.Relay.MaxRetries

			if minPort == 0 {
				minPort = 49152 // 기본값
			}
			if maxPort == 0 {
				maxPort = 65535 // 기본값
			}
			if maxRetries == 0 {
				maxRetries = 100 // 기본값
			}

			relayGenerator = &turn.RelayAddressGeneratorPortRange{
				RelayAddress: net.ParseIP("0.0.0.0"),
				Address:      "0.0.0.0",
				MinPort:      uint16(minPort),
				MaxPort:      uint16(maxPort),
				MaxRetries:   maxRetries,
			}
			logger.Info("TURN server using port range relay generator", "minPort", minPort, "maxPort", maxPort)
		} else {
			// 개발/테스트: 동적 포트 할당
			relayGenerator = &turn.RelayAddressGeneratorNone{
				Address: "0.0.0.0",
			}
			logger.Info("TURN server using dynamic port allocation")
		}

		// AuthHandler 설정 (long-term credentials 지원)
		authHandler := turn.AuthHandler(func(username, realm string, srcAddr net.Addr) ([]byte, bool) {
			logger.Debug("TURN auth request", "username", username, "realm", realm, "srcAddr", srcAddr)

			// Long-term credentials 지원 (프로덕션 권장)
			if cf.Turn.SharedSecret != "" && cf.Server.Mode == "prod" {
				// RFC 5389 Long-term credential 검증
				// 실제 구현에서는 timestamp 기반 validation 필요
				if strings.Contains(username, ":") {
					return turn.GenerateAuthKey(username, realm, cf.Turn.SharedSecret), true
				}
			}

			// 기본 사용자 인증
			if cf.Turn.Username != "" && cf.Turn.Password != "" {
				if username == cf.Turn.Username {
					return turn.GenerateAuthKey(username, realm, cf.Turn.Password), true
				}
			}

			// WebRTC 설정의 TURN 인증 정보 지원 (하위 호환성)
			if cf.WebRTC.TurnUsername != "" && cf.WebRTC.TurnPassword != "" {
				if username == cf.WebRTC.TurnUsername {
					return turn.GenerateAuthKey(username, realm, cf.WebRTC.TurnPassword), true
				}
			}

			logger.Warn("TURN authentication failed", "username", username)
			return nil, false
		})

		// 성능 최적화 설정
		channelBindTimeout := time.Duration(cf.Turn.Performance.ChannelBindTimeout) * time.Second
		if channelBindTimeout == 0 {
			channelBindTimeout = 10 * time.Minute // 기본값
		}

		inboundMTU := cf.Turn.Performance.InboundMTU
		if inboundMTU == 0 {
			inboundMTU = 1500 // 기본값
		}

		// TURN 서버 구성
		realm := cf.Turn.Realm
		if realm == "" {
			realm = "ms-gateway-turn" // 기본값
		}

		turnServer, err = turn.NewServer(turn.ServerConfig{
			Realm:       realm,
			AuthHandler: authHandler,
			PacketConnConfigs: []turn.PacketConnConfig{
				{
					PacketConn:            udpListener,
					RelayAddressGenerator: relayGenerator,
					PermissionHandler: func(clientAddr net.Addr, peerIP net.IP) bool {
						// 보안 설정에 따른 권한 제어
						if cf.Turn.Security.EnableAuthentication {
							logger.Debug("Permission check", "client", clientAddr, "peer", peerIP)
							// 여기서 추가적인 보안 로직 구현 가능
							return true // 현재는 모든 연결 허용
						}
						return true
					},
				},
			},
			ChannelBindTimeout: channelBindTimeout,
			InboundMTU:         inboundMTU,
		})
		if err != nil {
			panic(fmt.Errorf("failed to create TURN server: %v", err))
		}

		logger.Info("TURN server configured successfully",
			"listenAddr", listenAddr,
			"realm", realm,
			"channelBindTimeout", channelBindTimeout,
			"inboundMTU", inboundMTU,
			"mode", cf.Server.Mode)
	} else {
		logger.Info("TURN server is disabled")
		return nil
	}

	return turnServer
}

// startTurnMonitoring TURN 서버 모니터링을 별도 함수로 분리
func startTurnMonitoring(ctx context.Context, turnServer *turn.Server) error {
	if turnServer == nil {
		return nil
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.Info("TURN server monitoring started")

	for {
		select {
		case <-ticker.C:
			allocCount := turnServer.AllocationCount()
			logger.Debug("TURN server stats", " activeAllocations : ", allocCount)

			// 통계를 Redis에 저장하거나 모니터링 시스템에 전송 가능
			// mod.GetRedisDB().SetCache("turn:stats:allocations", strconv.Itoa(allocCount))

		case <-ctx.Done():
			logger.Info("TURN server monitoring stopped")
			return nil
		}
	}
}

func main() {
	flag.Parse()
	cf := conf.NewConfig(*configFlag)

	if err := logger.InitLogger(cf.Server.Name, cf.Server.Mode, cf.LogInfo.MaxAgeHour, cf.LogInfo.RotateHour); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}

	logger.Debug("ready server....")

	/* 	hch := hachecker.NewHAChecker(cf)
	   	if hch == nil {
	   		fmt.Printf("hachecker.NewHAChecker failed")
	   		return
	   	}
	*/
	//model 모듈 선언
	if mod, err := models.NewModel(cf); err != nil {
		panic(err)
		// } else if controller, err := ctl.NewCTL(cf, hch, mod); err != nil {
	} else if controller, err := ctl.NewCTL(cf, mod); err != nil {
		panic(fmt.Errorf("controller.New > %v", err))
	} else if _, err := schd.NewScheduler(cf, mod); err != nil { // 스케쥴러 초기화 추가
		panic(fmt.Errorf("scheduler.New > %v", err))
	} else if rt, err := rt.NewRouter(cf, controller); err != nil {
		panic(fmt.Errorf("router.NewRouter > %v", err))
	} else {
		// Context 생성 (graceful shutdown을 위해)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// TURN 서버 시작
		turnServer := startTurnServer(cf)

		// TURN 서버 모니터링 고루틴 (활성화된 경우에만)
		if turnServer != nil {
			g.Go(func() error {
				return startTurnMonitoring(ctx, turnServer)
			})
		}

		// HTTP 서버 설정
		mapi := &http.Server{
			Addr:           cf.Server.Port,
			Handler:        rt.Idx(),
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    120 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		// HTTP 서버 고루틴
		g.Go(func() error {
			logger.Info("HTTP server listening on", cf.Server.Port)
			if err := mapi.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		})

		// 종료 시그널 대기
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Warn("Shutdown signal received, starting graceful shutdown...")

		// Graceful shutdown 시작
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// 1. 모니터링 및 백그라운드 고루틴 종료
		logger.Info("Stopping background services...")
		cancel() // 모든 context를 사용하는 고루틴들에게 종료 신호

		// 2. TURN 서버 종료 (활성화된 경우에만)
		if turnServer != nil {
			logger.Info("Shutting down TURN server...")
			if err := turnServer.Close(); err != nil {
				logger.Error("TURN Server shutdown error:", err)
			} else {
				logger.Info("TURN Server shutdown successfully")
			}
		}

		// 3. HTTP 서버 종료
		logger.Info("Shutting down HTTP server...")
		if err := mapi.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP Server shutdown error:", err)
		} else {
			logger.Info("HTTP Server shutdown successfully")
		}

		// 4. 모든 고루틴 종료 대기 (타임아웃 포함)
		done := make(chan error, 1)
		go func() {
			done <- g.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.Error("Some goroutines failed to shutdown cleanly:", err)
			} else {
				logger.Info("All services shutdown successfully")
			}
		case <-shutdownCtx.Done():
			logger.Warn("Shutdown timeout reached, forcing exit")
		}

		logger.Info("Server exiting")
	}
}
