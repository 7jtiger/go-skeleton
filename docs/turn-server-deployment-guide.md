# TURN 서버 배포 및 최적화 가이드

## 개요

이 문서는 ms-gateway 프로젝트의 TURN 서버 구성, 배포 및 최적화에 대한 종합적인 가이드입니다.

## TURN 서버 구성

### 1. 기본 설정 (config.toml)

```toml
[webrtc]
signalingServerUrl = "wss://your-domain.com/ws"
stunServers = [
    "stun:stun.l.google.com:19302",
    "stun:stun1.l.google.com:19302"
]
turnServer = "turn:your-domain.com:3478"
turnUsername = "turn"
turnPassword = "secure-password"

[turn]
enabled = true
listenAddr = "0.0.0.0:3478"
realm = "ms-gateway-turn"
username = "turn"
password = "secure-password"
sharedSecret = "your-shared-secret-2024"
credentialTTL = 86400

[turn.relay]
minPort = 49152
maxPort = 65535
maxRetries = 100

[turn.performance]
channelBindTimeout = 600
inboundMTU = 1500
allocationLifetime = 3600

[turn.security]
permissionLifetime = 300
enableAuthentication = true
requireCredentials = true
```

### 2. 환경별 최적화

#### 개발 환경
- 동적 포트 할당 사용
- 간단한 정적 인증
- 모든 연결 허용

#### 프로덕션 환경
- 포트 범위 제한 (49152-65535)
- Long-term credentials 사용
- 보안 강화 설정

## 성능 최적화

### 1. 네트워크 최적화

```bash
# UDP 버퍼 크기 증가
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.rmem_default = 65536' >> /etc/sysctl.conf
echo 'net.core.wmem_default = 65536' >> /etc/sysctl.conf

# 적용
sysctl -p
```

### 2. 방화벽 설정

```bash
# TURN 서버 포트 열기
ufw allow 3478/udp
ufw allow 3478/tcp

# Relay 포트 범위 열기
ufw allow 49152:65535/udp

# 특정 IP에서만 접근 허용 (선택적)
ufw allow from YOUR_CLIENT_IP to any port 3478
```

### 3. 모니터링

#### Prometheus 메트릭
```go
// 메트릭 수집 예제
func collectTurnMetrics(server *turn.Server) {
    allocationsGauge.Set(float64(server.AllocationCount()))
    uptimeGauge.Set(time.Since(startTime).Seconds())
}
```

#### 로그 설정
```toml
[loginfo]
fpath = "./logs/gateway"
maxAgeHour = 168  # 7일
rotateHour = 24   # 매일 로테이션
```

## 보안 강화

### 1. Long-term Credentials

```go
// RFC 5389 Long-term credentials 생성
credentials := utils.GenerateTurnCredentials(
    "your-shared-secret", 
    "username", 
    86400, // 24시간
)
```

### 2. IP 화이트리스트

```toml
[whitelist]
ips = [
    "192.168.1.0/24",
    "10.0.0.0/8",
    "client-ip-address"
]
```

### 3. Rate Limiting

```toml
[turn.security.rateLimiting]
enabled = true
maxAllocsPerIP = 10
maxBytesPerSecond = 1048576  # 1MB/s
windowSize = 300  # 5분
```

## Docker 배포

### 1. Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o ms-gateway .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ms-gateway .
COPY --from=builder /app/conf ./conf
EXPOSE 8080 3478/udp
CMD ["./ms-gateway"]
```

### 2. docker-compose.yml

```yaml
version: '3.8'
services:
  ms-gateway:
    build: .
    ports:
      - "8080:8080"
      - "3478:3478/udp"
      - "49152-65535:49152-65535/udp"
    environment:
      - SERVER_MODE=prod
    volumes:
      - ./conf:/root/conf
      - ./logs:/root/logs
    restart: unless-stopped
    
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  redis_data:
```

## 성능 테스트

### 1. 연결 테스트

```bash
# STUN 테스트
stun-client stun://your-domain.com:3478

# TURN 테스트
turn-client -host your-domain.com:3478 -user turn:password
```

### 2. 부하 테스트

```go
// 동시 연결 테스트
func TestTurnServerLoad(t *testing.T) {
    for i := 0; i < 100; i++ {
        go func(id int) {
            client := createTurnClient()
            conn, err := client.Allocate()
            if err != nil {
                t.Errorf("Client %d failed: %v", id, err)
            }
            defer conn.Close()
            
            // 트래픽 시뮬레이션
            time.Sleep(30 * time.Second)
        }(i)
    }
}
```

## 트러블슈팅

### 1. 일반적인 문제들

#### 연결 실패
- 방화벽 설정 확인
- NAT 설정 확인
- 인증 정보 확인

#### 성능 저하
- CPU/메모리 사용률 확인
- 네트워크 대역폭 확인
- 동시 연결 수 제한 확인

#### 메모리 누수
- 할당 생명주기 설정 확인
- 정기적인 가비지 컬렉션 모니터링

### 2. 로그 분석

```bash
# TURN 관련 로그 필터링
grep "TURN" /path/to/logs/gateway.log

# 에러 로그 확인
grep "ERROR" /path/to/logs/gateway.log | grep -i turn

# 연결 통계
grep "Active Allocations" /path/to/logs/gateway.log
```

## 모니터링 대시보드

### Grafana 설정 예제

```json
{
  "dashboard": {
    "title": "TURN Server Monitoring",
    "panels": [
      {
        "title": "Active Allocations",
        "type": "graph",
        "targets": [
          {
            "expr": "turn_active_allocations"
          }
        ]
      },
      {
        "title": "Bandwidth Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "turn_bytes_per_second"
          }
        ]
      }
    ]
  }
}
```

## 고가용성 구성

### 1. 로드 밸런서 설정

```nginx
upstream turn_servers {
    least_conn;
    server turn1.example.com:3478;
    server turn2.example.com:3478;
    server turn3.example.com:3478;
}

server {
    listen 3478 udp;
    proxy_pass turn_servers;
    proxy_timeout 1s;
    proxy_responses 1;
}
```

### 2. 클러스터 설정

여러 TURN 서버 인스턴스를 실행하여 고가용성을 확보할 수 있습니다:

- 각 인스턴스는 다른 포트 범위 사용
- Redis를 통한 세션 공유
- 헬스 체크 및 자동 장애 조치

## 참고 자료

- [RFC 5766 - TURN Specification](https://tools.ietf.org/html/rfc5766)
- [RFC 5389 - STUN Specification](https://tools.ietf.org/html/rfc5389)
- [Pion TURN Documentation](https://pkg.go.dev/github.com/pion/turn/v2)
- [WebRTC TURN 가이드](https://webrtc.org/getting-started/turn-server) 