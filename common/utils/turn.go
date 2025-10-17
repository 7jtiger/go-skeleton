package utils

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"github.com/pion/turn/v2"
)

// TurnCredentials TURN 인증 정보
type TurnCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TTL      int    `json:"ttl"`
}

// GenerateTurnCredentials Long-term credential을 생성합니다.
// RFC 5389 섹션 10.2에 따른 time-windowed credentials 생성
func GenerateTurnCredentials(sharedSecret string, username string, ttl int) *TurnCredentials {
	if ttl <= 0 {
		ttl = 86400 // 24시간 기본값
	}

	// timestamp 기반 사용자명 생성 (만료 시간 포함)
	timestamp := time.Now().Unix() + int64(ttl)
	timeUsername := fmt.Sprintf("%d:%s", timestamp, username)

	// HMAC-SHA1을 사용하여 패스워드 생성
	h := hmac.New(sha1.New, []byte(sharedSecret))
	h.Write([]byte(timeUsername))
	password := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return &TurnCredentials{
		Username: timeUsername,
		Password: password,
		TTL:      ttl,
	}
}

// ValidateTurnCredentials time-windowed credential을 검증합니다.
func ValidateTurnCredentials(username, password, sharedSecret string) bool {
	// 사용자명에서 timestamp 추출
	var timestamp int64
	var actualUsername string

	if n, err := fmt.Sscanf(username, "%d:%s", &timestamp, &actualUsername); n != 2 || err != nil {
		return false
	}

	// 만료 시간 확인
	if time.Now().Unix() > timestamp {
		return false
	}

	// 패스워드 검증
	h := hmac.New(sha1.New, []byte(sharedSecret))
	h.Write([]byte(username))
	expectedPassword := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return password == expectedPassword
}

// TurnServerStats TURN 서버 통계 정보
type TurnServerStats struct {
	ActiveAllocations   int                    `json:"activeAllocations"`
	TotalAllocations    int64                  `json:"totalAllocations"`
	TotalBytes          int64                  `json:"totalBytes"`
	ActiveSessions      int                    `json:"activeSessions"`
	Uptime              time.Duration          `json:"uptime"`
	LastStatsUpdate     time.Time              `json:"lastStatsUpdate"`
	AllocationsByClient map[string]int         `json:"allocationsByClient"`
	Performance         TurnPerformanceMetrics `json:"performance"`
}

// TurnPerformanceMetrics TURN 서버 성능 메트릭
type TurnPerformanceMetrics struct {
	PacketsPerSecond      float64 `json:"packetsPerSecond"`
	BytesPerSecond        float64 `json:"bytesPerSecond"`
	AverageLatency        float64 `json:"averageLatency"`
	PacketLossRate        float64 `json:"packetLossRate"`
	ConnectionSuccessRate float64 `json:"connectionSuccessRate"`
}

// TurnServerConfig TURN 서버 설정 도우미 구조체
type TurnServerConfig struct {
	Enabled               bool                       `json:"enabled"`
	ListenAddr            string                     `json:"listenAddr"`
	Realm                 string                     `json:"realm"`
	AuthType              string                     `json:"authType"`  // "static", "long-term", "custom"
	RelayType             string                     `json:"relayType"` // "none", "static", "range"
	RelayAddressGenerator turn.RelayAddressGenerator `json:"-"`
	AuthHandler           turn.AuthHandler           `json:"-"`
	PerformanceConfig     TurnPerformanceConfig      `json:"performance"`
	SecurityConfig        TurnSecurityConfig         `json:"security"`
}

// TurnPerformanceConfig TURN 성능 설정
type TurnPerformanceConfig struct {
	ChannelBindTimeout  time.Duration `json:"channelBindTimeout"`
	AllocationLifetime  time.Duration `json:"allocationLifetime"`
	PermissionLifetime  time.Duration `json:"permissionLifetime"`
	InboundMTU          int           `json:"inboundMTU"`
	MaxConcurrentAllocs int           `json:"maxConcurrentAllocs"`
}

// TurnSecurityConfig TURN 보안 설정
type TurnSecurityConfig struct {
	EnableAuthentication bool                `json:"enableAuthentication"`
	RequireCredentials   bool                `json:"requireCredentials"`
	AllowedIPRanges      []string            `json:"allowedIPRanges"`
	BlockedIPRanges      []string            `json:"blockedIPRanges"`
	RateLimiting         TurnRateLimitConfig `json:"rateLimiting"`
}

// TurnRateLimitConfig TURN 속도 제한 설정
type TurnRateLimitConfig struct {
	Enabled           bool `json:"enabled"`
	MaxAllocsPerIP    int  `json:"maxAllocsPerIP"`
	MaxBytesPerSecond int  `json:"maxBytesPerSecond"`
	WindowSize        int  `json:"windowSize"` // seconds
}

// CreateOptimizedTurnServer 최적화된 TURN 서버 생성
func CreateOptimizedTurnServer(config TurnServerConfig) (*turn.Server, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("TURN server is disabled")
	}

	// 기본값 설정
	if config.ListenAddr == "" {
		config.ListenAddr = "0.0.0.0:3478"
	}
	if config.Realm == "" {
		config.Realm = "ms-gateway"
	}
	if config.PerformanceConfig.ChannelBindTimeout == 0 {
		config.PerformanceConfig.ChannelBindTimeout = 10 * time.Minute
	}
	if config.PerformanceConfig.InboundMTU == 0 {
		config.PerformanceConfig.InboundMTU = 1500
	}

	// 서버 설정 구성
	serverConfig := turn.ServerConfig{
		Realm:              config.Realm,
		AuthHandler:        config.AuthHandler,
		ChannelBindTimeout: config.PerformanceConfig.ChannelBindTimeout,
		InboundMTU:         config.PerformanceConfig.InboundMTU,
	}

	return turn.NewServer(serverConfig)
}

// MonitorTurnServer TURN 서버 모니터링
func MonitorTurnServer(server *turn.Server, statsCallback func(stats TurnServerStats)) {
	if server == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	var totalAllocations int64
	var totalBytes int64

	for range ticker.C {
		stats := TurnServerStats{
			ActiveAllocations: server.AllocationCount(),
			TotalAllocations:  totalAllocations,
			TotalBytes:        totalBytes,
			Uptime:            time.Since(startTime),
			LastStatsUpdate:   time.Now(),
		}

		if statsCallback != nil {
			statsCallback(stats)
		}
	}
}

// LogTurnServerStats TURN 서버 통계 로깅
func LogTurnServerStats(stats TurnServerStats) {
	fmt.Printf("[TURN] Active Allocations: %d, Uptime: %v, Last Update: %v\n",
		stats.ActiveAllocations,
		stats.Uptime,
		stats.LastStatsUpdate.Format("2006-01-02 15:04:05"))
}

// GetTurnServerHealth TURN 서버 상태 확인
func GetTurnServerHealth(server *turn.Server) map[string]interface{} {
	if server == nil {
		return map[string]interface{}{
			"status":      "disabled",
			"allocations": 0,
			"healthy":     false,
		}
	}

	allocCount := server.AllocationCount()
	return map[string]interface{}{
		"status":      "running",
		"allocations": allocCount,
		"healthy":     true,
		"timestamp":   time.Now().Unix(),
	}
}

// OptimizeTurnServerForProduction 프로덕션 환경을 위한 TURN 서버 최적화
func OptimizeTurnServerForProduction(baseConfig TurnServerConfig) TurnServerConfig {
	// 프로덕션 최적화 설정
	baseConfig.PerformanceConfig.ChannelBindTimeout = 5 * time.Minute  // 더 짧은 타임아웃
	baseConfig.PerformanceConfig.AllocationLifetime = 30 * time.Minute // 할당 생명주기 제한
	baseConfig.PerformanceConfig.InboundMTU = 1400                     // 더 작은 MTU로 호환성 증대
	baseConfig.PerformanceConfig.MaxConcurrentAllocs = 1000            // 동시 할당 제한

	// 보안 강화
	baseConfig.SecurityConfig.EnableAuthentication = true
	baseConfig.SecurityConfig.RequireCredentials = true
	baseConfig.SecurityConfig.RateLimiting.Enabled = true
	baseConfig.SecurityConfig.RateLimiting.MaxAllocsPerIP = 10
	baseConfig.SecurityConfig.RateLimiting.WindowSize = 300 // 5분

	return baseConfig
}

// CreateTurnAuthHandler 인증 핸들러 생성 도우미
func CreateTurnAuthHandler(authType string, config map[string]string) turn.AuthHandler {
	switch authType {
	case "long-term":
		sharedSecret := config["shared_secret"]
		if sharedSecret == "" {
			sharedSecret = "default-secret"
		}
		return turn.NewLongTermAuthHandler(sharedSecret, nil)

	case "static":
		username := config["username"]
		password := config["password"]
		realm := config["realm"]
		if realm == "" {
			realm = "ms-gateway"
		}

		return turn.AuthHandler(func(user, r string, srcAddr net.Addr) ([]byte, bool) {
			if user == username && r == realm {
				return turn.GenerateAuthKey(user, r, password), true
			}
			return nil, false
		})

	default:
		// 기본 인증 (모든 연결 허용 - 개발용만)
		return turn.AuthHandler(func(user, realm string, srcAddr net.Addr) ([]byte, bool) {
			return turn.GenerateAuthKey(user, realm, "default"), true
		})
	}
}
