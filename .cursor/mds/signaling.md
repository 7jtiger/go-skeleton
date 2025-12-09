# Signaling Controller Function Feature Summary

## signaling.go - WebRTC 시그널링 컨트롤러

### 핵심 데이터 구조

#### SignalingController struct
- `ctl *Controller`: 메인 컨트롤러 참조
- `cfg *conf.Config`: 설정 정보
- `rep *models.Repositories`: 데이터베이스 리포지토리
- `rooms map[string]*VDRoom`: 방 맵 (roomId -> VDRoom)
- `roomsMu sync.RWMutex`: 방 맵 동기화용 뮤텍스
- `waitingRoom *WaitingRoom`: 대기방 (통화 요청 전 대기하는 사용자들)
- `upgrader websocket.Upgrader`: WebSocket 업그레이더
- `messagePool sync.Pool`: 메시지 재사용을 위한 풀
- `totalConn int64`: 전체 연결 수 (atomic)
- `wrkQueue chan WorkItem`: 메시지 처리 워커 큐
- `brcQueue chan *BroadcastJob`: 브로드캐스트 작업 큐
- `ctx context.Context`: 컨텍스트
- `cancel context.CancelFunc`: 취소 함수

#### VDRoom struct
- `Name string`: 방 이름
- `ID int64`: 방 ID
- `Clients map[*WSClient]bool`: 클라이언트 맵
- `UserNames map[*WSClient]string`: 클라이언트 -> 사용자 이름 매핑
- `Broadcast chan *BroadcastMessage`: 브로드캐스트 채널
- `Unregister chan *WSClient`: 클라이언트 해제 채널
- `mu sync.RWMutex`: 동기화 뮤텍스
- `createdAt time.Time`: 생성 시간
- `lastActivity time.Time`: 마지막 활동 시간

#### WSClient struct
- `conn *websocket.Conn`: WebSocket 연결
- `send chan []byte`: 송신 버퍼
- `room *VDRoom`: 속한 방 (nil이면 대기방)
- `userName string`: 사용자 이름
- `userID string`: 사용자 ID
- `mu sync.Mutex`: 동기화 뮤텍스
- `lastSeen time.Time`: 마지막 활동 시간

#### WaitingRoom struct
- `Clients map[string]*WSClient`: userId -> WSClient 맵
- `mu sync.RWMutex`: 동기화 뮤텍스
- `createdAt time.Time`: 생성 시간
- `lastActivity time.Time`: 마지막 활동 시간

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
- 방 맵 및 대기방 초기화
- 메시지 풀, 워커 큐, 브로드캐스트 큐 초기화
- NUM_WORKERS(50)개의 메시지 처리 워커 고루틴 시작
- NUM_BRC_WORKERS(15)개의 브로드캐스트 워커 고루틴 시작
- 방 정리 고루틴(roomCleaner) 시작

---

## WebSocket 연결 관리 함수

### HandleConnection(c *gin.Context)
- WebSocket 연결 요청 처리 (최대 연결 수 제한: MAX_VDWS_CONNECT = 25000)
- WebSocket 프로토콜로 업그레이드
- 첫 메시지로 "join-waiting" 또는 "join" 메시지 대기 (10초 타임아웃)
- "join-waiting": 대기방에 추가, 사용자 목록 전송, 다른 사용자들에게 입장 알림
- "join": 기존 방식으로 직접 방 입장 (하위 호환성)
- 클라이언트 객체 생성 및 등록
- 읽기/쓰기 고루틴 시작

### 대기방 관련 함수
- `sendUserList(client *WSClient)`: 대기방의 사용자 목록을 클라이언트에게 전송 (자신 제외)
- `broadcastUserJoined(newClient *WSClient)`: 새 사용자 입장을 다른 사용자들에게 알림
- `broadcastUserLeft(leftClient *WSClient)`: 사용자 나감을 다른 사용자들에게 알림

---

## 메시지 처리 함수

### 워커 시스템
- `msgWorker()`: 메시지 처리 워커 (NUM_WORKERS = 50개)
- `brcWorker()`: 브로드캐스트 처리 워커 (NUM_BRC_WORKERS = 15개)
- `processMessage(client *WSClient, data []byte)`: 메시지 타입별 처리
  - 대기방 메시지: `handleWTRoom()` 호출
  - 방 내 메시지: offer/answer/ice_candidate는 브로드캐스트, chat는 채팅 메시지 처리

### readPump(client *WSClient)
- WebSocket으로부터 메시지 읽기 (고루틴)
- Pong 핸들러 설정 (PONG_WAIT = 60초 타임아웃)
- JSON 메시지 파싱
- 메시지를 워커 큐(wrkQueue)에 추가
- 연결 종료 시 대기방 또는 방에서 제거

### writePump(client *WSClient)
- WebSocket으로 메시지 쓰기 (고루틴)
- PING_PERIOD(54초)마다 Ping 메시지 자동 전송
- 송신 채널의 메시지 일괄 전송 (버퍼에 대기 중인 메시지 포함)
- 연결 타임아웃 관리 (WRITE_WAIT = 10초)

### 대기방 메시지 처리
- `handleWTRoom(client *WSClient, msg *Message)`: 대기방 메시지 처리
  - "join-waiting": 사용자 목록 전송
  - "call-request": 연결 요청 처리
  - "call-response": 연결 응답 처리 (수락 시 방으로 이동)
- `handleCallRequest(client *WSClient, msg *Message)`: 통화 요청 처리
  - 대상 사용자에게 call-request 메시지 전송
  - 대상 사용자가 없으면 에러 메시지 반환
- `handleCallResponse(client *WSClient, msg *Message)`: 통화 응답 처리
  - 수락 시 `moveToRoom()` 호출하여 두 사용자를 방으로 이동
  - 거절 시 요청자에게 거절 메시지 전송

---

## 방 관리 함수

### getRoom(roomName string) (*VDRoom, error)
- 방을 가져오거나 새로 생성
- 방 수 제한 확인 (MAX_VIDEO_ROOMS = 5000)
- 방 고루틴(run) 시작

### moveToRoom(roomID string, client1, client2 *WSClient)
- 두 사용자를 대기방에서 방으로 이동
- 대기방에서 제거
- 방에 추가 및 room-joined 메시지 전송
- start 시그널 전송 (화상채팅 시작)

### 방 실행 함수
- `room.run(p *SignalingController)`: 방 관리 고루틴
  - 클라이언트 해제 처리
  - 브로드캐스트 메시지 처리 (brcQueue에 추가)
  - PING_PERIOD마다 Ping 메시지 전송

### 방 정리 함수
- `roomCleaner()`: 주기적으로 비어있고 비활성인 방 정리 (1분마다)
- `cleanEmptyRooms()`: 5분 이상 비활성인 빈 방 삭제

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

### 대기방 메시지 (클라이언트 → 서버)
- `join-waiting`: 대기방 입장 (필수: name, user_id)
- `call-request`: 통화 요청 (from, to, room_id)
- `call-response`: 통화 응답 (from, to, room_id, accept)

### 대기방 메시지 (서버 → 클라이언트)
- `user-list`: 대기방 사용자 목록 (users 배열)
- `user-joined`: 새 사용자 입장 알림 (user_id, name)
- `user-left`: 사용자 나감 알림 (user_id)
- `call-request`: 통화 요청 수신 (from, to, room_id)
- `call-accept`: 통화 수락 (from, to, room_id, accept: true)
- `call-reject`: 통화 거절 (from, to, room_id, accept: false)
- `call-error`: 통화 요청 오류 (message)
- `room-joined`: 방 입장 완료 (room_id)

### 방 내 메시지 (클라이언트 → 서버)
- `offer`: WebRTC Offer 메시지
- `answer`: WebRTC Answer 메시지
- `ice_candidate`: ICE Candidate 정보
- `chat`: 채팅 메시지

### 방 내 메시지 (서버 → 클라이언트)
- `start`: 화상채팅 시작 신호
- `offer`: 중계된 Offer 메시지
- `answer`: 중계된 Answer 메시지
- `ice_candidate`: 중계된 ICE Candidate
- `chat`: 채팅 메시지 (sender, message)
- `ping`: 연결 유지 Ping 메시지

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
- **RWMutex 사용**: 방 맵, 대기방 동시 접근 보호
- **Thread-Safe**: 다중 클라이언트 동시 연결 안전
- **워커 풀**: 메시지 처리 및 브로드캐스트를 워커 풀로 분산 처리
- **Atomic 카운터**: 전체 연결 수 추적 (totalConn)

### 사용자 목록 동기화
- **실시간 업데이트**: 대기방 입장/나감 시 즉시 브로드캐스트
- **자동 알림**: 모든 클라이언트에게 변경사항 전파
- **방 이동**: 통화 수락 시 두 사용자를 대기방에서 방으로 자동 이동

---

## 라우터 연동

### WebSocket 엔드포인트
```go
webrtc.GET("/ws", p.sig.HandleConnection)
```
- URL: `ws://server:port/webrtc/v01/ws`
- 첫 메시지: `{"type":"join-waiting","data":{"name":"사용자명","user_id":"사용자ID"}}`
- 인증: 개발용으로 인증 없음 (프로덕션에서는 JWT 추천)
- 최대 연결 수: 25000

### HTTP 엔드포인트
```go
webrtc.GET("/connected-users", p.sig.GetConnectedUsers)
```
- URL: `http://server:port/webrtc/v01/connected-users`
- 응답: `{"users": ["user1", "user2"], "count": 2}`

---

## 사용 예시

### 클라이언트 연결 및 대기방 입장
```javascript
const ws = new WebSocket('ws://localhost:8080/webrtc/v01/ws');

// 연결 후 첫 메시지: 대기방 입장
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'join-waiting',
    data: {
      name: 'Alice',
      user_id: 'alice123'
    }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
  
  // 사용자 목록 수신
  if (message.type === 'user-list') {
    console.log('Available users:', message.data.users);
  }
  
  // 통화 요청 수신
  if (message.type === 'call-request') {
    // 통화 수락 또는 거절
    ws.send(JSON.stringify({
      type: 'call-response',
      data: {
        from: 'alice123',
        to: message.data.from,
        room_id: message.data.room_id,
        accept: true
      }
    }));
  }
  
  // 방 입장 완료
  if (message.type === 'room-joined') {
    console.log('Joined room:', message.data.room_id);
  }
  
  // 화상채팅 시작
  if (message.type === 'start') {
    console.log('Video chat started');
  }
};

// 통화 요청 전송
function callUser(targetUserId) {
  const roomId = `room_${Date.now()}`;
  ws.send(JSON.stringify({
    type: 'call-request',
    data: {
      from: 'alice123',
      to: targetUserId,
      room_id: roomId
    }
  }));
}

// WebRTC 시그널링 메시지 전송 (방 내)
function sendOffer(offerSDP) {
  ws.send(JSON.stringify({
    type: 'offer',
    data: {
      sdp: offerSDP
    }
  }));
}
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
- 송신 채널 버퍼 크기: MSG_BUFFER_SIZE = 256
- 워커 큐 크기: WORKER_QUEUE_SIZE = 1000
- 브로드캐스트 큐 크기: BROADCAST_QUEUE_SIZE = 10000
- 대기 중인 메시지 일괄 전송
- 메시지 풀을 통한 메모리 재사용

### 연결 타임아웃
- Ping 주기: PING_PERIOD = 54초
- Pong 타임아웃: PONG_WAIT = 60초
- 쓰기 타임아웃: WRITE_WAIT = 10초
- 첫 메시지 대기: 10초

### 동시성
- 클라이언트별 독립 고루틴 (readPump, writePump)
- 워커 풀: 50개 메시지 처리 워커, 15개 브로드캐스트 워커
- RWMutex로 읽기 동시성 극대화
- 방별 독립 고루틴 (room.run)

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

---

## 주요 변경사항 (최신 업데이트)

### 대기방 시스템 추가
- 통화 요청 전 사용자들이 대기하는 대기방(WaitingRoom) 구현
- 대기방 입장 시 사용자 목록 자동 전송
- 사용자 입장/나감 실시간 알림

### 통화 요청/응답 시스템
- `call-request`: 통화 요청 메시지
- `call-response`: 통화 수락/거절 응답
- 수락 시 자동으로 두 사용자를 방으로 이동

### 워커 풀 시스템
- 메시지 처리 워커: 50개
- 브로드캐스트 워커: 15개
- 큐 기반 비동기 처리로 성능 향상

### 방 관리 개선
- 방별 독립 고루틴 실행
- 비활성 방 자동 정리 (1분 주기)
- 방 입장 시 room-joined 및 start 시그널 자동 전송

---

*생성일: 2025-10-20*
*최종 업데이트: 2025-11-30*
*파일 위치: /home/jino/go/src/ms-gateway/controller/signaling.go*

