# 대역폭 모니터링 가이드

## 📊 개요

최적화 서버(`server_optimized.go`)는 실시간으로 네트워크 트래픽을 추적하여 대역폭 사용량을 모니터링합니다.

## 🎯 추적 항목

### 1. 수신 데이터 (Received)
- **Total Received**: 클라이언트로부터 받은 총 데이터량 (KB)
- **Received/sec**: 초당 수신 데이터량 (KB/s)

### 2. 송신 데이터 (Sent)
- **Total Sent**: 클라이언트로 보낸 총 데이터량 (KB)
- **Sent/sec**: 초당 송신 데이터량 (KB/s)

## 📈 모니터링 방법

### 1. 콘솔 로그

서버는 5초마다 통계를 자동 출력합니다:

```
=== Server Stats ===
Active Connections: 100
Total Connections: 250
Active Rooms: 10
Total Messages: 5000
Messages/sec: 100
--- Packet Size ---
Total Received: 512.34 KB (0.50 MB)
Total Sent: 2048.56 KB (2.00 MB)
Received/sec: 10.24 KB/s
Sent/sec: 40.96 KB/s
--- Memory ---
Memory (Alloc): 15.23 MB
Memory (Sys): 28.45 MB
Goroutines: 150
==================
```

### 2. REST API (헬스 체크)

```bash
# 기본 헬스 체크
curl http://localhost:3000/health

# JSON 보기 좋게 출력
curl -s http://localhost:3000/health | jq

# 실시간 모니터링 (1초마다)
watch -n 1 'curl -s http://localhost:3000/health | jq'
```

**응답 예시**:
```json
{
  "status": "ok",
  "active_connections": 100,
  "total_rooms": 10,
  "total_messages": 5000,
  "bandwidth": {
    "total_received_kb": 512.34,
    "total_sent_kb": 2048.56,
    "received_kb_per_sec": 10.24,
    "sent_kb_per_sec": 40.96
  }
}
```

## 🔍 추적 구현 상세

### 수신 데이터 추적

```go
// readPump 함수에서 메시지 수신 시
for {
    _, message, err := c.conn.ReadMessage()
    if err != nil {
        break
    }
    
    // 수신 바이트 수 추적
    messageSize := int64(len(message))
    atomic.AddInt64(&s.stats.TotalBytesReceived, messageSize)
    
    // 워커 큐에 추가
    s.workerQueue <- WorkItem{client: c, message: message}
}
```

### 송신 데이터 추적

```go
// broadcastWorker에서 브로드캐스트 시
case job := <-s.broadcastQueue:
    messageSize := int64(len(job.message))
    sentCount := int64(0)
    
    // 각 클라이언트에게 전송
    for client := range job.room.Clients {
        if client.send <- job.message {
            sentCount++
        }
    }
    
    // 송신 바이트 수 추적 (메시지 크기 × 전송 성공한 클라이언트 수)
    atomic.AddInt64(&s.stats.TotalBytesSent, messageSize*sentCount)
```

### 초당 처리량 계산

```go
// statsCollector에서 1초마다 계산
func (s *Server) statsCollector() {
    ticker := time.NewTicker(1 * time.Second)
    
    for range ticker.C {
        s.stats.mu.Lock()
        
        // 초당 수신 바이트 계산
        currentBytesReceived := atomic.LoadInt64(&s.stats.TotalBytesReceived)
        s.stats.BytesReceivedPerSec = currentBytesReceived - s.stats.lastBytesReceived
        s.stats.lastBytesReceived = currentBytesReceived
        
        // 초당 송신 바이트 계산
        currentBytesSent := atomic.LoadInt64(&s.stats.TotalBytesSent)
        s.stats.BytesSentPerSec = currentBytesSent - s.stats.lastBytesSent
        s.stats.lastBytesSent = currentBytesSent
        
        s.stats.mu.Unlock()
    }
}
```

## 📊 추적되는 트래픽 유형

### 1. WebRTC 시그널링
- **Offer**: SDP offer 메시지 (~1-2 KB)
- **Answer**: SDP answer 메시지 (~1-2 KB)
- **ICE Candidate**: 네트워크 경로 정보 (~100-200 bytes)

### 2. 채팅 메시지
- 텍스트 메시지 크기에 따라 가변
- 평균 ~100-500 bytes per message

### 3. 제어 메시지
- **Join**: 방 입장 메시지 (~50 bytes)
- **Start**: WebRTC 시작 시그널 (~20 bytes)
- **Ping/Pong**: 헬스 체크 (~20 bytes)

## 💡 대역폭 최적화 팁

### 1. 메시지 크기 줄이기

**압축 사용**:
```go
import "compress/gzip"

// 큰 메시지는 압축
if len(message) > 1024 {
    compressed := gzipCompress(message)
    // 압축된 메시지 전송
}
```

### 2. Protocol Buffers 사용

JSON 대신 Protocol Buffers 사용 시 크기 50-70% 감소:

```protobuf
message ChatMessage {
    string sender = 1;
    string text = 2;
}
```

### 3. 불필요한 데이터 제거

메시지에서 중복되거나 불필요한 필드 제거:

```javascript
// 나쁜 예
{
    "type": "chat",
    "timestamp": "2024-11-04T12:00:00Z",
    "server_version": "1.0",
    "data": {
        "sender": "Alice",
        "message": "Hello",
        "room": "room1"  // 불필요 (이미 서버가 알고 있음)
    }
}

// 좋은 예
{
    "type": "chat",
    "data": {
        "sender": "Alice",
        "message": "Hello"
    }
}
```

### 4. 배치 전송

여러 메시지를 배치로 모아서 전송:

```go
// 이미 구현됨: writePump에서 배치 전송
n := len(c.send)
for i := 0; i < n; i++ {
    w.Write(<-c.send)
}
```

## 📈 성능 벤치마크

### 일반적인 대역폭 사용량

#### 1:1 비디오 채팅 (시그널링만)
```
초기 연결: ~5-10 KB (Offer, Answer, ICE)
Ping/Pong: ~0.5 KB/min
채팅: ~0.5-2 KB per message
```

#### 100명 방
```
초기 연결: ~500 KB-1 MB
메시지 브로드캐스트: ~50-200 KB per message
Ping/Pong: ~50 KB/min
```

### 예상 대역폭 (1000명 동시 접속)

```
수신:
- Join 메시지: 1000 × 50 bytes = 50 KB
- 채팅 메시지 (100 msg/sec): 100 × 200 bytes = 20 KB/s
- ICE candidates: ~50 KB/s
총 수신: ~70-100 KB/s

송신:
- Start 시그널: ~20 KB
- Ping 메시지: 1000 × 20 bytes = 20 KB per minute
- 브로드캐스트 (100 msg/sec, 평균 10명 방):
  100 × 200 bytes × 9 = ~180 KB/s
총 송신: ~200-250 KB/s
```

## 🔍 트러블슈팅

### 대역폭이 예상보다 높은 경우

1. **메시지 크기 확인**
```bash
# 콘솔 로그에서 평균 메시지 크기 확인
Total Messages: 10000
Total Received: 5000 KB
평균 메시지 크기: 5000 / 10 = 500 bytes
```

2. **불필요한 메시지 확인**
- Ping/Pong 빈도 조정
- 브로드캐스트 최적화
- 메시지 중복 제거

3. **압축 고려**
큰 메시지(> 1KB)는 압축 적용

### 비대칭 트래픽 (송신 >> 수신)

정상적인 현상입니다:
- 하나의 메시지를 여러 클라이언트에게 브로드캐스트하므로
- 송신 데이터가 수신 데이터보다 많음
- 평균 비율: 송신 / 수신 ≈ 방 평균 인원 수

## 🎯 모니터링 알람 설정

### 1. 대역폭 임계값

```bash
# 초당 1MB 이상 송신 시 알람
if [ $sent_kb_per_sec -gt 1024 ]; then
    echo "High bandwidth usage alert!"
fi
```

### 2. Grafana 대시보드

```promql
# Prometheus 쿼리 예시
rate(bandwidth_sent_bytes_total[1m])
rate(bandwidth_received_bytes_total[1m])
```

## 📝 요약

- ✅ 실시간 대역폭 모니터링
- ✅ KB 단위로 표시 (읽기 쉬움)
- ✅ 초당 처리량 추적
- ✅ REST API로 쉽게 조회
- ✅ Atomic 연산으로 정확한 추적
- ✅ 모든 메시지 타입 추적 (시그널링, 채팅, Ping 등)

---

**참고**: 실제 비디오/오디오 데이터는 P2P로 직접 전송되므로 서버를 거치지 않습니다. 서버는 시그널링과 채팅 메시지만 처리합니다.

