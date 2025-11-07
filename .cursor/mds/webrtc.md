# WebRTC Package Function Feature Summary

## Overview
WebRTC (Web Real-Time Communication) 기능을 지원하는 패키지 모음입니다. P2P 비디오/오디오 통화, 시그널링 서버, STUN 서버 관리, 그리고 연결성 테스트 기능을 제공합니다.

**Note**: TURN 서버 지원은 릴레이 트래픽 비용 문제로 제거되었습니다. Google STUN 서버만을 사용하여 NAT 통과를 처리합니다.

---

## controller/webrtc.go - WebRTC Connectivity Testing

### STUN Testing Data Structures

#### STUNTestResult struct
- `Success bool`: STUN 테스트 성공 여부
- `ServerURL string`: 테스트한 STUN 서버 URL
- `PublicIP string`: 검색된 공인 IP 주소
- `PublicPort int`: 검색된 공인 포트 번호
- `ResponseTime int64`: 응답 시간 (밀리초)
- `Error string`: 오류 메시지 (실패 시)

#### ICEConnectivityTest struct
- `STUNServers []STUNTestResult`: 각 STUN 서버 테스트 결과 배열
- `Summary.TotalTested int`: 테스트한 총 서버 수
- `Summary.Successful int`: 성공한 서버 수
- `Summary.Failed int`: 실패한 서버 수

---

### STUN Server Testing Functions

#### TestStunServer(stunURL string, timeout time.Duration) STUNTestResult
STUN 서버 연결성을 테스트하고 공인 IP/포트를 반환합니다.

**동작 과정**:
1. STUN URL 파싱 (stun:host:port)
2. UDP 연결 생성
3. STUN Binding Request 패킷 생성 및 전송
4. 응답 패킷 수신 및 파싱
5. 공인 IP/포트 추출

**반환값**: 테스트 결과 (성공 여부, 공인 IP/포트, 응답 시간)

#### TestMultipleStunServers(stunURLs []string, timeout time.Duration) ICEConnectivityTest
여러 STUN 서버를 동시에 테스트합니다.

**특징**:
- 고루틴을 사용한 동시 테스트
- 채널을 통한 결과 수집
- 요약 통계 자동 계산

**반환값**: 전체 테스트 결과 및 요약 통계

---

### STUN Protocol Implementation Functions

#### parseStunURL(stunURL string) (host string, port int, err error)
STUN URL을 파싱하여 호스트와 포트를 추출합니다.

**지원 형식**: `stun:host:port`
**기본 포트**: 19302

#### createStunBindingRequest() ([]byte, []byte, error)
RFC 5389 표준을 따르는 STUN Binding Request 패킷을 생성합니다.

**패킷 구조**:
- Message Type: 0x0001 (Binding Request)
- Message Length: 0x0000
- Magic Cookie: 0x2112A442
- Transaction ID: 12바이트 랜덤

#### parseStunResponse(response []byte, expectedTransactionID []byte) (publicIP string, publicPort int, err error)
STUN 응답 패킷을 파싱하여 공인 IP와 포트를 추출합니다.

**처리 항목**:
- Transaction ID 검증
- Message Type 확인 (0x0101)
- MAPPED-ADDRESS (0x0001) 또는 XOR-MAPPED-ADDRESS (0x0020) 파싱
- XOR 디코딩 (XOR-MAPPED-ADDRESS의 경우)

---

### WebRTC Configuration Functions

#### GetRecommendedStunServers() []string
권장 STUN 서버 목록을 반환합니다.

**반환 서버**:
- stun:stun.l.google.com:19302
- stun:stun1.l.google.com:19302
- stun:stun2.l.google.com:19302
- stun:stun3.l.google.com:19302
- stun:stun4.l.google.com:19302

#### ValidateWebRTCConfig(config map[string]interface{}) error
WebRTC 설정의 유효성을 검증합니다.

**검증 항목**:
- ICE 서버 존재 여부
- 각 ICE 서버의 URLs 필드
- URL 형식 (stun: 또는 turn: 접두사)

#### CreateStunTestReport(ctx context.Context, stunServers []string) (map[string]interface{}, error)
포괄적인 STUN 테스트 보고서를 생성합니다.

**보고서 내용**:
- 타임스탬프
- 요약 통계 (총 테스트 수, 성공/실패 수, 성공률)
- 각 서버별 상세 결과 (URL, 성공 여부, 응답 시간, 공인 IP/포트)

---

## controller/signaling.go - WebRTC Signaling Server

### Core Data Structures

#### SignalingController struct
- `ctl *Controller`: 메인 컨트롤러 참조
- `cfg *conf.Config`: 설정 정보
- `rep *models.Repositories`: 데이터베이스 리포지토리
- `clients map[string]*Client`: 연결된 클라이언트 맵 (userId → Client)
- `clientsMux sync.RWMutex`: 클라이언트 맵 동기화용 뮤텍스
- `upgrader websocket.Upgrader`: WebSocket 업그레이더

#### Client struct
- `UserID string`: 사용자 ID
- `Conn *websocket.Conn`: WebSocket 연결
- `Send chan []byte`: 송신 메시지 채널 (버퍼 크기: 256)

#### SignalingMessage struct
- `Type string`: 메시지 타입
- `From string`: 발신자 ID
- `To string`: 수신자 ID
- `Payload interface{}`: 메시지 데이터

---

### WebSocket Connection Management

#### NewSignalingController(ctl *Controller, rep *models.Repositories) (*SignalingController, error)
시그널링 컨트롤러를 초기화합니다.

**초기화 항목**:
- WebSocket Upgrader 설정 (CORS 허용)
- 클라이언트 맵 초기화
- 컨트롤러 및 리포지토리 연결

#### HandleWebSocket(c *gin.Context)
WebSocket 연결 요청을 처리합니다.

**처리 과정**:
1. userId 쿼리 파라미터 검증
2. WebSocket 프로토콜로 업그레이드
3. 클라이언트 객체 생성 및 등록
4. 읽기/쓰기 고루틴 시작

#### registerClient(client *Client)
새 클라이언트를 활성 연결 맵에 등록합니다.

**동작**:
- 기존 연결이 있으면 종료 후 재등록
- 클라이언트 맵에 추가 (Thread-safe)
- 모든 클라이언트에게 사용자 목록 브로드캐스트

#### unregisterClient(client *Client)
클라이언트를 연결 맵에서 제거합니다.

**동작**:
- 클라이언트 맵에서 제거 (Thread-safe)
- 송신 채널 종료
- 모든 클라이언트에게 사용자 목록 브로드캐스트

---

### Message Processing

#### readPump(client *Client)
WebSocket으로부터 메시지를 읽는 고루틴입니다.

**기능**:
- Pong 핸들러 설정 (60초 타임아웃)
- JSON 메시지 파싱
- 발신자 정보 자동 설정
- 메시지 타입별 처리 위임
- 연결 종료 시 클라이언트 제거

#### writePump(client *Client)
WebSocket으로 메시지를 쓰는 고루틴입니다.

**기능**:
- 54초마다 Ping 메시지 자동 전송
- 송신 채널의 메시지 일괄 전송
- 연결 타임아웃 관리 (10초)
- 채널 종료 감지

#### handleMessage(client *Client, msg *SignalingMessage)
수신된 시그널링 메시지를 처리합니다.

**메시지 타입**:
- `offer`: WebRTC Offer 중계
- `answer`: WebRTC Answer 중계
- `ice-candidate`: ICE Candidate 중계
- `hangup`: 통화 종료 신호 중계

---

### Signaling Relay

#### relayMessage(from *Client, msg *SignalingMessage)
시그널링 메시지를 대상 사용자에게 중계합니다.

**동작**:
1. 대상 사용자 존재 여부 확인
2. 존재하지 않으면 발신자에게 에러 메시지 전송
3. 메시지를 대상 사용자에게 전달
4. 중계 로깅

#### sendToClient(client *Client, msg *SignalingMessage)
특정 클라이언트에게 메시지를 전송합니다.

**동작**:
1. 메시지를 JSON으로 직렬화
2. 송신 채널로 메시지 전달
3. 채널이 가득 차면 클라이언트 제거

---

### User List Management

#### broadcastUserList()
현재 연결된 모든 사용자 목록을 브로드캐스트합니다.

**동작**:
- 연결된 사용자 ID 리스트 생성
- `user-list` 타입 메시지로 모든 클라이언트에게 전송
- 로깅

#### GetConnectedUsers(c *gin.Context)
HTTP API로 현재 연결된 사용자 목록을 조회합니다.

**응답 형식**:
```json
{
  "users": ["user1", "user2"],
  "count": 2
}
```

---

## controller/account.go - WebRTC Session Management

### WebRTC Configuration

#### GetWebRTCConfig(c *gin.Context)
인증된 사용자의 WebRTC 설정을 조회합니다.

**반환 정보**:
- ICE 서버 목록 (STUN only - Google STUN servers)
- 미디어 설정 (비디오/오디오 품질)
- 세션 타임아웃 및 하트비트 간격

**데이터 소스**: Redis (설정 캐싱)

---

### User Availability

#### GetAvailableUsers(c *gin.Context)
통화 가능한 사용자 목록을 조회합니다.

**필터링**:
- 현재 통화 중이 아닌 사용자
- 자기 자신 제외
- 활성 WebRTC 세션이 있는 사용자

**반환값**: 사용자 목록 (ID, 닉네임, 프로필 사진 등)

---

### Call Status Management

#### UpdateCallStatus(c *gin.Context)
사용자의 통화 상태를 업데이트합니다.

**상태 종류**:
- `in_call`: 통화 중
- `available`: 통화 가능
- `busy`: 다른 작업 중

**추가 정보**: 통화 상대방 ID (선택적)

---

### Session Heartbeat

#### UpdateWebRTCHeartbeat(c *gin.Context)
WebRTC 세션 하트비트를 업데이트합니다.

**목적**: 활성 세션 유지 및 비활성 세션 정리
**업데이트 주기**: 클라이언트에서 주기적으로 호출 (권장: 30초)

---

### Statistics

#### GetWebRTCStats(c *gin.Context)
WebRTC 시스템 통계를 조회합니다.

**통계 정보**:
- 활성 세션 수
- 진행 중인 통화 수
- 활성 사용자 수

---

### STUN Server Testing

#### TestStunServers(c *gin.Context)
권장 STUN 서버들의 연결성을 테스트합니다.

**응답 형식**:
```json
{
  "summary": {
    "totalTested": 5,
    "successful": 5,
    "failed": 0,
    "successRate": 100.0
  },
  "details": [
    {
      "serverUrl": "stun:stun.l.google.com:19302",
      "success": true,
      "publicIp": "1.2.3.4",
      "publicPort": 12345,
      "responseTime": 123
    }
  ]
}
```

#### TestSpecificStunServer(c *gin.Context)
특정 STUN 서버의 연결성을 테스트합니다.

**요청 형식**:
```json
{
  "stunUrl": "stun:stun.example.com:19302"
}
```

---

## models/redis_db.go - WebRTC Session Storage

### Session Management

#### SetUserWebRTCSession(userID string, config map[string]interface{}) error
사용자 WebRTC 세션 정보를 저장합니다.

**저장 정보**:
- 사용자 ID
- WebRTC 설정
- 마지막 활성 시간
- 통화 상태

**키 형식**: `WEBRTC:SESSION:{userID}`
**TTL**: 30분 (하트비트로 갱신)

#### GetUserWebRTCSession(userID string) (map[string]interface{}, error)
사용자 WebRTC 세션 정보를 조회합니다.

**반환값**: 세션 정보 맵 또는 nil (세션 없음)

---

### User Status

#### UpdateUserCallStatus(userID, status string, partnerID string) error
사용자의 통화 상태를 업데이트합니다.

**상태 필드**:
- `callStatus`: 통화 상태 (available/in_call/busy)
- `callPartner`: 통화 상대방 ID (선택적)
- `lastUpdate`: 마지막 업데이트 시간

#### GetAvailableUsersForCall(excludeUserID string) ([]string, error)
통화 가능한 사용자 목록을 조회합니다.

**필터링 로직**:
- callStatus가 "available"인 사용자
- 제외 사용자 ID (자기 자신)
- 활성 WebRTC 세션이 있는 사용자

---

### Configuration

#### GetWebRTCConfig() map[string]interface{}
기본 WebRTC 설정을 반환합니다.

**설정 항목**:
- ICE 서버 (STUN only - Google STUN servers)
- 미디어 제약 조건
- 세션 타임아웃
- 하트비트 간격

**Note**: 설정 파일 기반으로 변경 가능. TURN 서버는 트래픽 비용 문제로 제외됨.

---

### Cleanup

#### CleanupInactiveSessions() error
비활성 WebRTC 세션을 정리합니다.

**정리 대상**: 30분 이상 하트비트가 없는 세션
**실행 주기**: 스케줄러에서 주기적 실행 (권장: 10분)

---

### Statistics

#### GetWebRTCStats() (map[string]interface{}, error)
WebRTC 시스템 통계를 조회합니다.

**통계 항목**:
- `activeSessions`: 활성 세션 수
- `activeUsers`: 활성 사용자 수
- `ongoingCalls`: 진행 중인 통화 수
- `availableUsers`: 통화 가능한 사용자 수

---

## ~~common/utils/turn.go - TURN Server Management~~ (REMOVED)

**TURN 서버 지원이 제거되었습니다.**

**제거 이유**: 
- TURN 서버는 릴레이 트래픽을 발생시켜 서버 대역폭 비용이 증가
- 대부분의 NAT 환경에서 STUN만으로 충분히 P2P 연결 가능
- 프로젝트 복잡도 감소 및 유지보수 부담 경감

**대안**:
- Google STUN 서버를 통한 NAT 통과
- 직접 P2P 연결이 불가능한 극소수 환경에서는 연결 실패 처리

---

## API Endpoints Summary

### WebRTC Configuration
- `GET /webrtc/v01/config` - WebRTC 설정 조회 (JWT 필요)

### User Management
- `GET /webrtc/v01/available-users` - 통화 가능 사용자 목록 (JWT 필요)
- `POST /webrtc/v01/call-status` - 통화 상태 업데이트 (JWT 필요)
- `POST /webrtc/v01/heartbeat` - 세션 하트비트 (JWT 필요)

### Statistics
- `GET /webrtc/v01/stats` - WebRTC 시스템 통계 (JWT 필요)

### STUN Testing
- `GET /webrtc/v01/test-stun` - 권장 STUN 서버 테스트 (JWT 필요)
- `POST /webrtc/v01/test-stun-server` - 특정 STUN 서버 테스트 (JWT 필요)

### WebSocket Signaling
- `GET /webrtc/v01/ws?userId={userId}` - WebSocket 시그널링 연결 (개발: 인증 없음, 프로덕션: JWT 권장)
- `GET /webrtc/v01/connected-users` - 연결된 사용자 목록 (JWT 필요)

---

## WebSocket Signaling Protocol

### Message Types

#### Client → Server
```json
{
  "type": "offer|answer|ice-candidate|hangup",
  "to": "target_user_id",
  "payload": {}
}
```

#### Server → Client
```json
{
  "type": "offer|answer|ice-candidate|hangup|user-list|error",
  "from": "sender_user_id",
  "payload": {}
}
```

### Message Type Details

#### offer
WebRTC Offer SDP를 대상 사용자에게 전달합니다.
```json
{
  "type": "offer",
  "to": "bob",
  "payload": {
    "sdp": "v=0\r\no=- ...",
    "type": "offer"
  }
}
```

#### answer
WebRTC Answer SDP를 대상 사용자에게 전달합니다.
```json
{
  "type": "answer",
  "to": "alice",
  "payload": {
    "sdp": "v=0\r\no=- ...",
    "type": "answer"
  }
}
```

#### ice-candidate
ICE Candidate 정보를 대상 사용자에게 전달합니다.
```json
{
  "type": "ice-candidate",
  "to": "bob",
  "payload": {
    "candidate": "candidate:...",
    "sdpMid": "0",
    "sdpMLineIndex": 0
  }
}
```

#### hangup
통화 종료 신호를 전달합니다.
```json
{
  "type": "hangup",
  "to": "bob"
}
```

#### user-list (Server → Client only)
연결된 사용자 목록을 클라이언트에게 브로드캐스트합니다.
```json
{
  "type": "user-list",
  "payload": ["alice", "bob", "charlie"]
}
```

#### error (Server → Client only)
오류 메시지를 클라이언트에게 전송합니다.
```json
{
  "type": "error",
  "payload": {
    "message": "User not found"
  }
}
```

---

## Configuration (config.toml)

### WebRTC Section (STUN only)
```toml
[webrtc]
signalingServerUrl = "ws://localhost:8080/ws"
stunServers = [
    "stun:stun.l.google.com:19302",
    "stun:stun1.l.google.com:19302",
    "stun:stun2.l.google.com:19302",
    "stun:stun3.l.google.com:19302",
    "stun:stun4.l.google.com:19302"
]
maxVideoWidth = 1080
maxVideoHeight = 1920
maxVideoFrameRate = 30
videoBitrate = 8000000
audioBitrate = 128000
sessionTimeout = 14400  # 4 hours
heartbeatInterval = 30
```

**Note**: TURN 서버 설정이 제거되었습니다. Google STUN 서버만 사용합니다.

---

## Connection Flow

### WebRTC P2P Connection Establishment

1. **WebSocket 연결**
   - Client A: `ws://server/webrtc/v01/ws?userId=alice`
   - Client B: `ws://server/webrtc/v01/ws?userId=bob`

2. **Offer 생성 및 전송 (Alice → Bob)**
   ```javascript
   const offer = await peerConnection.createOffer();
   await peerConnection.setLocalDescription(offer);
   ws.send(JSON.stringify({
     type: 'offer',
     to: 'bob',
     payload: offer
   }));
   ```

3. **Answer 생성 및 전송 (Bob → Alice)**
   ```javascript
   await peerConnection.setRemoteDescription(receivedOffer);
   const answer = await peerConnection.createAnswer();
   await peerConnection.setLocalDescription(answer);
   ws.send(JSON.stringify({
     type: 'answer',
     to: 'alice',
     payload: answer
   }));
   ```

4. **ICE Candidate 교환**
   ```javascript
   peerConnection.onicecandidate = (event) => {
     if (event.candidate) {
       ws.send(JSON.stringify({
         type: 'ice-candidate',
         to: 'bob',
         payload: event.candidate
       }));
     }
   };
   ```

5. **P2P 연결 완료**
   - STUN 서버를 통한 NAT traversal
   - 직접 P2P 연결 수립 (TURN 릴레이 없음)

---

## Performance Considerations

### WebSocket Connection
- **Ping/Pong**: 54초 주기 Ping, 60초 타임아웃
- **송신 버퍼**: 256 메시지
- **동시 연결**: RWMutex로 Thread-safe 보장

### STUN Server Testing
- **타임아웃**: 5초 (권장)
- **동시 테스트**: 고루틴 기반 병렬 처리
- **응답 시간**: 밀리초 단위 측정

### Redis Session
- **TTL**: 30분 (하트비트로 갱신)
- **정리 주기**: 10분 (스케줄러)
- **키 형식**: `WEBRTC:SESSION:{userID}`

---

## Security Considerations

### Current Implementation (Development)
- WebSocket: 인증 없음 (userId 쿼리 파라미터)
- CORS: 모든 Origin 허용
- NAT Traversal: Google STUN 서버만 사용

### Production Recommendations
- **WebSocket 인증**: JWT 토큰 검증 필수
- **CORS 제한**: 허용된 Origin만 접근
- **Rate Limiting**: 메시지 전송 빈도 제한
- **사용자 검증**: DB에서 실제 사용자 확인
- **TLS/SSL**: wss:// 사용 (암호화된 WebSocket)
- **연결 실패 처리**: STUN으로 연결 불가 시 사용자에게 안내

---

## Monitoring and Debugging

### Logging
- WebSocket 연결/종료
- 시그널링 메시지 중계
- STUN 테스트 결과

### Metrics
- 활성 WebSocket 연결 수
- 시그널링 메시지 처리율
- STUN 성공률
- P2P 연결 성공/실패율

### Health Checks
- WebSocket 연결 상태
- STUN 서버 연결성
- Redis 세션 상태

---

## Test Clients

### 1. DataChannel Text Client
**파일**: `examples/video_chat_client.go`

**기능**:
- WebSocket 시그널링
- WebRTC DataChannel P2P 메시지
- 터미널 채팅 인터페이스

**사용법**:
```bash
cd examples
go build -o video-chat-client video_chat_client.go
./video-chat-client alice
```

### 2. Audio/Video Streaming Client
**파일**: `examples/video_chat_av_client.go`

**기능**:
- 실제 오디오/비디오 스트림
- VP8 비디오 + Opus 오디오
- IVF/OGG 파일 지원
- RTCP 피드백 및 통계

**사용법**:
```bash
cd examples
go build -o video-chat-av video_chat_av_client.go
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \
    test_media/test_video.ivf test_media/test_audio.ogg
```

---

## Troubleshooting

### WebSocket Connection Failed
- CORS 설정 확인
- WebSocket URL 확인 (ws:// 또는 wss://)
- 방화벽 설정 확인

### STUN Server Test Failed
- 네트워크 방화벽 확인 (UDP 19302 포트)
- STUN 서버 URL 형식 확인
- 타임아웃 설정 증가

### P2P Connection Failed
- STUN 서버 연결 확인
- ICE Candidate 교환 확인
- 네트워크 방화벽 설정 확인
- **Note**: TURN 서버가 없으므로 Symmetric NAT 환경에서는 연결 불가능할 수 있음

---

*Created: 2025-11-05*
*Last Updated: 2025-11-07*
*File Location: /home/jino/go/src/ms-gateway/controller/webrtc.go, controller/signaling.go, models/redis_db.go*
*Note: TURN server support removed on 2025-11-07 to avoid relay traffic overhead. Uses Google STUN servers only.*

