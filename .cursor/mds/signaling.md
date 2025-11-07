# Signaling Controller Function Feature Summary

## signaling.go - WebRTC 시그널링 컨트롤러

### 핵심 데이터 구조

#### SignalingController struct
- `ctl *Controller`: 메인 컨트롤러 참조
- `cfg *conf.Config`: 설정 정보
- `rep *models.Repositories`: 데이터베이스 리포지토리
- `clients map[string]*Client`: 연결된 클라이언트 맵 (userId -> Client)
- `clientsMux sync.RWMutex`: 클라이언트 맵 동기화용 뮤텍스
- `upgrader websocket.Upgrader`: WebSocket 업그레이더

#### Client struct
- `UserID string`: 사용자 ID
- `Conn *websocket.Conn`: WebSocket 연결
- `Send chan []byte`: 송신 메시지 채널

#### SignalingMessage struct
- `Type string`: 메시지 타입 (offer/answer/ice-candidate/hangup/user-list/error)
- `From string`: 발신자 ID
- `To string`: 수신자 ID
- `Payload interface{}`: 메시지 데이터

---

## 컨트롤러 관리 함수

### NewSignalingController(ctl *Controller, rep *models.Repositories) (*SignalingController, error)
- 시그널링 컨트롤러 인스턴스 생성
- WebSocket Upgrader 초기화 (CORS 허용)
- 클라이언트 맵 초기화

---

## WebSocket 연결 관리 함수

### HandleWebSocket(c *gin.Context)
- WebSocket 연결 요청 처리
- `userId` 쿼리 파라미터 검증
- WebSocket 프로토콜로 업그레이드
- 클라이언트 객체 생성 및 등록
- 읽기/쓰기 고루틴 시작

### registerClient(client *Client)
- 새 클라이언트를 활성 클라이언트 맵에 등록
- 기존 연결이 있으면 종료 후 재등록
- 사용자 목록 브로드캐스트

### unregisterClient(client *Client)
- 클라이언트를 활성 맵에서 제거
- 송신 채널 종료
- 사용자 목록 브로드캐스트

---

## 메시지 처리 함수

### readPump(client *Client)
- WebSocket으로부터 메시지 읽기 (고루틴)
- Pong 핸들러 설정 (60초 타임아웃)
- JSON 메시지 파싱
- 발신자 정보 자동 설정
- 메시지 타입별 처리 위임

### writePump(client *Client)
- WebSocket으로 메시지 쓰기 (고루틴)
- 54초마다 Ping 메시지 자동 전송
- 송신 채널의 메시지 일괄 전송
- 연결 타임아웃 관리

### handleMessage(client *Client, msg *SignalingMessage)
- 수신된 시그널링 메시지 처리
- 메시지 타입별 분기 (offer/answer/ice-candidate/hangup)
- 중계 메시지를 대상 사용자에게 전달

---

## 시그널링 중계 함수

### relayMessage(from *Client, msg *SignalingMessage)
- 시그널링 메시지를 대상 사용자에게 중계
- 대상 사용자 존재 여부 확인
- 존재하지 않으면 발신자에게 에러 메시지 전송
- 메시지 전달 로깅

### sendToClient(client *Client, msg *SignalingMessage)
- 특정 클라이언트에게 메시지 전송
- JSON 직렬화
- 송신 채널로 메시지 전달
- 채널이 가득 차면 클라이언트 제거

---

## 사용자 목록 관리 함수

### broadcastUserList()
- 현재 연결된 모든 사용자 목록 브로드캐스트
- 사용자 ID 리스트 생성
- `user-list` 타입 메시지로 모든 클라이언트에게 전송

### GetConnectedUsers(c *gin.Context) (HTTP API)
- 현재 연결된 사용자 목록 조회 (REST API)
- 사용자 ID 배열과 총 개수 반환

---

## 시그널링 메시지 타입

### 클라이언트 → 서버
- `offer`: WebRTC Offer 메시지
- `answer`: WebRTC Answer 메시지
- `ice-candidate`: ICE Candidate 정보
- `hangup`: 통화 종료 신호

### 서버 → 클라이언트
- `user-list`: 연결된 사용자 목록 업데이트
- `offer`: 중계된 Offer 메시지
- `answer`: 중계된 Answer 메시지
- `ice-candidate`: 중계된 ICE Candidate
- `hangup`: 중계된 통화 종료 신호
- `error`: 오류 메시지

---

## 주요 기능

### WebSocket 연결 관리
- **자동 재연결 처리**: 기존 연결 종료 후 새 연결 수립
- **Ping/Pong 메커니즘**: 54초 주기 Ping, 60초 타임아웃
- **고루틴 기반**: 각 클라이언트당 읽기/쓰기 고루틴 분리

### 시그널링 중계
- **P2P 시그널링**: Offer/Answer/ICE Candidate 중계
- **사용자 검증**: 대상 사용자 존재 여부 확인
- **에러 처리**: 대상 사용자 미존재 시 에러 메시지 반환

### 동시성 제어
- **RWMutex 사용**: 클라이언트 맵 동시 접근 보호
- **Thread-Safe**: 다중 클라이언트 동시 연결 안전

### 사용자 목록 동기화
- **실시간 업데이트**: 연결/종료 시 즉시 브로드캐스트
- **자동 알림**: 모든 클라이언트에게 변경사항 전파

---

## 라우터 연동

### WebSocket 엔드포인트
```go
webrtc.GET("/ws", p.sig.HandleWebSocket)
```
- URL: `ws://server:port/webrtc/v01/ws?userId={userId}`
- 인증: 개발용으로 인증 없음 (프로덕션에서는 JWT 추천)

### HTTP 엔드포인트
```go
webrtc.GET("/connected-users", p.sig.GetConnectedUsers)
```
- URL: `http://server:port/webrtc/v01/connected-users`
- 응답: `{"users": ["user1", "user2"], "count": 2}`

---

## 사용 예시

### 클라이언트 연결
```javascript
const ws = new WebSocket('ws://localhost:8080/webrtc/v01/ws?userId=user1');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

### 시그널링 메시지 전송
```javascript
const message = {
  type: 'offer',
  to: 'user2',
  payload: offerSDP
};
ws.send(JSON.stringify(message));
```

---

## 보안 고려사항

### 현재 구현
- **CORS 전체 허용**: 개발 단계에서 모든 Origin 허용
- **인증 없음**: userId 쿼리 파라미터로만 식별

### 프로덕션 권장사항
- **JWT 인증**: WebSocket 연결 시 토큰 검증
- **CORS 제한**: 허용된 Origin만 접근 가능
- **Rate Limiting**: 메시지 전송 빈도 제한
- **사용자 검증**: DB에서 실제 사용자 존재 여부 확인

---

## 성능 최적화

### 메시지 버퍼링
- 송신 채널 버퍼 크기: 256
- 대기 중인 메시지 일괄 전송

### 연결 타임아웃
- Ping 주기: 54초
- Pong 타임아웃: 60초
- 쓰기 타임아웃: 10초

### 동시성
- 클라이언트별 독립 고루틴
- RWMutex로 읽기 동시성 극대화

---

## 로깅 및 모니터링

### 로그 메시지
- 새 연결: `New WebSocket connection: {userId}`
- 연결 종료: `Client disconnected: {userId}`
- 메시지 중계: `Message relayed: {type} from {from} to {to}`
- 사용자 목록: `User list broadcasted: {users}`

### 에러 로깅
- WebSocket 오류
- 메시지 파싱 오류
- JSON 직렬화 오류

---

## 테스트 클라이언트

### 1. DataChannel 텍스트 클라이언트
- **파일**: `examples/video_chat_client.go`
- **빌드**: `cd examples && go build -o video-chat-client video_chat_client.go`
- **실행**: `./video-chat-client <userId> [WebSocket URL]`
- **문서**: `examples/README_GO_CLIENT.md`, `examples/QUICKSTART.md`

**기능**:
- WebSocket 시그널링 연결
- WebRTC DataChannel P2P 메시지 전송
- 터미널 기반 채팅 인터페이스

**사용 예시**:
```bash
# 터미널 1
./video-chat-client alice

# 터미널 2
./video-chat-client bob

# alice에서
> call bob
> send bob Hello!
> end bob
```

### 2. 오디오/비디오 스트리밍 클라이언트
- **파일**: `examples/video_chat_av_client.go`
- **빌드**: `cd examples && go build -o video-chat-av video_chat_av_client.go`
- **실행**: `./video-chat-av <userId> [WebSocket URL] [video.ivf] [audio.ogg]`
- **문서**: `examples/README_AV_CLIENT.md`, `examples/QUICKSTART_AV.md`

**기능**:
- 실제 오디오/비디오 스트림 송수신
- VP8 비디오 + Opus 오디오 코덱
- IVF/OGG 파일 지원 또는 더미 스트림
- RTCP 피드백 및 통계
- 원격 트랙 수신 및 처리

**사용 예시**:
```bash
# 미디어 파일 생성
./generate_test_media.sh

# 터미널 1
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \
    test_media/test_video.ivf test_media/test_audio.ogg

# 터미널 2
./video-chat-av bob ws://localhost:8080/webrtc/v01/ws \
    test_media/colorbar_video.ivf test_media/melody_audio.ogg

# alice에서
> call bob
> stats bob
> end bob
```

### 미디어 파일 생성
```bash
cd examples
./generate_test_media.sh
```

자동으로 VP8 비디오(IVF)와 Opus 오디오(OGG) 테스트 파일을 생성합니다.

---

*생성일: 2025-10-20*
*최종 업데이트: 2025-11-05*
*파일 위치: /home/jino/go/src/ms-gateway/controller/signaling.go*

