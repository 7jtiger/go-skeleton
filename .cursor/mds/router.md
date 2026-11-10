# Router Package Function Feature Summary

## mdlware.go - Middleware Functions

### Security and Authentication Middleware
- `RateLimiter()`: Rate limiter middleware that prevents excessive requests by limiting request frequency per IP
- `CORS()`: Middleware that sets HTTP headers to allow Cross-Origin Resource Sharing
- `JwtAuth()`: Middleware that handles user authentication by validating JWT tokens
- `OptionalJwt()`: Optional authentication middleware that validates JWT tokens if present but allows passage without them
- `CheckBlacklist()`: Middleware that checks if client IP is included in the blacklist
- `SecurityHeaders()`: Middleware that sets security headers to prevent XSS, clickjacking, MIME sniffing, etc.

### Data Processing Middleware
- `GetReqXMeta()`: Middleware that parses user metadata from x-meta header and stores it in context
- `ValidateFileUpload()`: 멀티파트 업로드 검증. **이미지·썸네일 ≤ size MB(기본 5MB)**, **동영상 ≤ 100MB**. 이미지만이면 `files`만 허용(`thbnl` 첨부 시 400). 동영상 포함 시 `thbnl` 1:1 필수. context: `upFiles`, `upThumb`, `sinfo`.
- `AesEncrypt()`: Middleware that encrypts response data using AES GCM mode for transmission
- `AesDecrypt()`: Middleware that decrypts request data using AES GCM mode for processing

### Utility Middleware
- `ParsePagination()`: Function that parses pagination information from query parameters and returns it
- `Pagination()`: Middleware that validates pagination information and stores it in context

---

## router.go - Router Configuration Functions

### Router Initialization and Configuration
- `NewRouter()`: Constructor function that creates router instance and connects controllers
- `convertWhiteList()`: Utility function that converts IP whitelist slice to map
- `Idx()`: Main router function that initializes Gin engine and sets up all routes. **2026-07 최적화**: `prod` 모드면 엔진 생성 전에 `gin.SetMode(gin.ReleaseMode)` 호출(생성 후 호출은 미적용), `gin.Default()` 대신 `gin.New()`를 사용해 기본 Logger/Recovery와 커스텀 `GinLogger`/`GinRecovery`의 이중 실행 제거

### Authentication Related Functions
- `validateOTP()`: Function that validates the validity of OTP (One-Time Password) tokens
- `liteAuth()`: Middleware that performs lightweight authentication (OTP validation currently disabled)
- `otpAuth()`: Middleware that performs advanced authentication through IP whitelist and OTP

---

## Overall Structure Summary

### mdlware.go (12 functions)
- **Security & Authentication**: Rate limiter, CORS, JWT auth, blacklist, security headers (5 functions)
- **Data Processing**: Metadata parsing, file upload validation, AES encryption/decryption (4 functions)
- **Utilities**: Pagination parsing and validation (2 functions)
- **Optional Authentication**: Optional JWT (1 function)

### router.go (6 functions)
- **Router Configuration**: Constructor, whitelist conversion, main router (3 functions)
- **Authentication**: OTP validation, lightweight auth, advanced auth (3 functions)

### Main Route Groups
- **serv/v01**: Basic service information (version, WebRTC configuration)
- **acc/v01**: Account management (registration, login, profile updates, etc.)
- **webrtc/v01**: WebRTC related features (configuration, call status, STUN testing)
- **user/v01, home/v01, present/v01, etc.**: Various service-specific endpoints

---

## Security Features
- **Multi-layer Security**: Rate limiter → CORS → JWT → Blacklist → Security headers
- **Data Protection**: AES-256 GCM encryption for request/response data protection
- **Access Control**: IP whitelist and OTP-based authentication

---

## API Routing URL Function Summary

### Basic Endpoints
- `GET /swagger/:any`: Endpoint that provides Swagger API documentation
- `GET /test`: Simple test endpoint to check server status

### serv/v01 - Server Information (Security Headers Applied)
- `GET /serv/v01/version`: Endpoint to retrieve service version information
- `GET /serv/v01/wsconf`: Endpoint to retrieve WebRTC configuration information

### acc/v01 - Account Management (Security Headers Applied)
- `GET /acc/v01/check/:id`: Endpoint to check user ID duplication
- `GET /acc/v01/ckemail/:email`: Endpoint to check email duplication
- `POST /acc/v01/regist`: Endpoint to register new users (AES decryption applied)
- `POST /acc/v01/login`: Endpoint to handle user login (AES decryption applied)
- `POST /acc/v01/refresh`: Endpoint to rotate access/refresh token pair using refresh token (Authorization header or body `refTok`)
- `POST /acc/v01/fnid`: Endpoint to find user ID by name and birth date
- `POST /acc/v01/fnpw`: Endpoint to find password by ID, email, and birth date
- `POST /acc/v01/cngpw`: Endpoint to change user password
- `POST /acc/v01/verifyotp`: Endpoint to verify OTP token

### inserv/v01 - Internal Service (Security Headers + JWT Auth)
- `POST /inserv/v01/modify`: Endpoint to modify user information (category: pw/area/nick/email)
- `POST /inserv/v01/upd/mpic`: Endpoint to update user main profile image (AES decryption applied)
- `POST /inserv/v01/logout`: Endpoint to handle user logout
- `POST /inserv/v01/leave`: Endpoint to handle user withdrawal
- `GET /inserv/v01/info/:id`: Endpoint to retrieve user information
- `POST /inserv/v01/delete/:id`: Endpoint to delete user account

### user/v01 - User Information (Lightweight Authentication Required)
- `GET /user/v01/myinfo`: Endpoint to retrieve user's balance and personal information
- `GET /user/v01/wyinfo`: Endpoint to retrieve user's favorites and call history

### home/v01 - Home Screen (Security Headers + JWT Auth)
- `GET /home/v01/mdata`: Endpoint to retrieve male home screen data
- `GET /home/v01/wdata`: Endpoint to retrieve female home screen data

### present/v01 - Gift Feature (Lightweight Authentication Required)
- `PUT /present/v01/target`: Endpoint to set gift target

### mission/v01 - Mission Feature (Lightweight Authentication Required)
- `GET /mission/v01/list`: Endpoint to retrieve mission list
- `GET /mission/v01/detail/:id`: Endpoint to retrieve specific mission details

### story/v01 - Story Feature (Security Headers Applied)
- `GET /story/v01/prf/info/:tid`: 프로필 + 팔로워 수 (`PrfInfoResp`: `prf_Info`, `flw_cnt`)
- `GET /story/v01/prf/assets`: JWT 본인 자산 (화살/캐시/쪽지, `PrfAssetsResp`)
- `GET /story/v01/prf/pic/list`: JWT 본인 프로필 사진 슬롯 맵 (`PrfPicListResp`)
- `GET /story/v01/prf/story/lists/:tid`: 대상 UID 최근 스토리 대표 미디어 (`PrfStoryListsResp`)
- `POST /story/v01/prf/upd`: JWT 본인 프로필 부분 수정 (`PrfUpdResp`)
- `POST /story/v01/prf/set/mainpic/:url`: 대표사진 설정/삭제 (`PrfUpdResp`, URL PathEscape)
- `POST /story/v01/prf/pic/upload`: 프로필 사진 업로드 (`ValidateFileUpload(5,5)`, `PrfPicUploadResp`)
- `POST /story/v01/prf/pic/delete/:idx`: 프로필 사진 슬롯 삭제+shift (`RespHeader`)
- `GET /story/v01/prf/pic/waiting/:page/:limit`: 승인 대기 프로필 사진 목록 (`PrfPicWaitingListResp`)
- `POST /story/v01/prf/pic/stat`: 프로필 사진 승인 (`SetPrfPicStatResp`)
- `GET /story/v01/home/:id`: Endpoint to retrieve user's story home (placeholder)
- `GET /story/v01/list/:page/:limit`: 본인 스토리 기본 목록 (JWT UID, `StoryListResp`)
- `GET /story/v01/list/:page/:limit/:tid`: 타인 스토리 기본 목록 (`tid` = 대상 UID)
- `GET /story/v01/condition/list/:area/:stat/:type/:order/:gen/:page/:limit`: 조건부 피드 (`StoryListResp` 배열, JWT 시 cutout 제외)
- `GET /story/v01/detail/:idx/:page/:limit`: 본인 스토리 상세 목록 (JWT UID, `idx` 이하·최신순)
- `GET /story/v01/detail/:idx/:page/:limit/:tid`: 타인 스토리 상세 목록 (`tid` = 작성자 UID)
- `POST /story/v01/upload`: 스토리 미디어만 Cloudflare 업로드 (Images/Stream). `ValidateFileUpload` (이미지·썸네일 5MB, 동영상 100MB). 응답 **`data.media`** 통합 슬롯 맵 `{"N":{"type":"img|vdo","url":"...","thumb":"..."}}`
- `POST /story/v01/create`: JSON 스토리 생성 — body `{stat,sbody,media}`(upload `data.media`), JWT 프로필 필드 포함 INSERT, 응답 `{msg,idx}`
- `POST /story/v01/updstat`: 스토리 상태 변경 — body `{idx,stat}` (0=del,1=pub,2=private,3=limit,4=resv)
- `POST /story/v01/updbody`: 스토리 본문 수정 — body `{idx,body}` (최대 512자)
- `POST /story/v01/delpic`: 스토리 미디어 슬롯 삭제 — body `{idx,pic_idx}`, bak 이동 + 1..N 재번호
- `POST /story/v01/comment/create`: 스토리 댓글 작성 (`StrCmtCreateResp`, SimpleRespOK)
- `GET /story/v01/comment/:idx/:page/:limit`: 스토리 댓글 페이징 목록 (`StrCmtDetailResp`)
- `POST /story/v01/comment/updstat`: 댓글 상태 변경 (`StrCmtUpdStatResp`, SimpleRespOK)
- `POST /story/v01/comment/updbody`: 댓글 본문 수정 (`StrCmtUpdBodyResp`, SimpleRespOK)
- `POST /story/v01/like/toggle`: 좋아요 토글 (`story_idx`,`uid`) → `{liked,like_count}` (`ToggleStoryLikeTx`)
- `POST /story/v01/follow/set/:tid`: 팔로우 설정 (JWT=follower, tid=followee, upsert stat=1)
- `POST /story/v01/follow/cancel/:tid`: 언팔로우 (JWT=follower, tid=followee, `stat=0`)
- `GET /story/v01/follower/list/:page/:limit`: JWT 본인 팔로워 목록 (`FollowUserItem[]`, total_count)
- `GET /story/v01/following/list/:page/:limit`: JWT 본인 팔로잉 목록 (`FollowUserItem[]`, total_count)
- `POST /story/v01/cutout/set/:tid`: 스토리 cutout 설정 (대상 feed 숨김)
- `POST /story/v01/cutout/cancel/:tid`: 스토리 cutout 해제 (`stat=0`, 응답 `CutoutActionResp`)
- `POST /story/v01/cutout/count`: 활성 cutout 개수 조회 (응답 `CutoutCountResp`)
- `GET /story/v01/cutout/list/:page/:limit`: 활성 cutout 목록 (응답 `CutoutListResp`)
- `GET /story/v01/pcs/list/:page/:limit`: JWT 본인이 구매한 스토리 목록 (`str_pcs` → `story`, 최신순)
- `POST /story/v01/buy/:idx`: JWT 본인이 스토리 구매 기록 (`str_pcs` INSERT, 결제 TODO)
- `POST /story/v01/view/update/:idx`: 스토리 조회수(`qt_view`) +1. Redis `story:view:{uid}:{idx}`가 있으면 24시간 동안 건너뜀 (`UpdateViewHistory`)
- `GET /story/v01/view/count/:idx`: 스토리 조회수(`qt_view`) 조회
- `GET /story/v01/like/count/:idx`: 스토리 좋아요 수(`qt_good`) 조회
- `POST /story/v01/block/create`: Block user endpoint
- `POST /story/v01/block/cancel`: Unblock user endpoint
- `GET /story/v01/block/list/:uid/:page/:limit`: Blocked user list endpoint (page starts at 1, limit 1~100)

### upload/v01 - File Upload (Lightweight Authentication Required)
- `POST /upload/v01/story/img`: Endpoint to upload images for stories
- `POST /upload/v01/present`: Endpoint to upload images for gifts

### setting/v01 - Settings (Lightweight Authentication Required)
- `GET /setting/v01/alert`: Endpoint to retrieve notification settings

### noti/v01 - Notifications (Security Headers + JWT Auth)
- `GET /noti/v01/list`: Endpoint to retrieve personal notification list
- `GET /noti/v01/detail/:idx`: Endpoint to retrieve specific notification detail and mark as read
- `GET /noti/v01/anc/list`: Endpoint to retrieve public announcement list
- `GET /noti/v01/anc/detail/:idx`: Endpoint to retrieve specific announcement detail

### dm/v01 - Text Chat Features (Security Headers)
- `GET /dm/v01/ws`: WebSocket endpoint for real-time text chat (query param: userId)
- `POST /dm/v01/mkroom/:pid`: DM 방 생성/재참여 (`CreateChatRoom`)
- `POST /dm/v01/rmroom/:room_id`: DM 방 나가기 — 1명만 전이 (`DeleteChatRoom`)
- `GET /dm/v01/list/:page`: 사용자 DM 채팅방 목록 — 정렬: unread>0 우선 → `:ts` 최근 수신 → `at_update` (`partner_left` 포함)
- `GET /dm/v01/rinfo/:room_id`: DM 채팅방 단건 조회 (상대방 정보, last_msg, unread, partner_left)
- `GET /dm/v01/history/:room_id`: DM 채팅 히스토리 (커서 페이지네이션, query: `limit`, `cursor`, `pgSize`)
- `POST /dm/v01/message`: REST API endpoint to send message (alternative to WebSocket)
- `GET /dm/v01/mlistvd`: Endpoint to retrieve male video chat list
- `GET /dm/v01/wlistvd`: Endpoint to retrieve female video chat list
- `GET /dm/v01/mlistvo`: Endpoint to retrieve male voice chat list
- `GET /dm/v01/wlistvo`: Endpoint to retrieve female voice chat list

### inbox/v01 - Inbox Management (Security Headers + JWT)
- `GET /inbox/v01/list`: DM 방 목록 조회 (`chat.GetChatRooms` 재사용)
- `GET /inbox/v01/unread`: 전체 unread 합계 조회

### webrtc/v01 - WebRTC Features (JWT Authentication Required)
- `GET /webrtc/v01/config`: Endpoint to retrieve WebRTC configuration information
- `GET /webrtc/v01/available-users`: Endpoint to retrieve list of users available for calls
- `POST /webrtc/v01/call-status`: Endpoint to update user's call status
- `POST /webrtc/v01/heartbeat`: Endpoint to update WebRTC session heartbeat
- `GET /webrtc/v01/stats`: Endpoint to retrieve WebRTC system statistics
- `GET /webrtc/v01/test-stun`: Endpoint to test STUN server connectivity
- `POST /webrtc/v01/test-stun-server`: Endpoint to test specific STUN server
- `GET /webrtc/v01/ws`: WebSocket signaling endpoint for real-time P2P video chat communication
- `GET /webrtc/v01/connected-users`: Endpoint to retrieve list of currently connected users via WebSocket

---

## Classification by Authentication Level
- **No Authentication**: Basic endpoints (swagger, test)
- **Security Headers Only**: Account management (acc/v01), chat, inbox
- **Lightweight Authentication (liteAuth)**: User information, gifts, missions, stories, uploads, settings
- **JWT Authentication**: Internal services (inserv/v01), home screen, notifications, WebRTC features

---

## Router Structure Updates

### Router Initialization
- `NewRouter()`: Initializes router with 5 main controllers: Account, Profile, Home, Noti
- Added controller references: `acc`, `pf`, `hm`, `nt` (chat controller commented out)

### Authentication Middleware Chain
1. **SecurityHeaders()**: Applied to all route groups for XSS, clickjacking protection
2. **liteAuth()**: Lightweight OTP authentication (currently disabled for development)
3. **JwtAuth()**: Full JWT token authentication for protected routes
4. **otpAuth()**: IP whitelist + OTP validation for admin operations

---

## WebSocket Endpoints

### 1. WebRTC Signaling WebSocket
- **URL**: `ws://server:port/webrtc/v01/ws?userId={userId}`
- **Protocol**: WebSocket
- **Authentication**: No Auth (개발용, 프로덕션에서는 JWT 권장)
- **Purpose**: Real-time P2P video/audio chat signaling

**Query Parameters:**
- `userId` (required): User unique identifier

**Message Format:**
```json
{
  "type": "offer|answer|ice-candidate|hangup|user-list|error",
  "from": "sender ID",
  "to": "receiver ID",
  "payload": "message data"
}
```

**Message Types:**
- `offer`: WebRTC Offer message
- `answer`: WebRTC Answer message
- `ice-candidate`: ICE Candidate information
- `hangup`: Call termination signal
- `user-list`: Connected users list update (server → client)
- `error`: Error message (server → client)

### 2. Text Chat WebSocket
- **URL**: `ws://server:port/dm/v01/ws?userId={userId}`
- **Protocol**: WebSocket
- **Authentication**: No Auth (개발용, 프로덕션에서는 JWT 권장)
- **Purpose**: Real-time text messaging

**Query Parameters:**
- `userId` (required): User unique identifier

**Message Format:**
```json
{
  "type": "text-message|typing|read-receipt|call-request|call-accept|call-reject",
  "from": "sender ID",
  "to": "receiver ID",
  "roomId": "chat room ID",
  "content": "message content",
  "timestamp": 1234567890
}
```

**Message Types:**
- `text-message`: Text chat message
- `typing`: Typing indicator
- `read-receipt`: Message read confirmation
- `call-request`: Call request notification
- `call-accept`: Call acceptance notification
- `call-reject`: Call rejection notification

**Connection Features:**
- **Ping/Pong**: 54s interval ping, 60s pong timeout
- **Buffer Size**: 256 messages per client
- **Max Message Size**: 64KB
- **Message Batching**: Automatic batching of queued messages

---
*Last Updated: 2026-02-02*

