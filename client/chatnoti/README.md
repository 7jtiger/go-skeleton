# 채팅 대기방 테스트 클라이언트

## 개요
브라우저에서 채팅 대기방 기능을 테스트할 수 있는 클라이언트입니다.

## 사용 방법

### 1. 서버 실행
```bash
cd /home/jino/go/src/ms-gateway
go run main.go
```

### 2. 브라우저에서 테스트

1. **첫 번째 브라우저 창**
   - `index.html` 파일을 브라우저에서 열기
   - 사용자 ID: `user1` 입력
   - 서버 주소: `ws://localhost:8080` (기본값)
   - "대기방 접속" 버튼 클릭

2. **두 번째 브라우저 창**
   - `index.html` 파일을 새 브라우저 창에서 열기
   - 사용자 ID: `user2` 입력
   - 서버 주소: `ws://localhost:8080` (기본값)
   - "대기방 접속" 버튼 클릭

3. **채팅 요청**
   - 첫 번째 브라우저에서 "상대방 ID"에 `user2` 입력
   - "채팅 요청" 버튼 클릭

4. **채팅 수락**
   - 두 번째 브라우저에서 알림 팝업이 나타남
   - "수락" 버튼 클릭

5. **텍스트 채팅**
   - 양쪽 브라우저에서 텍스트 메시지 주고받기 가능

## 기능

- ✅ WebSocket 상시 연결 (대기방 접속 시 자동 연결)
- ✅ 채팅 요청 전송
- ✅ 채팅 요청 수락/거절
- ✅ 텍스트 메시지 송수신
- ✅ 실시간 메시지 표시
- ✅ 시스템 메시지 표시

## API 엔드포인트

### WebSocket 연결
```
ws://localhost:8080/chat/v01/ws?userId={userId}
```

### 채팅방 생성
```
POST http://localhost:8080/chat/v01/room
Content-Type: application/json

{
  "roomName": "Chat with user2",
  "userId": "user1",
  "isPrivate": true
}
```

## 메시지 형식

### 채팅 요청
```json
{
  "type": "call-request",
  "from": "user1",
  "to": "user2",
  "roomId": "room_abc123",
  "content": "{\"callType\":\"text\"}",
  "timestamp": 1234567890
}
```

### 채팅 수락
```json
{
  "type": "call-accept",
  "from": "user2",
  "to": "user1",
  "roomId": "room_abc123",
  "content": "accepted",
  "timestamp": 1234567890
}
```

### 텍스트 메시지
```json
{
  "type": "text-message",
  "from": "user1",
  "to": "user2",
  "roomId": "room_abc123",
  "content": "안녕하세요",
  "timestamp": 1234567890
}
```

## 주의사항

- 서버가 실행 중이어야 합니다
- CORS 설정이 필요할 수 있습니다 (개발 환경에서는 모든 origin 허용)
- 브라우저 콘솔에서 WebSocket 연결 상태 확인 가능

