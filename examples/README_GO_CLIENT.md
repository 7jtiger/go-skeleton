# Go P2P 화상채팅 테스트 클라이언트

터미널에서 실행 가능한 Go 기반 WebRTC P2P 화상채팅 테스트 클라이언트입니다.

## 📋 개요

이 클라이언트는 MS-Gateway 시그널링 서버를 통해 P2P 연결을 설정하고,
DataChannel을 사용하여 실시간 텍스트 메시지를 주고받을 수 있습니다.

### 특징

- ✅ WebSocket 기반 시그널링
- ✅ WebRTC PeerConnection
- ✅ DataChannel을 통한 P2P 메시지 전송
- ✅ 터미널 기반 인터페이스
- ✅ 다중 사용자 지원
- ✅ 실시간 연결 상태 모니터링

## 🚀 빠른 시작

### 1. 의존성 설치

```bash
cd examples
go mod download
```

### 2. 서버 시작

별도 터미널에서 MS-Gateway 서버를 시작합니다:

```bash
cd /home/jino/go/src/ms-gateway
./ms-gateway
```

### 3. 클라이언트 실행

#### 첫 번째 사용자 (user1)

```bash
cd examples
go run video_chat_client.go user1
```

#### 두 번째 사용자 (user2)

새 터미널 창을 열고:

```bash
cd examples
go run video_chat_client.go user2
```

## 💬 사용 방법

### 명령어

클라이언트 실행 후 다음 명령어를 사용할 수 있습니다:

#### 1. 통화 걸기

```bash
> call user2
```

다른 사용자에게 P2P 연결을 시작합니다.

#### 2. 메시지 전송

```bash
> send user2 안녕하세요!
```

연결된 사용자에게 텍스트 메시지를 전송합니다.

#### 3. 통화 종료

```bash
> end user2
```

특정 사용자와의 연결을 종료합니다.

#### 4. 도움말

```bash
> help
```

사용 가능한 명령어 목록을 표시합니다.

#### 5. 종료

```bash
> quit
```

클라이언트를 종료합니다.

## 📝 사용 예시

### 시나리오: user1과 user2가 대화하기

**터미널 1 (user1):**
```bash
$ go run video_chat_client.go user1
🔌 시그널링 서버 연결 중: ws://localhost:8080/webrtc/v01/ws?userId=user1
✅ 시그널링 서버 연결됨: user1

========================================
🎥 MS-Gateway P2P 화상채팅 클라이언트
========================================
명령어:
  call <userID>         - 사용자에게 통화 걸기
  send <userID> <msg>   - 메시지 전송
  end <userID>          - 통화 종료
  help                  - 도움말 표시
  quit                  - 종료
========================================

👥 접속 중인 사용자 (2명):
  1. user2

> call user2
📞 통화 시작: user2
📞 통화 중... user2
🔗 연결 상태 (user2): connecting
🧊 ICE 상태 (user2): checking
🔗 연결 상태 (user2): connected
✅ P2P 연결 성공: user2
💬 DataChannel 연결됨: user2
   이제 'send <userID> <message>' 명령으로 메시지를 보낼 수 있습니다.

> send user2 안녕하세요!

> 💬 [user2] 반갑습니다!

> send user2 테스트 메시지입니다.

> end user2
📴 통화 종료: user2

> quit

👋 종료 중...
```

**터미널 2 (user2):**
```bash
$ go run video_chat_client.go user2
🔌 시그널링 서버 연결 중: ws://localhost:8080/webrtc/v01/ws?userId=user2
✅ 시그널링 서버 연결됨: user2

========================================
🎥 MS-Gateway P2P 화상채팅 클라이언트
========================================

👥 접속 중인 사용자 (2명):
  1. user1

📨 Offer 수신: user1
🔗 연결 상태 (user1): connecting
🧊 ICE 상태 (user1): checking
🔗 연결 상태 (user1): connected
✅ P2P 연결 성공: user1
💬 DataChannel 연결됨: user1
   이제 'send <userID> <message>' 명령으로 메시지를 보낼 수 있습니다.

💬 [user1] 안녕하세요!

> send user1 반갑습니다!

💬 [user1] 테스트 메시지입니다.

> 📴 통화 종료됨: user1
```

## 🏗️ 아키텍처

### 컴포넌트

```
VideoChatClient
├── WebSocket 연결 (시그널링)
├── PeerConnection (WebRTC)
│   ├── ICE Candidate 수집
│   ├── SDP 교환 (Offer/Answer)
│   └── DataChannel (P2P 메시지)
└── 상태 관리
    ├── 사용자 목록
    ├── PeerConnection 맵
    └── DataChannel 맵
```

### 시그널링 흐름

```
user1                   Server                   user2
  │                       │                        │
  ├─── Connect ──────────>│                        │
  │                       ├─── user-list ─────────>│
  │                       │                        │
  ├─── Offer ────────────>│                        │
  │                       ├─── Offer ─────────────>│
  │                       │                        │
  │                       │<─── Answer ────────────┤
  │<─── Answer ───────────┤                        │
  │                       │                        │
  ├─── ICE Candidate ────>│                        │
  │                       ├─── ICE Candidate ─────>│
  │                       │                        │
  │<══════════ P2P DataChannel ══════════════════>│
  │                       │                        │
```

## 🔧 기술 스택

### Go 패키지

- **gorilla/websocket**: WebSocket 클라이언트
- **pion/webrtc**: WebRTC 구현 (PeerConnection, DataChannel)

### WebRTC 구성

- **ICE 서버**: Google STUN 서버 (stun.l.google.com:19302)
- **DataChannel**: 텍스트 메시지 전송용
- **연결 타입**: P2P (Peer-to-Peer)

## 📊 로그 아이콘 의미

- 🔌 연결/종료
- ✅ 성공
- ❌ 오류
- 📞 통화 시작
- 📨 메시지 수신
- 📴 통화 종료
- 💬 DataChannel 메시지
- 👥 사용자 목록
- 🔗 연결 상태
- 🧊 ICE 상태
- ⚠️  경고
- 👋 종료

## 🧪 테스트 시나리오

### 1. 기본 P2P 연결 테스트

```bash
# 터미널 1
go run video_chat_client.go alice

# 터미널 2
go run video_chat_client.go bob

# 터미널 1에서
> call bob

# 연결 확인 후 메시지 전송
> send bob Hello from Alice!
```

### 2. 다중 사용자 테스트

```bash
# 3개의 터미널에서 각각 실행
go run video_chat_client.go user1
go run video_chat_client.go user2
go run video_chat_client.go user3

# user1이 user2, user3와 동시에 연결
> call user2
> call user3
> send user2 Message to user2
> send user3 Message to user3
```

### 3. 재연결 테스트

```bash
# user1 연결
> call user2

# user2가 종료되었다가 다시 연결
> call user2  # 재연결 시도
```

## 🔍 디버깅

### 연결 상태 확인

로그에서 다음 상태를 확인하세요:

```
🔗 연결 상태 (user2): new
🔗 연결 상태 (user2): connecting
🔗 연결 상태 (user2): connected  ← 성공!
```

### ICE 연결 상태 확인

```
🧊 ICE 상태 (user2): checking
🧊 ICE 상태 (user2): connected  ← ICE 성공!
```

### DataChannel 상태 확인

```
💬 DataChannel 연결됨: user2  ← 메시지 전송 가능
```

## ⚠️ 문제 해결

### 1. 연결 실패

**증상:** `connection failed` 또는 타임아웃

**해결:**
- 서버가 실행 중인지 확인
- 방화벽 설정 확인
- STUN 서버 연결 확인

### 2. DataChannel이 열리지 않음

**증상:** 메시지 전송 시 `DataChannel이 열려있지 않음` 오류

**해결:**
- P2P 연결이 완료될 때까지 대기
- `✅ P2P 연결 성공` 로그 확인 후 전송

### 3. ICE 연결 실패

**증상:** `ICE 상태: failed`

**해결:**
- NAT/방화벽 환경 확인
- TURN 서버 추가 고려
- 동일 네트워크에서 테스트

### 4. 메시지가 전송되지 않음

**체크리스트:**
1. PeerConnection이 연결되었는가?
2. DataChannel이 열렸는가?
3. 올바른 userID를 사용했는가?

## 🚀 고급 사용법

### 커스텀 시그널링 서버 URL

```bash
go run video_chat_client.go myuser ws://custom-server:8080/webrtc/v01/ws
```

### 환경 변수 사용

```bash
export SIGNALING_URL="ws://production-server:8080/webrtc/v01/ws"
go run video_chat_client.go myuser $SIGNALING_URL
```

## 📦 빌드 및 배포

### 실행 파일 생성

```bash
cd examples
go build -o video-chat-client video_chat_client.go
```

### 실행

```bash
./video-chat-client user1
```

### 크로스 컴파일

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o video-chat-client-linux video_chat_client.go

# Windows
GOOS=windows GOARCH=amd64 go build -o video-chat-client.exe video_chat_client.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o video-chat-client-mac video_chat_client.go
```

## 🔮 향후 개선 사항

- [ ] 오디오/비디오 스트림 지원
- [ ] 파일 전송 기능
- [ ] 그룹 채팅 지원
- [ ] TUI (Text User Interface) 개선
- [ ] 설정 파일 지원
- [ ] 로그 레벨 조정
- [ ] 통계 정보 표시

## 📚 참고 자료

- [Pion WebRTC](https://github.com/pion/webrtc)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [WebRTC 표준](https://www.w3.org/TR/webrtc/)

## 📄 라이선스

이 프로젝트는 MS-Gateway의 일부입니다.

---

**최종 업데이트:** 2025-10-20
**작성자:** MS-Gateway Team

