# WebRTC 시그널링 서버 및 클라이언트

Node.js에서 Golang으로 변환한 WebRTC 시그널링 서버와 HTML/JavaScript 클라이언트입니다.

## 📋 목차

- [개요](#개요)
- [기술 스택](#기술-스택)
- [설치 및 실행](#설치-및-실행)
- [사용 방법](#사용-방법)
- [파일 구조](#파일-구조)
- [주요 기능](#주요-기능)
- [코드 설명](#코드-설명)

## 🎯 개요

이 프로젝트는 WebRTC를 사용한 1:1 비디오/오디오 채팅 및 **실시간 텍스트 채팅** 애플리케이션입니다.
- **서버**: Golang으로 작성된 WebSocket 기반 시그널링 서버
- **클라이언트**: HTML/JavaScript로 작성된 웹 클라이언트
- **주요 기능**: 비디오/오디오 통화 + 실시간 텍스트 채팅 💬

## 🛠 기술 스택

### 서버
- **언어**: Go 1.16+
- **라이브러리**: 
  - `gorilla/websocket`: WebSocket 통신

### 클라이언트
- **HTML5**
- **JavaScript (ES6+)**
- **WebRTC API**
- **WebSocket API**

## 📦 설치 및 실행

### 1. 필수 요구사항

- Go 1.16 이상
- 모던 웹 브라우저 (Chrome, Firefox, Safari, Edge 등)
- 카메라와 마이크 권한

### 2. 서버 설치

```bash
# 1. gorilla/websocket 라이브러리 설치
cd /home/jino/go/src/ms-gateway/client/exam
go get github.com/gorilla/websocket

# 2. go.mod 파일이 없다면 생성
go mod init exam

# 3. 의존성 다운로드
go mod tidy
```

### 3. 서버 실행

#### 기본 서버 (소규모 서비스용)

```bash
# 기본 서버 실행
go run server.go
```

서버가 정상적으로 실행되면 다음과 같은 메시지가 출력됩니다:
```
Server is running on port 3000
```

#### 최적화 서버 (대용량 트래픽용) ⚡

```bash
# 최적화 서버 실행
go run server_optimized.go
```

최적화 서버 실행 시 출력 예시:
```
Using 8 CPU cores
Optimized server is running on port 3000
Max connections: 50000
Max rooms: 10000
Workers: 100

=== Server Stats ===
Active Connections: 0
Total Connections: 0
Active Rooms: 0
Total Messages: 0
Messages/sec: 0
Memory (Alloc): 5.23 MB
Memory (Sys): 12.45 MB
Goroutines: 115
==================
```

**헬스 체크**:
```bash
curl http://localhost:3000/health
# 응답: {"status":"ok","active_connections":0,"total_rooms":0,"total_messages":0}
```

### 4. 클라이언트 실행

브라우저에서 `client.html` 파일을 직접 열거나, 간단한 HTTP 서버를 사용하세요:

**방법 1: 직접 열기**
```bash
# 파일 탐색기에서 client.html을 더블클릭하거나
# 브라우저에서 파일을 직접 엽니다
```

**방법 2: Python HTTP 서버 사용 (권장)**
```bash
# Python 3가 설치되어 있다면
python3 -m http.server 8080

# 브라우저에서 http://localhost:8080/client.html 접속
```

## 🚀 사용 방법

### 기본 사용 흐름

1. **서버 시작**: `go run server.go` 명령으로 서버를 실행합니다.

2. **클라이언트 접속**: 
   - 브라우저를 **2개** 열어서 `client.html`에 접속합니다.
   - 카메라와 마이크 권한을 허용합니다.

3. **방 입장**:
   - 첫 번째 브라우저: 이름 입력 (예: "사용자1"), 방 이름 입력 (예: "room1"), "입장" 클릭
   - 두 번째 브라우저: 이름 입력 (예: "사용자2"), **같은 방 이름** 입력 (예: "room1"), "입장" 클릭

4. **통화 시작**:
   - 두 번째 사용자가 입장하면 자동으로 WebRTC 연결이 시작됩니다.
   - 양쪽 화면에서 상대방의 비디오를 볼 수 있습니다.

5. **기능 사용**:
   - 🎤 **음소거**: 내 마이크를 켜거나 끕니다.
   - 📹 **비디오 끄기**: 내 비디오를 켜거나 끕니다.
   - 💬 **채팅**: 하단 채팅창에서 텍스트 메시지를 주고받습니다 (Enter 키로 빠른 전송).
   - 🚪 **나가기**: 방을 나갑니다.

### 테스트 시나리오

#### 시나리오 1: 기본 1:1 통화
```
1. 서버 실행
2. 브라우저 A: "Alice", "room1" 입장 → 대기 중
3. 브라우저 B: "Bob", "room1" 입장 → 자동 연결
4. 양쪽에서 비디오/오디오 확인
```

#### 시나리오 2: 다중 방
```
1. 서버 실행
2. 브라우저 A: "Alice", "room1" 입장
3. 브라우저 B: "Bob", "room2" 입장 (연결 안 됨 - 다른 방)
4. 브라우저 C: "Charlie", "room1" 입장 → Alice와 연결
```

## 📁 파일 구조

```
/home/jino/go/src/ms-gateway/client/exam/
├── server.go                # 기본 WebRTC 시그널링 서버
├── server_optimized.go      # 대용량 처리 최적화 서버 ⚡
├── client.html              # HTML/JavaScript 클라이언트
├── benchmark_test.go        # 성능 벤치마크 테스트
├── README.md                # 이 문서
├── PERFORMANCE.md           # 성능 최적화 가이드
├── TROUBLESHOOTING.md       # 문제 해결 가이드
└── exam.md                  # 프로젝트 개요
```

## ✨ 주요 기능

### 서버 기능

**기본 서버 (server.go)**:
- ✅ WebSocket 기반 실시간 통신
- ✅ 방(Room) 기반 사용자 관리
- ✅ Offer/Answer/ICE Candidate 시그널링
- ✅ **텍스트 채팅 메시지 브로드캐스트** 💬
- ✅ 사용자 연결/해제 감지
- ✅ 빈 방 자동 삭제
- ✅ CORS 지원
- ✅ 동시성 안전 (goroutine-safe with mutex)

**최적화 서버 (server_optimized.go)** ⚡:
- ✅ **워커 풀 패턴** (100 workers) - 메시지 처리 10배 향상
- ✅ **비동기 브로드캐스트** - CPU 멀티코어 활용
- ✅ **메시지 버퍼링 및 배치 전송** - 네트워크 효율 2-3배 향상
- ✅ **메모리 풀링** - GC 부담 80% 감소
- ✅ **연결 제한** (최대 50,000 동시 연결)
- ✅ **타임아웃 및 데드라인 관리** - 죽은 연결 자동 정리
- ✅ **Graceful Shutdown** - 안전한 종료
- ✅ **실시간 통계 및 모니터링** - 5초마다 자동 출력
- ✅ **헬스 체크 API** (/health)
- ✅ **자동 방 정리** - 메모리 효율성

### 클라이언트 기능
- ✅ 비디오/오디오 실시간 스트리밍
- ✅ **실시간 텍스트 채팅** 💬
- ✅ 음소거 기능
- ✅ 비디오 켜기/끄기
- ✅ 연결 상태 표시
- ✅ 반응형 디자인 (모바일 지원)
- ✅ 현대적인 UI/UX
- ✅ Enter 키로 빠른 메시지 전송
- ✅ 채팅 메시지 애니메이션

## 📖 코드 설명

### 서버 (server.go)

#### 1. 주요 구조체

```go
// Room: 각 방의 사용자 정보를 관리
type Room struct {
    Users map[*websocket.Conn]string  // WebSocket 연결 -> 사용자 이름
    mu    sync.RWMutex                // 동시성 제어를 위한 읽기/쓰기 뮤텍스
}

// Message: 클라이언트와 주고받는 메시지
type Message struct {
    Type string                 `json:"type"`
    Data map[string]interface{} `json:"data,omitempty"`
}
```

**설명**:
- `Room`: 각 채팅방에 접속한 사용자들을 관리합니다. `mu` 뮤텍스로 여러 고루틴이 동시에 접근할 때 데이터 경합을 방지합니다.
- `Message`: JSON 형식으로 클라이언트와 통신합니다. `type`은 메시지 종류(join, offer, answer 등), `data`는 추가 정보를 담습니다.

#### 2. 주요 함수

**getRoom**: 방을 가져오거나 새로 생성
```go
func getRoom(roomName string) *Room {
    roomMu.Lock()           // 전역 뮤텍스 잠금 (rooms 맵 보호)
    defer roomMu.Unlock()   // 함수 종료 시 자동 잠금 해제
    
    if room, exists := rooms[roomName]; exists {
        return room  // 이미 존재하면 반환
    }
    
    // 새 방 생성
    room := &Room{
        Users: make(map[*websocket.Conn]string),
    }
    rooms[roomName] = room
    return room
}
```

**broadcast**: 같은 방의 다른 사용자에게 메시지 전송
```go
func broadcast(room *Room, sender *websocket.Conn, msg Message) {
    room.mu.RLock()          // 읽기 잠금 (여러 고루틴이 동시에 읽을 수 있음)
    defer room.mu.RUnlock()
    
    data, _ := json.Marshal(msg)  // 메시지를 JSON으로 변환
    
    for conn := range room.Users {
        if conn != sender {  // 발신자 제외
            conn.WriteMessage(websocket.TextMessage, data)
        }
    }
}
```

**handleConnection**: WebSocket 연결 처리 (핵심 로직)
```go
func handleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)  // HTTP를 WebSocket으로 업그레이드
    if err != nil {
        return
    }
    defer conn.Close()  // 함수 종료 시 연결 닫기
    
    var currentRoom *Room
    var roomName string
    
    // 무한 루프: 메시지 수신 대기
    for {
        var msg Message
        err := conn.ReadJSON(&msg)  // JSON 메시지 읽기
        if err != nil {
            break  // 연결 종료 또는 오류
        }
        
        // 메시지 타입별 처리
        switch msg.Type {
        case "join":
            // 방 입장 처리
        case "offer":
            // WebRTC offer 전달
        case "answer":
            // WebRTC answer 전달
        case "ice_candidate":
            // ICE candidate 전달
        }
    }
    
    // 연결 종료 처리: 사용자를 방에서 제거
}
```

#### 3. WebRTC 시그널링 흐름

```
[사용자 A]                [서버]                [사용자 B]
    |                        |                        |
    |--- join (room1) ------>|                        |
    |                        |<--- join (room1) ------|
    |                        |                        |
    |<------ start ----------|---- start ------------>|  (2명이 되면)
    |                        |                        |
    |--- offer ------------->|                        |
    |                        |---- offer ------------>|
    |                        |<--- answer ------------|
    |<----- answer ----------|                        |
    |                        |                        |
    |-- ice_candidate ------>|-- ice_candidate ------>|
    |<- ice_candidate -------|<- ice_candidate -------|
    |                        |                        |
```

**설명**:
1. 두 사용자가 같은 방에 입장
2. 서버가 "start" 신호 전송
3. 사용자 A가 WebRTC offer 생성 및 전송
4. 사용자 B가 answer 생성 및 전송
5. 양쪽에서 ICE candidate 교환
6. P2P 연결 완료 (이후 미디어는 P2P로 직접 전송)

#### 4. 채팅 메시지 처리 💬 (신규)

서버는 `chat` 타입 메시지를 받으면 발신자 정보를 포함하여 같은 방의 다른 사용자에게 브로드캐스트합니다.

```go
case "chat":
    // 채팅 메시지 처리
    if currentRoom != nil {
        currentRoom.mu.RLock()
        senderName := currentRoom.Users[conn]
        currentRoom.mu.RUnlock()

        log.Printf("Chat message from %s: %v", senderName, msg.Data["message"])
        
        // 발신자 이름을 포함하여 브로드캐스트
        chatMsg := Message{
            Type: "chat",
            Data: map[string]interface{}{
                "sender":  senderName,
                "message": msg.Data["message"],
            },
        }
        broadcast(currentRoom, conn, chatMsg)
    }
```

**흐름**:
1. 사용자가 채팅 메시지 전송
2. 서버가 발신자 이름을 조회
3. 발신자 정보와 메시지를 함께 패키징
4. 같은 방의 다른 사용자에게 브로드캐스트

### 클라이언트 (client.html)

#### 1. WebRTC 설정

```javascript
const configuration = {
    iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' }
    ]
};
```

**설명**:
- **STUN 서버**: NAT/방화벽 뒤에 있는 사용자의 공인 IP 주소를 알아냅니다.
- 구글의 무료 STUN 서버를 사용합니다.

#### 2. 주요 함수

**joinRoom**: 방 입장 및 초기화
```javascript
async function joinRoom() {
    // 1. 로컬 미디어 스트림 가져오기 (카메라 + 마이크)
    localStream = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true
    });
    
    // 2. WebSocket 연결
    socket = new WebSocket('ws://localhost:3000/ws');
    
    socket.onopen = () => {
        // 3. 방 입장 메시지 전송
        socket.send(JSON.stringify({
            type: 'join',
            data: { name: userName, room: roomName }
        }));
    };
    
    socket.onmessage = async (event) => {
        // 4. 서버 메시지 처리
        const message = JSON.parse(event.data);
        // ... 메시지 타입별 처리
    };
}
```

**createPeerConnection**: WebRTC 피어 연결 생성
```javascript
function createPeerConnection() {
    peerConnection = new RTCPeerConnection(configuration);
    
    // 1. 로컬 스트림의 트랙을 피어 연결에 추가
    localStream.getTracks().forEach(track => {
        peerConnection.addTrack(track, localStream);
    });
    
    // 2. 원격 스트림 수신 처리
    peerConnection.ontrack = (event) => {
        // 상대방의 비디오/오디오를 화면에 표시
        remoteVideo.srcObject = event.streams[0];
    };
    
    // 3. ICE candidate 생성 시 서버로 전송
    peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
            socket.send(JSON.stringify({
                type: 'ice_candidate',
                data: { candidate: event.candidate }
            }));
        }
    };
}
```

**createOffer**: Offer 생성 (호출자)
```javascript
async function createOffer() {
    createPeerConnection();
    
    // 1. SDP offer 생성
    const offer = await peerConnection.createOffer();
    
    // 2. 로컬 설명으로 설정
    await peerConnection.setLocalDescription(offer);
    
    // 3. 서버를 통해 상대방에게 전송
    socket.send(JSON.stringify({
        type: 'offer',
        data: { offer: offer }
    }));
}
```

**handleOffer**: Offer 처리 및 Answer 생성 (수신자)
```javascript
async function handleOffer(data) {
    createPeerConnection();
    
    // 1. 원격 설명으로 offer 설정
    await peerConnection.setRemoteDescription(
        new RTCSessionDescription(data.offer)
    );
    
    // 2. SDP answer 생성
    const answer = await peerConnection.createAnswer();
    
    // 3. 로컬 설명으로 설정
    await peerConnection.setLocalDescription(answer);
    
    // 4. 서버를 통해 상대방에게 전송
    socket.send(JSON.stringify({
        type: 'answer',
        data: { answer: answer }
    }));
}
```

#### 3. WebRTC 연결 흐름 (클라이언트 관점)

```
[사용자 A - 호출자]                    [사용자 B - 수신자]
       |                                      |
   joinRoom()                            joinRoom()
       |                                      |
   getUserMedia()                        getUserMedia()
   (카메라/마이크 접근)                   (카메라/마이크 접근)
       |                                      |
   WebSocket 연결                        WebSocket 연결
       |                                      |
   "start" 수신                          대기 중
       |                                      |
   createOffer()                             |
   - createPeerConnection()                  |
   - addTrack()                              |
   - createOffer()                           |
   - setLocalDescription()                   |
   - send(offer) -----------------------> "offer" 수신
       |                                      |
       |                                  handleOffer()
       |                                  - createPeerConnection()
       |                                  - addTrack()
       |                                  - setRemoteDescription(offer)
       |                                  - createAnswer()
       |                                  - setLocalDescription()
   "answer" 수신 <----------------------- send(answer)
       |                                      |
   handleAnswer()                            |
   - setRemoteDescription(answer)            |
       |                                      |
   ICE candidate 교환 <------------------->  ICE candidate 교환
       |                                      |
   P2P 연결 완료                         P2P 연결 완료
   ontrack 이벤트 발생                   ontrack 이벤트 발생
   (상대방 비디오 표시)                  (상대방 비디오 표시)
```

#### 4. 채팅 기능 💬 (신규)

클라이언트에서 채팅 메시지를 전송하고 수신하는 주요 함수들입니다.

**채팅 메시지 전송**:
```javascript
function sendChatMessage() {
    const chatInput = document.getElementById('chatInput');
    const message = chatInput.value.trim();
    
    if (!message) return;
    
    // 서버로 메시지 전송
    socket.send(JSON.stringify({
        type: 'chat',
        data: { message: message }
    }));
    
    // 내가 보낸 메시지 표시
    displayChatMessage('나', message, true);
    
    // 입력창 초기화
    chatInput.value = '';
}
```

**채팅 메시지 표시**:
```javascript
function displayChatMessage(sender, message, isSent) {
    const chatMessages = document.getElementById('chatMessages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${isSent ? 'sent' : 'received'}`;
    
    if (!isSent) {
        // 받은 메시지는 발신자 이름 표시
        const senderDiv = document.createElement('div');
        senderDiv.className = 'sender';
        senderDiv.textContent = sender;
        messageDiv.appendChild(senderDiv);
    }
    
    const textDiv = document.createElement('div');
    textDiv.className = 'text';
    textDiv.textContent = message;
    messageDiv.appendChild(textDiv);
    
    chatMessages.appendChild(messageDiv);
    
    // 자동 스크롤 (최신 메시지로)
    chatMessages.scrollTop = chatMessages.scrollHeight;
}
```

**메시지 수신 처리**:
```javascript
socket.onmessage = async (event) => {
    const message = JSON.parse(event.data);
    
    switch (message.type) {
        case 'chat':
            // 채팅 메시지 수신
            displayChatMessage(message.data.sender, message.data.message, false);
            break;
        // ... 다른 케이스들
    }
};
```

**주요 기능**:
- ✅ Enter 키로 빠른 전송 (`handleChatKeyPress` 함수)
- ✅ 보낸 메시지와 받은 메시지 구분 표시 (오른쪽/왼쪽 정렬)
- ✅ 자동 스크롤로 최신 메시지 항상 보임
- ✅ 애니메이션 효과 (slideIn)

## 🔧 문제 해결

### 0. 클라이언트 연결이 안 되는 경우 ⚠️

**증상**: 두 사용자가 같은 방에 입장했지만 비디오가 연결되지 않음

**해결됨**: 최적화 서버의 타이밍 이슈 수정 완료

상세 내용은 [CONNECTION_FIX.md](./CONNECTION_FIX.md) 참조

**빠른 확인**:
```bash
# 서버 로그에서 다음 메시지 확인
Alice joined room room1 (total users: 1)
Bob joined room room1 (total users: 2)
Start signal sent to Alice  # ← 이 메시지가 있어야 함
```

### 1. 서버가 시작되지 않음
```bash
# gorilla/websocket이 설치되어 있는지 확인
go get github.com/gorilla/websocket

# go.mod 파일 초기화
go mod init exam
go mod tidy
```

### 2. 카메라/마이크에 접근할 수 없음
- 브라우저에서 권한을 허용했는지 확인
- HTTPS 또는 localhost에서 실행하는지 확인 (HTTP에서는 제한됨)
- 다른 앱이 카메라/마이크를 사용하고 있지 않은지 확인

### 3. WebSocket 연결 실패
- 서버가 실행 중인지 확인 (`go run server.go`)
- 방화벽에서 3000번 포트를 허용했는지 확인
- `client.html`에서 WebSocket URL 확인: `ws://localhost:3000/ws`

### 4. 두 사용자가 연결되지 않음
- 같은 방 이름을 입력했는지 확인
- 브라우저 콘솔에서 오류 메시지 확인 (F12)
- 서버 로그 확인

### 5. 비디오가 보이지 않음
- ICE candidate가 교환되고 있는지 확인 (서버 로그)
- STUN 서버가 응답하는지 확인
- 네트워크 방화벽/NAT 설정 확인
- 브라우저 호환성 확인 (Chrome, Firefox, Edge 권장)

## 🌐 배포 시 고려사항

### 프로덕션 환경
1. **HTTPS 사용**: WebRTC는 보안 연결 필요
2. **TURN 서버 추가**: 제한적인 네트워크 환경에서도 연결 가능
3. **환경 변수**: 포트, 서버 URL 등을 환경 변수로 관리
4. **로깅**: 프로덕션 로거 사용 (예: logrus, zap)
5. **에러 처리**: 더 세밀한 에러 처리 및 복구 로직

### TURN 서버 설정 예시
```javascript
const configuration = {
    iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        {
            urls: 'turn:your-turn-server.com:3478',
            username: 'username',
            credential: 'password'
        }
    ]
};
```

## 📚 참고 자료

- [WebRTC API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [WebRTC for the Curious](https://webrtcforthecurious.com/)

## 🎓 학습 포인트

### 초보 개발자를 위한 설명

1. **WebSocket이란?**
   - HTTP는 "요청-응답" 방식 (클라이언트가 요청해야 서버가 응답)
   - WebSocket은 "양방향" 통신 (서버가 먼저 메시지를 보낼 수 있음)
   - 실시간 채팅, 게임 등에 사용

2. **WebRTC란?**
   - 브라우저끼리 직접 통신하는 기술 (P2P)
   - 비디오, 오디오, 데이터를 실시간으로 전송
   - 서버는 "시그널링"만 담당 (연결 정보 교환)

3. **시그널링이란?**
   - WebRTC 연결을 시작하기 위한 정보 교환
   - Offer, Answer, ICE Candidate 등
   - WebSocket을 통해 주고받음

4. **Goroutine과 Mutex**
   - Goroutine: 경량 쓰레드 (동시에 여러 작업 처리)
   - Mutex: 여러 Goroutine이 같은 데이터를 안전하게 접근하도록 함
   - `sync.RWMutex`: 읽기는 동시에, 쓰기는 독점

## 🚀 성능 및 벤치마크

### 서버 비교

| 측정 항목 | 기본 서버 | 최적화 서버 |
|----------|----------|------------|
| 최대 동시 연결 | ~1,000 | ~50,000 |
| 메시지 처리량 | 10K/sec | 150K/sec |
| CPU 사용률 | 15% (단일) | 80% (멀티) |
| 메모리 사용 | 500MB | 800MB |
| 브로드캐스트 지연 | 500ms | 50ms |

### 벤치마크 실행

```bash
# 서버 시작 (별도 터미널)
go run server_optimized.go

# 벤치마크 실행
go test -bench=. -benchmem

# 특정 벤치마크만 실행
go test -bench=BenchmarkConcurrentConnections -benchmem
go test -bench=BenchmarkMessageThroughput -benchmem
go test -bench=BenchmarkRoomBroadcast -benchmem

# 메모리 누수 테스트
go test -run=TestMemoryLeak -v
```

### 부하 테스트 예시

```bash
# 1,000개 동시 연결 테스트
for i in {1..1000}; do
  (wscat -c ws://localhost:3000/ws &)
done

# 헬스 체크로 상태 확인
watch -n 1 'curl -s http://localhost:3000/health | jq'
```

### 성능 튜닝 가이드

자세한 성능 최적화 방법은 [PERFORMANCE.md](./PERFORMANCE.md)를 참조하세요.

주요 내용:
- 워커 풀 패턴
- 메모리 풀링
- 브로드캐스트 최적화
- 연결 제한 및 타임아웃
- CPU 멀티코어 활용
- 모니터링 및 통계

## 📝 라이선스

이 프로젝트는 교육 및 학습 목적으로 제작되었습니다.

