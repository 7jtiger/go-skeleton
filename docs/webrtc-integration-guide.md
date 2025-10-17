# WebRTC P2P 화상채팅 통합 가이드

## 개요

이 문서는 모바일 앱에서 WebRTC P2P 화상채팅을 구현하기 위한 서버 측 API 및 클라이언트 통합 가이드입니다.

## 서버 API 엔드포인트

### 1. 로그인 시 WebRTC 설정 정보 획득

**POST** `/acc/v01/login`

로그인 성공 시 응답에 WebRTC 설정 정보가 포함됩니다.

**응답 예시:**
```json
{
  "result": 0,
  "msg": "success",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "uid": "user123",
  "webrtc": {
    "iceServers": [
      {
        "urls": ["stun:stun.l.google.com:19302"],
        "type": "stun"
      },
      {
        "urls": ["stun:stun1.l.google.com:19302"],
        "type": "stun"
      }
    ],
    "signalingServer": "ws://localhost:8080/ws",
    "stunServers": [
      "stun:stun.l.google.com:19302",
      "stun:stun1.l.google.com:19302"
    ],
    "mediaSettings": {
      "video": {
        "enabled": true,
        "width": 1280,
        "height": 720,
        "frameRate": 30,
        "maxBitrate": 2000000
      },
      "audio": {
        "enabled": true,
        "echoCancellation": true,
        "noiseSuppression": true,
        "autoGainControl": true,
        "maxBitrate": 128000
      }
    }
  }
}
```

### 2. WebRTC 설정 정보 조회

**GET** `/webrtc/v01/config`

**Headers:**
- `Authorization: Bearer <JWT_TOKEN>`

**응답:**
```json
{
  "iceServers": [...],
  "signalingServer": "ws://localhost:8080/ws",
  "mediaSettings": {...}
}
```

### 3. 통화 가능한 사용자 목록 조회

**GET** `/webrtc/v01/available-users`

**Headers:**
- `Authorization: Bearer <JWT_TOKEN>`

**응답:**
```json
{
  "result": 0,
  "data": ["user456", "user789", "user101"]
}
```

### 4. 통화 상태 업데이트

**POST** `/webrtc/v01/call-status`

**Headers:**
- `Authorization: Bearer <JWT_TOKEN>`

**요청 Body:**
```json
{
  "isInCall": true,
  "callWith": "user456"
}
```

### 5. WebRTC 세션 하트비트

**POST** `/webrtc/v01/heartbeat`

**Headers:**
- `Authorization: Bearer <JWT_TOKEN>`

**요청 Body:**
```json
{
  "deviceInfo": "iPhone 15 Pro",
  "networkInfo": "WiFi"
}
```

### 6. WebRTC 통계 조회

**GET** `/webrtc/v01/stats`

**응답:**
```json
{
  "totalSessions": 150,
  "availableUsers": 45,
  "activeCalls": 12,
  "lastUpdated": "2024-01-20T10:30:00Z"
}
```

## 클라이언트 통합 가이드

### 1. 로그인 후 WebRTC 초기화

```javascript
// 1. 로그인 수행
const loginResponse = await fetch('/acc/v01/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    id: 'user123',
    pw: 'hashedPassword'
  })
});

const loginData = await loginResponse.json();

// 2. WebRTC 설정 저장
const webrtcConfig = loginData.webrtc;
const token = loginData.token;

// 3. WebRTC PeerConnection 초기화
const peerConnection = new RTCPeerConnection({
  iceServers: webrtcConfig.iceServers
});

// 4. 시그널링 서버 연결
const signalingSocket = new WebSocket(webrtcConfig.signalingServer);
```

### 2. 미디어 스트림 설정

```javascript
// 로그인 시 받은 설정을 사용하여 미디어 스트림 생성
const mediaConstraints = {
  video: {
    width: webrtcConfig.mediaSettings.video.width,
    height: webrtcConfig.mediaSettings.video.height,
    frameRate: webrtcConfig.mediaSettings.video.frameRate
  },
  audio: {
    echoCancellation: webrtcConfig.mediaSettings.audio.echoCancellation,
    noiseSuppression: webrtcConfig.mediaSettings.audio.noiseSuppression,
    autoGainControl: webrtcConfig.mediaSettings.audio.autoGainControl
  }
};

const localStream = await navigator.mediaDevices.getUserMedia(mediaConstraints);
```

### 3. 통화 가능한 사용자 목록 조회

```javascript
const getAvailableUsers = async () => {
  const response = await fetch('/webrtc/v01/available-users', {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  
  const data = await response.json();
  return data.data; // 사용자 ID 배열
};
```

### 4. 통화 상태 관리

```javascript
// 통화 시작 시
const startCall = async (targetUserId) => {
  await fetch('/webrtc/v01/call-status', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      isInCall: true,
      callWith: targetUserId
    })
  });
};

// 통화 종료 시
const endCall = async () => {
  await fetch('/webrtc/v01/call-status', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      isInCall: false,
      callWith: ""
    })
  });
};
```

### 5. 세션 하트비트 유지

```javascript
// 30초마다 하트비트 전송
setInterval(async () => {
  await fetch('/webrtc/v01/heartbeat', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      deviceInfo: navigator.userAgent,
      networkInfo: navigator.connection?.effectiveType || "unknown"
    })
  });
}, 30000);
```

## Redis 데이터 구조

### WebRTC 세션 정보 (HSET)
```
Key: webrtc:sessions
Field: userID
Value: {
  "userId": "user123",
  "sessionId": "session-uuid",
  "isInCall": false,
  "connectionState": "connected",
  "lastHeartbeat": "2024-01-20T10:30:00Z",
  "deviceInfo": "iPhone 15 Pro",
  "networkInfo": "WiFi"
}
```

### 통화 가능한 사용자 (SET)
```
Key: webrtc:available_users
Members: ["user123", "user456", "user789"]
```

### JWT 토큰 세션 (HSET)
```
Key: jwt:sessions
Field: token
Value: {
  "at_login": "2024-01-20T10:00:00Z",
  "at_expire": "2024-01-21T10:00:00Z",
  "sid": "127.0.0.1",
  "uid": "Mozilla/5.0..."
}
```

## 보안 고려사항

1. **JWT 토큰 관리**: 모든 WebRTC API는 JWT 인증이 필요합니다.
2. **TURN 서버 크리덴셜**: 프로덕션 환경에서는 임시 크리덴셜 사용을 권장합니다.
3. **세션 타임아웃**: 4시간 후 자동으로 세션이 만료됩니다.
4. **하트비트**: 30분 이상 비활성 세션은 자동으로 정리됩니다.

## 네트워크 요구사항

- **STUN 서버**: NAT 트래버설을 위해 Google STUN 서버 사용
- **TURN 서버**: 대칭 NAT 환경에서 필요 (선택사항)
- **시그널링 서버**: WebSocket 연결을 통한 시그널링

## 모바일 최적화

1. **배터리 최적화**: 백그라운드에서 하트비트 간격 조정
2. **네트워크 적응**: 네트워크 상태에 따른 비트레이트 조정
3. **메모리 관리**: 통화 종료 시 리소스 정리

## 문제 해결

### 연결 실패 시
1. 네트워크 연결 상태 확인
2. JWT 토큰 유효성 검증
3. 방화벽 및 NAT 설정 확인

### 오디오/비디오 문제 시
1. 미디어 권한 확인
2. 디바이스 가용성 확인
3. 브라우저 호환성 확인

이 가이드를 참고하여 모바일 앱에 WebRTC P2P 화상채팅 기능을 통합하시기 바랍니다. 