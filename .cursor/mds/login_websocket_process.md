# 로그인 시 WebSocket 연결 및 메모리 저장 프로세스

## 개요

이 문서는 사용자 로그인 시 WebSocket 연결을 자동으로 수립하고, 연결 정보를 메모리에 저장하는 전체 프로세스에 대해 설명합니다.

---

## 전체 프로세스 흐름

### 1단계: 사용자 로그인 요청

**엔드포인트**: `POST /acc/v01/login`

**요청 예시**:
```json
{
  "id": "user123",
  "pw": "hashed_password"
}
```

**처리 위치**: `controller/account.go` - `LoginUser()` 함수

**주요 처리 내용**:
1. 사용자 인증 정보 검증 (ID, PW)
2. 데이터베이스에서 사용자 정보 조회
3. 비밀번호 검증
4. JWT 토큰 생성 (AccessToken, RefreshToken)
5. Redis에 JWT 토큰 저장
6. WebRTC 세션 정보 초기화 (Redis 저장)

**응답 예시**:
```json
{
  "message": "success",
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "uid": "user-uuid-123",
  "webrtcConfig": {
    "stunServers": [...],
    "turnServers": [...]
  }
}
```

---

### 2단계: 클라이언트 WebSocket 연결 요청

**엔드포인트**: `GET /webrtc/v01/ws?userId={uid}`

**연결 URL 예시**:
```
ws://server:port/webrtc/v01/ws?userId=user-uuid-123
```

**처리 위치**: `controller/signaling.go` - `HandleWebSocket()` 함수

**주요 처리 내용**:
1. `userId` 쿼리 파라미터 검증
2. HTTP 연결을 WebSocket 프로토콜로 업그레이드
3. `Client` 구조체 생성
4. 메모리에 클라이언트 등록

---

### 3단계: WebSocket 연결 수립 및 메모리 저장

#### 3.1 WebSocket 업그레이드

**코드 위치**: `controller/signaling.go:286-300`

```go
// WebSocket 업그레이드
conn, err := p.upgrader.Upgrade(c.Writer, c.Request, nil)
```

**Upgrader 설정**:
- `ReadBufferSize`: 4096 bytes
- `WriteBufferSize`: 4096 bytes
- `CheckOrigin`: 개발 중 모든 Origin 허용

#### 3.2 Client 객체 생성

**코드 위치**: `controller/signaling.go:302-306`

```go
client := &Client{
    conn: conn,
    send: make(chan []byte, MSG_BUFFER_SIZE),  // 256 버퍼
}
```

**Client 구조체**:
- `conn *websocket.Conn`: WebSocket 연결 객체
- `send chan []byte`: 송신 메시지 버퍼 채널
- `room *Room`: 속한 방 정보
- `userName string`: 사용자 이름
- `lastSeen time.Time`: 마지막 활동 시간

#### 3.3 메모리에 클라이언트 등록

**코드 위치**: `controller/signaling.go:318-335` - `registerClient()` 함수

**저장 위치**: `SignalingController.rooms` 맵

**저장 구조**:
```go
type SignalingController struct {
    rooms   map[string]*Room  // roomId -> Room
    roomsMu sync.RWMutex      // 동기화 뮤텍스
    // ...
}
```

**등록 프로세스**:
1. `roomsMu.Lock()` - 쓰기 락 획득
2. 기존 연결이 있으면 종료 처리
3. `rooms[roomId] = room` - 메모리 맵에 저장
4. `roomsMu.Unlock()` - 락 해제
5. 사용자 목록 브로드캐스트

**메모리 저장 구조**:
```
SignalingController
└── rooms (map[string]*Room)
    └── Room
        ├── ID (int64)
        ├── UserIDs (map[*Client]bool)  ← 실제 클라이언트 저장 위치
        ├── UserNames (map[*Client]string)
        ├── Broadcast (chan *BroadcastMessage)
        └── Unregister (chan *Client)
```

---

### 4단계: 읽기/쓰기 고루틴 시작

**코드 위치**: `controller/signaling.go:313-315`

```go
go p.writePump(client)  // 쓰기 고루틴
go p.readPump(client)   // 읽기 고루틴
```

#### 4.1 readPump (읽기 고루틴)

**기능**:
- WebSocket으로부터 메시지 수신
- Pong 핸들러 설정 (60초 타임아웃)
- JSON 메시지 파싱
- 메시지 타입별 처리

**타임아웃 설정**:
- 읽기 타임아웃: 60초
- Pong 응답 시 타임아웃 갱신

#### 4.2 writePump (쓰기 고루틴)

**기능**:
- `client.send` 채널의 메시지를 WebSocket으로 전송
- 54초마다 Ping 메시지 자동 전송
- 대기 중인 메시지 일괄 전송

**타임아웃 설정**:
- 쓰기 타임아웃: 10초
- Ping 주기: 54초

---

## 로그인 후 권장 프로세스

### 프로세스 1: 순차적 연결 (권장)

```
1. 로그인 API 호출
   ↓
2. 로그인 성공 응답 수신 (JWT 토큰 포함)
   ↓
3. JWT 토큰 저장 (로컬 스토리지/메모리)
   ↓
4. WebSocket 연결 요청
   - URL: ws://server:port/webrtc/v01/ws?userId={uid}
   - Header: Authorization: Bearer {accessToken} (선택사항)
   ↓
5. WebSocket 연결 성공
   ↓
6. 연결 확인 메시지 수신 대기
   ↓
7. 서비스 사용 시작
```

**장점**:
- 명확한 순서 보장
- 각 단계별 에러 처리 용이
- 재연결 로직 구현 간단

**구현 예시 (JavaScript)**:
```javascript
async function loginAndConnect() {
    try {
        // 1. 로그인
        const loginResponse = await fetch('/acc/v01/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: 'user123', pw: 'password' })
        });
        
        const loginData = await loginResponse.json();
        
        // 2. 토큰 저장
        localStorage.setItem('accessToken', loginData.accessToken);
        localStorage.setItem('uid', loginData.uid);
        
        // 3. WebSocket 연결
        const ws = new WebSocket(
            `ws://server:port/webrtc/v01/ws?userId=${loginData.uid}`
        );
        
        ws.onopen = () => {
            console.log('WebSocket 연결 성공');
            // 서비스 사용 시작
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket 연결 실패:', error);
        };
        
    } catch (error) {
        console.error('로그인 실패:', error);
    }
}
```

---

### 프로세스 2: 병렬 연결 (고급)

```
1. 로그인 API 호출
   ↓
2. 로그인 성공 응답 수신
   ↓
3. 동시에:
   - JWT 토큰 저장
   - WebSocket 연결 요청
   ↓
4. WebSocket 연결 성공 대기
   ↓
5. 서비스 사용 시작
```

**장점**:
- 연결 시간 단축
- 빠른 사용자 경험

**주의사항**:
- WebSocket 연결 실패 시 재시도 로직 필요
- 토큰 만료 처리 필요

---

#### 프로세스 2 구현 시 수정 필요한 코드 부분

각 단계별로 수정해야 할 코드 위치와 내용:

##### 1단계: 로그인 API 호출

**현재 코드 위치**: `controller/account.go:200-283` - `LoginUser()` 함수

**수정 필요 사항**:
- 현재는 로그인 성공 후 응답만 반환
- **추가 필요**: WebSocket 연결을 위한 추가 정보 제공 (예: WebSocket URL, 연결 타임아웃 등)

**수정 위치**:
```go
// controller/account.go:273-282
responseData := ptl.LoginUserResp{
    Message:      "success",
    AccessToken:  acTok,
    RefreshToken: refTok,
    UID:          user.Uid,
    WebRTCConfig: *webrtcConfig,
    // 추가 필요: WebSocket 연결 정보
    // WebSocketURL: "ws://server:port/webrtc/v01/ws",
    // ConnectionTimeout: 30,
}
```

**WebRTCConfig 필수 필드** (`protocol/userptl.go:86-91`):
```go
type WebRTCConfig struct {
    ICEServers      []ICEServer `json:"iceServers"`      // 필수: ICE 서버 목록
    SignalingServer string      `json:"signalingServer"` // 필수: 시그널링 서버 URL
    StunServers     []string    `json:"stunServers"`    // 필수: STUN 서버 목록
    MediaSettings   MediaConfig `json:"mediaSettings"`   // 필수: 미디어 설정
}
```

**WebRTCConfig 필수 필드 상세**:

1. **ICEServers** (`[]ICEServer`):
   - **타입**: `[]ICEServer` 배열
   - **필수 여부**: 필수 (최소 1개 이상)
   - **구조**: 
     ```go
     type ICEServer struct {
         URLs []string `json:"urls"` // 필수: STUN 서버 URL 배열
         Type string   `json:"type"` // 필수: "stun" (현재는 STUN만 지원)
     }
     ```
   - **예시 값**: 
     ```json
     [
       {"urls": ["stun:stun.l.google.com:19302"], "type": "stun"},
       {"urls": ["stun:stun1.l.google.com:19302"], "type": "stun"}
     ]
     ```

2. **SignalingServer** (`string`):
   - **타입**: `string`
   - **필수 여부**: 필수
   - **형식**: WebSocket URL (예: `"ws://localhost:8080/webrtc/v01/ws"`)
   - **용도**: WebRTC 시그널링 서버 주소

3. **StunServers** (`[]string`):
   - **타입**: `[]string` 배열
   - **필수 여부**: 필수 (최소 1개 이상)
   - **형식**: STUN 서버 URL 배열
   - **예시 값**: 
     ```json
     [
       "stun:stun.l.google.com:19302",
       "stun:stun1.l.google.com:19302"
     ]
     ```

4. **MediaSettings** (`MediaConfig`):
   - **타입**: `MediaConfig` 구조체
   - **필수 여부**: 필수
   - **구조**:
     ```go
     type MediaConfig struct {
         Video VideoConfig `json:"video"` // 필수: 비디오 설정
         Audio AudioConfig `json:"audio"` // 필수: 오디오 설정
     }
     
     type VideoConfig struct {
         Enabled    bool `json:"enabled"`    // 필수: 비디오 활성화 여부
         Width      int  `json:"width"`     // 필수: 비디오 너비 (픽셀)
         Height     int  `json:"height"`    // 필수: 비디오 높이 (픽셀)
         FrameRate  int  `json:"frameRate"` // 필수: 프레임 레이트
         MaxBitrate int  `json:"maxBitrate"` // 필수: 최대 비트레이트 (bps)
     }
     
     type AudioConfig struct {
         Enabled          bool `json:"enabled"`          // 필수: 오디오 활성화 여부
         EchoCancellation bool `json:"echoCancellation"` // 필수: 에코 제거 여부
         NoiseSuppression bool `json:"noiseSuppression"` // 필수: 노이즈 제거 여부
         AutoGainControl  bool `json:"autoGainControl"` // 필수: 자동 게인 제어 여부
         MaxBitrate       int  `json:"maxBitrate"`      // 필수: 최대 비트레이트 (bps)
     }
     ```
   - **기본값** (현재 구현 기준, `models/redis_db.go:1232-1249`):
     ```go
     MediaSettings: MediaConfig{
         Video: VideoConfig{
             Enabled:    true,
             Width:      1920,
             Height:    1080,
             FrameRate:  30,
             MaxBitrate: 8000000, // 8Mbps
         },
         Audio: AudioConfig{
             Enabled:          true,
             EchoCancellation: true,
             NoiseSuppression: true,
             AutoGainControl:  true,
             MaxBitrate:       128000, // 128kbps
         },
     }
     ```
   
   - **숏폼 최적화 스펙** (모바일/데이터 사용량 최적화):
     ```go
     MediaSettings: MediaConfig{
         Video: VideoConfig{
             Enabled:    true,
             Width:      1280,   // 720p 해상도 (Full HD 대비 절반)
             Height:    720,     // 720p 해상도
             FrameRate:  30,     // 30fps 유지 (부드러운 영상)
             MaxBitrate: 2000000, // 2Mbps (8Mbps 대비 75% 감소)
         },
         Audio: AudioConfig{
             Enabled:          true,
             EchoCancellation: true,
             NoiseSuppression: true,
             AutoGainControl:  true,
             MaxBitrate:       96000,  // 96kbps (128kbps 대비 감소, 충분한 품질)
         },
     }
     ```
   
   - **초저사양 숏폼 스펙** (저사양 기기/느린 네트워크용):
     ```go
     MediaSettings: MediaConfig{
         Video: VideoConfig{
             Enabled:    true,
             Width:      854,    // 480p 해상도
             Height:    480,     // 480p 해상도
             FrameRate:  24,     // 24fps (영화 표준, 데이터 절약)
             MaxBitrate: 1500000, // 1.5Mbps
         },
         Audio: AudioConfig{
             Enabled:          true,
             EchoCancellation: true,
             NoiseSuppression: true,
             AutoGainControl:  true,
             MaxBitrate:       64000,  // 64kbps (전화 품질 수준)
         },
     }
     ```
   
   **스펙 비교표**:
   
   | 항목 | Full HD (현재) | 720p (숏폼 권장) | 480p (초저사양) |
   |------|---------------|-----------------|----------------|
   | 해상도 | 1920x1080 | 1280x720 | 854x480 |
   | 비트레이트 | 8Mbps | 2Mbps | 1.5Mbps |
   | 프레임레이트 | 30fps | 30fps | 24fps |
   | 오디오 비트레이트 | 128kbps | 96kbps | 64kbps |
   | 데이터 사용량 (1분) | ~60MB | ~15MB | ~11MB |
   | 모바일 적합성 | 낮음 | 높음 | 매우 높음 |
   | 권장 용도 | 데스크톱/고품질 | 모바일/일반 | 저사양/느린 네트워크 |

**주의사항**:
- `webrtcConfig`가 `nil`일 수 있음 (`controller/account.go:250-255`)
- 로그인 응답 시 `WebRTCConfig`가 `nil`이면 에러가 발생할 수 있으므로, 기본값 설정 필요
- 현재 구현에서는 `GetWebRTCConfig()` 실패 시 `nil`을 반환하지만, 로그인 응답에는 항상 유효한 값이 필요

---

##### 2단계: 로그인 성공 응답 수신

**현재 코드 위치**: `controller/account.go:282` - `p.ctl.RespSuccess(c, responseData)`

**수정 필요 사항**:
- 클라이언트 측에서 응답을 받은 후 즉시 WebSocket 연결 시도
- **클라이언트 코드 수정 필요** (프론트엔드/클라이언트 애플리케이션)

**수정 위치 (클라이언트 측)**:
```javascript
// 클라이언트 코드 예시
const loginResponse = await fetch('/acc/v01/login', {...});
const loginData = await loginResponse.json();

// 즉시 WebSocket 연결 시도 (병렬 처리)
const wsPromise = connectWebSocket(loginData.uid, loginData.accessToken);
const tokenSavePromise = saveToken(loginData.accessToken);

// 동시 실행
await Promise.all([tokenSavePromise, wsPromise]);
```

---

##### 3단계: 동시 처리 (JWT 토큰 저장 + WebSocket 연결 요청)

###### 3-1. JWT 토큰 저장

**현재 코드 위치**: 
- 서버 측: `controller/account.go:226-230` - `genLoginUserToken()` 함수
- Redis 저장: `controller/account.go:317-327` - `HSetJWTAccess()`, `HSetJWTRefresh()`

**수정 필요 사항**:
- 현재는 서버 측에서만 토큰 저장
- **클라이언트 측에서도 토큰 저장 로직 필요** (로컬 스토리지/메모리)
- **수정 위치**: 클라이언트 코드 (프론트엔드)

**클라이언트 코드 수정 예시**:
```javascript
// 클라이언트 측 토큰 저장 함수
async function saveToken(accessToken, refreshToken) {
    try {
        localStorage.setItem('accessToken', accessToken);
        localStorage.setItem('refreshToken', refreshToken);
        // 또는 메모리 저장
        tokenManager.setToken(accessToken);
        
        // 반환값: Promise<void> - 성공 시 아무 값도 반환하지 않음
        return Promise.resolve();
    } catch (error) {
        // 저장 실패 시 에러를 throw하여 Promise.reject() 처리
        return Promise.reject(error);
    }
}
```

**`saveToken()` 함수 반환값 상세**:

1. **반환 타입**: `Promise<void>` 또는 `Promise<boolean>`
   - 성공 시: `Promise<void>` (값 없음)
   - 또는 성공 여부를 반환하려면: `Promise<boolean>` (true/false)

2. **사용 예시**:
   ```javascript
   // 방법 1: await 사용
   const tokenSavePromise = saveToken(loginData.accessToken, loginData.refreshToken);
   await tokenSavePromise; // 완료 대기
   
   // 방법 2: .then() 사용
   const tokenSavePromise = saveToken(loginData.accessToken, loginData.refreshToken);
   tokenSavePromise
       .then(() => console.log('토큰 저장 완료'))
       .catch((error) => console.error('토큰 저장 실패:', error));
   
   // 방법 3: Promise.all()과 함께 사용 (병렬 처리)
   const wsPromise = connectWebSocket(loginData.uid, loginData.accessToken);
   const tokenSavePromise = saveToken(loginData.accessToken, loginData.refreshToken);
   await Promise.all([tokenSavePromise, wsPromise]); // 둘 다 완료 대기
   ```

3. **에러 처리**:
   - `localStorage` 저장 실패 시 (예: 저장 공간 부족)
   - 네트워크 오류 (원격 저장소 사용 시)
   - 권한 오류 (브라우저 설정)

**함수 반환값**:
- **타입**: `Promise<void>` 또는 `Promise<boolean>`
- **성공 시**: `Promise.resolve()` 또는 `Promise.resolve(true)`
- **실패 시**: `Promise.reject(error)` 또는 `throw new Error('저장 실패')`
- **사용 예시**:
  ```javascript
  const tokenSavePromise = saveToken(loginData.accessToken, loginData.refreshToken);
  // tokenSavePromise는 Promise 객체이므로 await 또는 .then()으로 처리 가능
  await tokenSavePromise; // 또는
  tokenSavePromise.then(() => console.log('토큰 저장 완료'));
  ```

###### 3-2. WebSocket 연결 요청

**현재 코드 위치**: 
- 라우터: `router/router.go:301` - `webrtc.GET("/ws", p.sig.HandleWebSocket)`
- 핸들러: `controller/signaling.go:286-316` - `HandleWebSocket()` 함수

**수정 필요 사항**:

**A. JWT 토큰 검증 추가 (현재 미구현)**

**수정 위치 1**: `router/router.go:277`
```go
// 현재: 인증 없음
webrtc := e.Group("webrtc/v01", p.SecurityHeaders())

// 수정 필요: JWT 인증 미들웨어 추가
webrtc := e.Group("webrtc/v01", p.SecurityHeaders(), p.JwtAuth())
```

**수정 위치 2**: `controller/signaling.go:286-293`
```go
// 현재: userId만 검증
func (p *SignalingController) HandleWebSocket(c *gin.Context) {
    userId := c.Query("userId")
    if userId == "" {
        // 에러 처리
    }
    
    // 수정 필요: JWT 토큰에서 userId 추출 및 검증
    // userID, exists := c.Get("user")  // JwtAuth 미들웨어에서 설정됨
    // if !exists {
    //     c.JSON(401, gin.H{"error": "Unauthorized"})
    //     return
    // }
    // userId := userID.(string)
}
```

**B. WebSocket 연결 시 Authorization 헤더 처리**

**수정 위치**: `controller/signaling.go:295-300`
```go
// 현재: 헤더 검증 없음
conn, err := p.upgrader.Upgrade(c.Writer, c.Request, nil)

// 수정 필요: Authorization 헤더에서 토큰 추출 및 검증
// authHeader := c.GetHeader("Authorization")
// if authHeader == "" {
//     c.JSON(401, gin.H{"error": "Authorization header required"})
//     return
// }
// tokens := strings.Split(authHeader, " ")
// if len(tokens) != 2 || tokens[0] != "Bearer" {
//     c.JSON(401, gin.H{"error": "Invalid authorization format"})
//     return
// }
// // 토큰 검증 로직 추가
```

**C. 클라이언트 측 WebSocket 연결 코드**

**수정 위치**: 클라이언트 코드 (프론트엔드)
```javascript
// 현재: userId만 전달
const ws = new WebSocket(`ws://server:port/webrtc/v01/ws?userId=${uid}`);

// 수정 필요: Authorization 헤더 추가 (WebSocket은 헤더 직접 설정 불가)
// 대안 1: 쿼리 파라미터로 토큰 전달 (보안상 권장하지 않음)
// const ws = new WebSocket(`ws://server:port/webrtc/v01/ws?userId=${uid}&token=${token}`);

// 대안 2: WebSocket 연결 전에 서버에서 연결 토큰 발급
// 1. POST /webrtc/v01/ws-token 요청 (JWT 필요)
// 2. 일회용 연결 토큰 받음
// 3. WebSocket 연결 시 해당 토큰 사용
```

---

##### 4단계: WebSocket 연결 성공 대기

**현재 코드 위치**: 
- 서버 측: `controller/signaling.go:311-315` - 고루틴 시작
- 클라이언트 측: 클라이언트 코드 (현재 문서화되지 않음)

**수정 필요 사항**:

**A. 연결 성공 확인 메시지 전송**

**수정 위치**: `controller/signaling.go:309-315`
```go
// 현재: 로그만 기록
log.Info(fmt.Sprintf("New WebSocket connection: %s", userId))

// 수정 필요: 클라이언트에게 연결 성공 메시지 전송
// connectionMsg := SignalingMessage{
//     Type:    "connection-established",
//     Payload: map[string]interface{}{
//         "userId": userId,
//         "timestamp": time.Now().Unix(),
//     },
// }
// data, _ := json.Marshal(connectionMsg)
// client.send <- data
```

**B. 클라이언트 측 연결 대기 로직**

**수정 위치**: 클라이언트 코드
```javascript
// 수정 필요: 연결 성공 대기 및 확인
const ws = new WebSocket(url);

ws.onopen = () => {
    console.log('WebSocket 연결 시도 완료');
    // 연결 성공 메시지 대기
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    if (message.type === 'connection-established') {
        console.log('WebSocket 연결 성공 확인');
        // 서비스 사용 시작
        startService();
    }
};

// 타임아웃 처리
const timeout = setTimeout(() => {
    if (ws.readyState !== WebSocket.OPEN) {
        console.error('WebSocket 연결 타임아웃');
        ws.close();
        // 재연결 시도
    }
}, 30000); // 30초 타임아웃
```

---

##### 5단계: 서비스 사용 시작

**현재 코드 위치**: 
- 서버 측: `controller/signaling.go:352-387` - `readPump()`, `writePump()` 함수
- 클라이언트 측: 클라이언트 코드

**수정 필요 사항**:

**A. 연결 상태 관리**

**수정 위치**: `controller/signaling.go:69-77` - `Client` 구조체
```go
// 현재 구조
type Client struct {
    conn     *websocket.Conn
    send     chan []byte
    room     *Room
    userName string
    mu       sync.Mutex
    lastSeen time.Time
}

// 수정 필요: 연결 상태 필드 추가
// type Client struct {
//     conn     *websocket.Conn
//     send     chan []byte
//     room     *Room
//     userName string
//     userID   string        // 추가: 사용자 ID 저장
//     isReady  bool          // 추가: 서비스 준비 상태
//     mu       sync.Mutex
//     lastSeen time.Time
// }
```

**B. 서비스 준비 완료 신호**

**수정 위치**: `controller/signaling.go:352-387` - `readPump()` 함수
```go
// 수정 필요: 클라이언트가 준비 완료 메시지 전송 시 처리
// case "ready":
//     client.isReady = true
//     // 다른 클라이언트에게 준비 완료 알림
```

**C. 클라이언트 측 서비스 시작**

**수정 위치**: 클라이언트 코드
```javascript
// 수정 필요: 연결 확인 후 서비스 시작
function startService() {
    // 1. 연결 상태 확인
    if (ws.readyState === WebSocket.OPEN) {
        // 2. 준비 완료 메시지 전송
        ws.send(JSON.stringify({
            type: 'ready',
            userId: currentUserId
        }));
        
        // 3. 서비스 기능 활성화
        enableChatFeatures();
        enableCallFeatures();
        startHeartbeat();
    }
}
```

---

#### 추가 수정 사항 요약

##### 서버 측 수정 필요 부분

1. **라우터 인증 추가** (`router/router.go:277`)
   - WebSocket 엔드포인트에 JWT 인증 미들웨어 추가

2. **WebSocket 핸들러 JWT 검증** (`controller/signaling.go:286-300`)
   - Authorization 헤더에서 토큰 추출 및 검증
   - JWT에서 userId 추출하여 사용

3. **연결 성공 메시지 전송** (`controller/signaling.go:311-315`)
   - 클라이언트에게 연결 성공 확인 메시지 전송

4. **Client 구조체 확장** (`controller/signaling.go:69-77`)
   - userID, isReady 필드 추가

##### 클라이언트 측 수정 필요 부분

1. **병렬 처리 로직**
   - 로그인 응답 후 토큰 저장과 WebSocket 연결을 동시에 실행

2. **연결 타임아웃 처리**
   - 30초 내 연결 실패 시 재시도 로직

3. **연결 확인 대기**
   - 서버로부터 연결 성공 메시지 수신 대기

4. **서비스 시작 로직**
   - 연결 확인 후 서비스 기능 활성화

---

## 메모리 저장 구조 상세

### SignalingController 구조

```go
type SignalingController struct {
    rooms       map[string]*Room  // 메인 저장소
    roomsMu     sync.RWMutex      // 동시성 제어
    totalConn   int64             // 전체 연결 수 (atomic)
    // ...
}
```

### Room 구조

```go
type Room struct {
    ID           int64
    UserIDs      map[*Client]bool       // 클라이언트 저장
    UserNames    map[*Client]string
    Broadcast    chan *BroadcastMessage
    Unregister   chan *Client
    mu           sync.RWMutex
    createdAt    time.Time
    lastActivity time.Time
}
```

### Client 구조

```go
type Client struct {
    conn     *websocket.Conn  // 실제 WebSocket 연결
    send     chan []byte      // 송신 버퍼
    room     *Room            // 속한 방 참조
    userName string
    mu       sync.Mutex
    lastSeen time.Time
}
```

---

## 연결 관리 및 정리

### 자동 정리 프로세스

**코드 위치**: `controller/signaling.go:210-240` - `roomCleaner()`

**동작**:
- 1분마다 실행
- 빈 방(연결 없는 방) 검사
- 5분 이상 비활성 방 자동 삭제

**정리 조건**:
```go
isEmpty := len(room.UserIDs) == 0
inactive := time.Since(room.lastActivity) > 5*time.Minute

if isEmpty && inactive {
    delete(r.rooms, name)
}
```

### 수동 연결 해제

**코드 위치**: `controller/signaling.go:337-350` - `unregisterClient()`

**트리거**:
- WebSocket 연결 종료
- 읽기/쓰기 에러 발생
- 클라이언트 명시적 종료

**처리 내용**:
1. `rooms` 맵에서 클라이언트 제거
2. `client.send` 채널 종료
3. WebSocket 연결 종료
4. 사용자 목록 브로드캐스트

---

## 연결 상태 확인

### 현재 연결된 사용자 조회

**엔드포인트**: `GET /webrtc/v01/connected-users`

**응답 예시**:
```json
{
  "users": ["user1", "user2", "user3"],
  "count": 3
}
```

**코드 위치**: `controller/signaling.go:536-549`

---

## 에러 처리 및 재연결

### 일반적인 에러 상황

1. **로그인 실패**
   - 사용자 정보 오류
   - 비밀번호 불일치
   - **처리**: 에러 메시지 반환, WebSocket 연결 시도 안 함

2. **WebSocket 연결 실패**
   - 서버 다운
   - 네트워크 오류
   - **처리**: 재연결 로직 구현 필요

3. **연결 타임아웃**
   - 60초 동안 Pong 응답 없음
   - **처리**: 자동으로 연결 종료, 재연결 필요

### 재연결 전략

**권장 구현**:
```javascript
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

function connectWebSocket(uid) {
    const ws = new WebSocket(`ws://server:port/webrtc/v01/ws?userId=${uid}`);
    
    ws.onclose = () => {
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            setTimeout(() => connectWebSocket(uid), delay);
        }
    };
    
    ws.onopen = () => {
        reconnectAttempts = 0; // 재연결 성공 시 카운터 리셋
    };
}
```

---

## 보안 고려사항

### 현재 구현 상태

1. **인증**: 
   - 로그인 시 JWT 토큰 발급
   - WebSocket 연결 시 userId만 사용 (JWT 검증 없음)

2. **CORS**: 
   - 개발 중 모든 Origin 허용
   - 프로덕션에서는 제한 필요

### 프로덕션 권장사항

1. **JWT 토큰 검증**:
   - WebSocket 연결 시 Authorization 헤더로 토큰 전달
   - 서버에서 토큰 검증 후 연결 허용

2. **사용자 검증**:
   - userId가 실제 존재하는 사용자인지 DB 확인
   - 토큰의 userId와 쿼리 파라미터의 userId 일치 확인

3. **Rate Limiting**:
   - 연결 시도 횟수 제한
   - 메시지 전송 빈도 제한

---

## 성능 최적화

### 메모리 관리

1. **버퍼 크기**:
   - 송신 버퍼: 256 메시지
   - 읽기 버퍼: 4096 bytes
   - 쓰기 버퍼: 4096 bytes

2. **연결 제한**:
   - 최대 방 수: 5000
   - 최대 연결 수: 25000

3. **자동 정리**:
   - 1분마다 빈 방 정리
   - 5분 이상 비활성 방 삭제

### 동시성 제어

1. **RWMutex 사용**:
   - 읽기 작업: 여러 고루틴 동시 접근 가능
   - 쓰기 작업: 단일 고루틴만 접근

2. **고루틴 관리**:
   - 클라이언트당 2개 고루틴 (읽기/쓰기)
   - 워커 풀: 50개 메시지 처리 워커
   - 브로드캐스트 워커: 15개

---

## 모니터링 및 로깅

### 주요 로그 메시지

1. **연결 성공**:
   ```
   New WebSocket connection: {userId}
   ```

2. **연결 종료**:
   ```
   Client disconnected: {userId}
   ```

3. **메시지 중계**:
   ```
   Message relayed: {type} from {from} to {to}
   ```

4. **방 정리**:
   ```
   Cleaned up inactive room: {roomId}
   ```

### 모니터링 지표

1. **연결 수**: `totalConn` (atomic)
2. **활성 방 수**: `len(rooms)`
3. **메시지 처리량**: 워커 큐 사용률
4. **브로드캐스트 처리량**: 브로드캐스트 큐 사용률

---

## 요약

### 로그인 → WebSocket 연결 프로세스

1. ✅ **로그인 API 호출** → JWT 토큰 발급
2. ✅ **WebSocket 연결 요청** → `userId` 파라미터로 연결
3. ✅ **메모리 저장** → `SignalingController.rooms` 맵에 저장
4. ✅ **읽기/쓰기 고루틴 시작** → 메시지 송수신 준비 완료
5. ✅ **자동 정리** → 비활성 연결 자동 제거

### 핵심 포인트

- **메모리 저장 위치**: `SignalingController.rooms` 맵
- **저장 단위**: `Room` 구조체 내 `UserIDs` 맵
- **동시성 제어**: `sync.RWMutex` 사용
- **자동 정리**: 1분 주기로 비활성 방 삭제
- **연결 유지**: Ping/Pong 메커니즘 (54초 주기)

---

*문서 생성일: 2025-01-27*
*관련 파일:*
- `/home/jino/go/src/ms-gateway/controller/account.go` (로그인)
- `/home/jino/go/src/ms-gateway/controller/signaling.go` (WebSocket)
- `/home/jino/go/src/ms-gateway/router/router.go` (라우팅)

