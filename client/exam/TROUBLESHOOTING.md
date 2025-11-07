# 문제 해결 가이드

## 🔧 연결 문제 해결

### 문제: 두 브라우저가 같은 방에 입장했지만 비디오가 연결되지 않음

#### 원인
1. **start 시그널 문제**: 두 사용자 모두에게 start 시그널이 전송되어 동시에 offer를 생성하려고 시도
2. **브로드캐스트 로직 문제**: 발신자를 제외한 사용자에게만 메시지 전송

#### 해결 방법

**서버 측 수정 (server.go)**:

1. **broadcastAll 함수 추가**: 모든 사용자에게 메시지를 전송하는 함수
```go
// broadcastAll: 같은 방의 모든 사용자에게 메시지 전송 (발신자 포함)
func broadcastAll(room *Room, msg Message) {
    room.mu.RLock()
    defer room.mu.RUnlock()

    data, err := json.Marshal(msg)
    if err != nil {
        log.Printf("Error marshaling message: %v", err)
        return
    }

    for conn := range room.Users {
        if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
            log.Printf("Error sending message: %v", err)
        }
    }
}
```

2. **start 시그널을 첫 번째 사용자에게만 전송**:
```go
// 방에 2명이 되면 첫 번째 사용자(기존 사용자)에게만 start 시그널 전송
if len(currentRoom.Users) == 2 {
    log.Printf("Room %s now has 2 users, sending start signal to first user", roomName)
    // 현재 연결(두 번째 사용자)이 아닌 다른 연결(첫 번째 사용자)에게 전송
    for otherConn := range currentRoom.Users {
        if otherConn != conn {
            startMsg := Message{Type: "start"}
            data, _ := json.Marshal(startMsg)
            if err := otherConn.WriteMessage(websocket.TextMessage, data); err != nil {
                log.Printf("Error sending start message: %v", err)
            } else {
                log.Printf("Start signal sent to %s", currentRoom.Users[otherConn])
            }
            break
        }
    }
}
```

3. **상세한 로깅 추가**:
- 각 메시지 타입별로 발신자와 방 상태 로깅
- 디버깅을 위한 상세 정보 출력

**클라이언트 측 수정 (client.html)**:

1. **상세한 콘솔 로그 추가**:
```javascript
console.log('Received message:', message.type, message);
```

2. **에러 핸들링 강화**:
```javascript
async function createOffer() {
    try {
        console.log('Creating offer...');
        // ... offer 생성 로직
    } catch (error) {
        console.error('Error creating offer:', error);
    }
}
```

3. **PeerConnection 상태 체크**:
```javascript
if (!peerConnection) {
    console.log('PeerConnection not ready, ignoring ICE candidate');
    return;
}
```

## 🎯 연결 흐름

### 정상 연결 흐름

```
[사용자 A - 첫 번째]           [서버]            [사용자 B - 두 번째]
      |                          |                        |
  join(room1) ------------------>|                        |
      |                          |                        |
      |                          |<-----------------  join(room1)
      |                          |                        |
  <--- start -------------------|   (첫 번째 사용자에게만)
      |                          |                        |
  createOffer()                  |                        |
  - createPeerConnection()       |                        |
  - createOffer()               |                        |
  - setLocalDescription()       |                        |
      |                          |                        |
  --- offer -------------------->|                        |
      |                          |---- offer ------------>|
      |                          |                        |
      |                          |                   handleOffer()
      |                          |              - createPeerConnection()
      |                          |              - setRemoteDescription()
      |                          |              - createAnswer()
      |                          |              - setLocalDescription()
      |                          |                        |
      |                          |<---- answer -----------|
  <--- answer -------------------|                        |
      |                          |                        |
  handleAnswer()                 |                        |
  - setRemoteDescription()       |                        |
      |                          |                        |
  <--- ICE candidates ---------->|<--- ICE candidates --->|
      |                          |                        |
    연결 완료!                    |                    연결 완료!
```

### 주요 포인트

1. **첫 번째 사용자만 start 시그널 수신**: offer를 생성할 책임
2. **두 번째 사용자는 대기**: offer를 받으면 answer 생성
3. **ICE candidate 양방향 교환**: 네트워크 정보 교환
4. **P2P 연결 완료**: 비디오/오디오 스트림 전송 시작

## 🐛 디버깅 방법

### 서버 로그 확인

```bash
go run server.go
```

정상 연결 시 다음과 같은 로그가 출력됩니다:
```
New user connected
Alice joined room room1 (total users: 1)
New user connected
Bob joined room room1 (total users: 2)
Room room1 now has 2 users, sending start signal to first user
Start signal sent to Alice
Received offer from Alice (room has 2 users)
Received answer from Bob
Received ICE candidate from Alice
Received ICE candidate from Bob
...
```

### 브라우저 콘솔 확인

**F12** 또는 **Cmd+Option+I** (Mac)를 눌러 개발자 도구를 엽니다.

**첫 번째 사용자 (Alice)** 콘솔:
```
WebSocket connected
Start signal received, creating offer...
Creating offer...
PeerConnection created for offer
Offer created: {...}
Local description set (offer)
Offer sent to server
Sending ICE candidate
Answer received, setting remote description...
Handling answer...
Remote description set (answer)
ICE candidate received
Adding ICE candidate
ICE candidate added successfully
Connection state: connected
Received remote track
```

**두 번째 사용자 (Bob)** 콘솔:
```
WebSocket connected
Offer received, creating answer...
Handling offer...
PeerConnection created for answer
Remote description set (offer)
Answer created: {...}
Local description set (answer)
Answer sent to server
Sending ICE candidate
ICE candidate received
Adding ICE candidate
ICE candidate added successfully
Connection state: connected
Received remote track
```

## ❓ 자주 묻는 질문

### Q1: 여전히 연결이 안 됩니다
A: 다음을 확인하세요:
1. 서버가 실행 중인지 확인
2. 브라우저 콘솔에서 오류 메시지 확인
3. 두 브라우저가 **정확히 같은 방 이름**을 입력했는지 확인
4. 방화벽이나 보안 소프트웨어가 WebSocket 연결을 차단하는지 확인
5. 카메라/마이크 권한을 허용했는지 확인

### Q2: ICE candidate 오류가 발생합니다
A: 
- STUN 서버 연결을 확인하세요
- 네트워크 방화벽 설정을 확인하세요
- 프로덕션 환경에서는 TURN 서버 추가를 고려하세요

### Q3: 한쪽 비디오만 보입니다
A:
- 양쪽 브라우저 콘솔을 모두 확인하세요
- ICE candidate가 양방향으로 교환되는지 확인하세요
- PeerConnection 상태를 확인하세요 (`connectionState`)

### Q4: 연결 후 바로 끊어집니다
A:
- NAT/방화벽 설정 확인
- TURN 서버 사용 고려
- 브라우저 호환성 확인 (Chrome, Firefox, Edge 권장)

## 🔍 추가 테스트

### 로컬 네트워크 테스트
```bash
# 서버 실행
go run server.go

# 같은 컴퓨터에서 브라우저 2개로 테스트
# Chrome과 Firefox를 각각 사용하거나
# Chrome 일반 모드와 시크릿 모드 사용
```

### 네트워크 간 테스트
```bash
# 서버가 실행 중인 컴퓨터의 IP 확인
ip addr show  # Linux
ipconfig      # Windows

# client.html에서 WebSocket URL 수정
socket = new WebSocket('ws://192.168.x.x:3000/ws');
```

## 📞 지원

문제가 계속되면:
1. 서버 로그 전체를 확인
2. 브라우저 콘솔 로그를 확인
3. 네트워크 환경 확인 (NAT, 방화벽)
4. 브라우저 버전 확인

---

**참고**: 이 문서는 WebRTC 시그널링 서버의 연결 문제 해결을 위한 가이드입니다.

