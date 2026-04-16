# Controller Package Function Feature Summary

## controller.go - Main Controller Core

### Controller Initialization
- `NewCTL()`: Creates main controller instance, initializes sub-controllers (Account, Profile), and sets up repository connections
- `NewCTL()`: FCMPusher 활성화 포함(초기화 실패 시 부팅 에러 반환)

### Response Management Functions
- `SimpleRespOK()`: Sends simple HTTP 200 OK response with JSON payload
- `RespError()`: Sends error response with status code, logs error message based on severity (server vs client error)
- `SimpleError()`: Sends standardized error response using protocol header format with logging
- `RespSuccess()`: Sends success response with HTTP 200 status and JSON payload
- `SendResponse()`: Sends response with string data wrapped in protocol response header
- `SendDataResponse()`: Sends response with data payload wrapped in protocol data response header

### Utility Functions
- `GetPaging()`: Calculates pagination parameters (page, limit, total pages) from string inputs
- `GetController()`: Returns specific controller instance based on type (AccountController, ProfileController)
- `GetRedis()`: Returns Redis database connection instance

---

## types.go - Utility Types and Functions

### Default Values and Error Definitions
- **Constants**: `defaultGasLimit` (100000000), error types `NotFoundChain`, `notFound`, `unknownUser`

### Helper Functions
- `joinMsg()`: Concatenates multiple interface{} arguments into a single formatted string message
- `GetRandDefIcon()`: Returns random default profile icon URL based on gender (male/female with 4 options each)
- `GetRandDefIntroImg()`: Returns random default intro image URL based on gender (male: 4 options, female: 12 options)

---

## account.go - Account Management Controller

### Controller Initialization
- `NewAccountController()`: Creates AccountController instance with database connections and repository setup

### API Version and Service Info
- `GetVersion()`: Returns service version information (0.9.1) for API status endpoint

### User Validation Functions
- `CheckID()`: Validates user ID availability by checking for duplicates in database
- `CheckEmail()`: Validates email availability by checking for duplicates in database

### User Authentication Functions
- `RegistUserInfo()`: Registers new user with encrypted data, generates default profile settings, and validates required fields
- `LoginUser()`: Authenticates user login, creates JWT access/refresh tokens, stores session in Redis, initializes WebRTC session with device/network info, sets x-meta header, and returns WebRTC configuration
- `genLoginUserToken()`: Internal function that generates JWT access token (24h) and refresh token (14d), stores them in Redis AUTH:ACCESS and AUTH:REFRESH hashes with user information
- `RefreshToken()`: Validates refresh token (header/body), verifies JWT signature and Redis AUTH:REFRESH session, rotates access/refresh token pair atomically, and returns renewed tokens with x-meta header
- `LogoutUser()`: Processes user logout, deletes JWT token from Redis, and updates last access time

### Account Management Functions
- `LeaveUser()`: Handles user account withdrawal by updating status to inactive (stat=4)
- `DeleteUser()`: Soft deletes user account by changing status to 4 instead of actual deletion

### Account Recovery Functions
- `FindID()`: Searches for user IDs using name and birth date, returns up to 10 matching IDs
- `FindPW()`: Validates password recovery request using ID, email, and birth date combination
- `ChangePW()`: Changes user password after validating current password and user credentials

### User Information Management
- `ModifyUserInfo()`: Updates user information (nickname, area, email) based on category parameter
- `GetUserInfo()`: Retrieves user profile information by ID with encrypted data decryption
- `ModifyMainPic()`: Updates user main profile picture with file upload validation (placeholder implementation)

### WebRTC Functions
- `GetWebRTCConfig()`: Retrieves WebRTC configuration for authenticated user from Redis (JWT required)
- `GetAvailableUsers()`: Returns list of users available for video calls based on current status (JWT required, excludes self)
- `UpdateCallStatus()`: Updates user's call status (isInCall, callWith) in Redis WebRTC session (JWT required)
- `UpdateWebRTCHeartbeat()`: Updates WebRTC session heartbeat, device info, and network info to maintain active session (JWT required)
- `GetWebRTCStats()`: Retrieves WebRTC system statistics for monitoring and admin purposes (total sessions, available users, active calls)

### STUN Server Testing Functions
- `TestStunServers()`: Tests connectivity to recommended STUN servers and returns connection report
- `TestSpecificStunServer()`: Tests connectivity to specific STUN server URL and returns detailed results

---

## webrtc.go - WebRTC Connectivity Testing

### STUN Testing Data Structures
- `STUNTestResult struct`: STUN server test result containing success flag, server URL, public IP/port, response time, and error message
- `ICEConnectivityTest struct`: ICE connectivity test result with STUN server results array and summary statistics

### STUN Server Testing Functions
- `TestStunServer()`: Tests single STUN server connectivity with timeout, parses STUN URL, creates binding request, and returns public IP/port mapping
- `TestMultipleStunServers()`: Tests multiple STUN servers concurrently using goroutines and channels, returns aggregate test results with success/failure counts

### STUN Protocol Functions
- `parseStunURL()`: Parses STUN URL format (stun:host:port) and extracts host and port components with default port 19302
- `createStunBindingRequest()`: Creates STUN Binding Request packet with message type 0x0001, magic cookie, and random transaction ID
- `parseStunResponse()`: Parses STUN response packet, validates transaction ID, extracts MAPPED-ADDRESS or XOR-MAPPED-ADDRESS attributes

### WebRTC Configuration Functions
- `GetRecommendedStunServers()`: Returns list of 5 recommended Google STUN servers for WebRTC connectivity testing
- `ValidateWebRTCConfig()`: Validates WebRTC configuration format, checks ICE servers, URLs, and ensures proper stun:/turn: prefix format
- `CreateStunTestReport()`: Creates comprehensive STUN connectivity test report with timestamp, summary statistics, success rate, and detailed results per server

---

## content.go - Content Management

### Content Controller
- `NewContentController()`: Creates ContentController instance for content management operations
- `GetTestContent()`: Simple test endpoint that returns success message for content controller validation

---

## home.go - Home Screen Controller

### Home Controller
- `NewHomeController()`: Creates HomeController instance with database connections for home screen data management
- `GetHomeMenInfo()`: Retrieves home screen information including notifications, messages, check-in status, video chat list, voice chat list, stories, and terms/policy links

---

## noti.go - Notification and Announcement Management

### Notification Controller
- `NewNotiController()`: Creates NotiController instance with database connections for notification management

### Notification Functions
- `GetNotiList()`: Retrieves user's notification list with parameter validation
- `GetNoti()`: Fetches all notifications for a specific user ID from database
- `GetNotiNewCount()`: Returns count of unread notifications for a user
- `GetNotiDetail()`: Retrieves notification detail by index and marks it as read automatically
- `SetNoti()`: Placeholder for setting notification (implementation pending)

### Announcement Functions
- `GetAnnouncement()`: Retrieves public announcement list (new badge for items within one week)
- `SetAnnouncement()`: Saves new announcement with admin permission validation (title, body, URL required)
- `GetAnnouncementDetail()`: Retrieves specific announcement details by index with not found error handling

---

## profile.go - Profile Management Controller

### Profile Controller
- `NewProfileController()`: Creates ProfileController instance for user profile management operations

---

## item.go - Item Management Controller

### Item Controller
- `NewItemController()`: Creates ItemController instance for item/product management operations

---

## history.go - History Management Controller

### History Controller
- `NewHistoryController()`: Creates HistoryController instance for user history and activity tracking

---

## ctl_test.go - Testing Functions

### OTP Testing Functions
- `genOtp()`: Generates OTP (One-Time Password) code using TOTP algorithm with secret key
- `TestGenOtp()`: Unit test function for OTP generation with base32 encoding and time validation

### HTTP Testing Functions
- `Get()`: Performs HTTP GET request with query parameters for API endpoint testing
- `PostJson()`: Performs HTTP POST request with JSON payload for API endpoint testing
- `PostJsonS()`: Performs HTTP POST request with string body for raw data testing
- `PostJsonData()`: Performs HTTP POST request with complex JSON data structure
- `Post()`: Standard HTTP POST request with form data for endpoint testing
- `PostWithHeader()`: HTTP POST request with custom headers for authentication testing

### File Upload Testing Functions
- `PostFile()`: Uploads single file via HTTP POST with multipart form data
- `PostFiles()`: Uploads multiple files via HTTP POST with multipart form data handling

### API Endpoint Testing Functions
- `TestGetCtx()`: Tests GET endpoint with query parameters and response validation
- `TestPostCtx()`: Tests POST endpoint with JSON payload and response validation
- `TestFileUpload()`: Tests file upload functionality with multipart form data
- `TestMultipleFileUpload()`: Tests multiple file upload with size and type validation
- `TestAccountRegister()`: Tests user registration endpoint with encrypted data payload
- `TestAccountLogin()`: Tests user login endpoint with credential validation
- `TestWebRTCEndpoints()`: Tests WebRTC-related endpoints including configuration and status updates

---

## imgCtl - Image Processing and NSFW Detection

### NSFW Detection System
- `NewNSFWDetector()`: Creates NSFW detector instance with TensorFlow model for content moderation
- `CheckImage()`: Analyzes image for NSFW content using machine learning model and returns scores
- `imageToTensor()`: Converts image to TensorFlow tensor format for model input processing
- `Close()`: Releases TensorFlow model resources and closes detector session

### Image Processing Functions
- `handleImageCheck()`: Gin handler for image upload and NSFW detection with temporary file management

---

## storyCtl.go - Story Management Controller

### Story Controller Initialization
- `NewStoryController()`: Creates StoryController instance with AccountDB, HistoryDB, and StoryDB connections

### Story Management Functions
- `GetStoryHomeList()`: Retrieves story home list for a specific user (currently connected opposite gender users)
- `GetStoryList()`: Retrieves all public stories (stat=1 or 0) for a user, ordered by creation time descending
- `GetStoryDetail()`: Retrieves detailed story information including all comments for a specific story index
- `UploadStoryPic()`: Uploads story images to Cloudflare Images (max 5 files, 5MB each), saves story with body text, nickname, and status

### Story Update Functions
- `UpdateStoryStat()`: Updates story status (0=deleted, 1=public, 2=private, 3=limited, 4=reserved)
- `UpdateStrBody()`: Updates story body content (max 512 characters)
- `DeleteStrPic()`: Deletes specific picture from story and moves it to backup (str_imgbak)

### Story Comment Functions
- `CreateStrComment()`: Creates new comment on a story with writer uid, nickname, body (max 256 chars), and status
- `CreateStrComment()`: 작성자 메타데이터(`thumb_url`, `wgender`, `wage`, `warea`)를 함께 저장
- `GetStrCmtDetail()`: `idx/page` 기반 댓글 페이지 조회
- `UpdateStrStatComment()`: Updates comment status (0=default, 1=private, 2=reserved, 3=reserved, 4=deleted)
- `UpdateStrBodyComment()`: Updates comment body content

### Cloudflare Integration
- **Image Upload**: Uses `utils.UploadCldFlr()` to upload images to Cloudflare Images
- **Image Storage**: Stores image URLs as JSON array with indexed keys (1, 2, 3, ...)
- **Configuration**: Uses `cfg.Server.CfId` and `cfg.Server.CfToken` for Cloudflare API authentication

### Key Features
- **Multiple File Upload**: Supports up to 5 images per story
- **JSON Image Storage**: Images stored as JSON with numeric indexes
- **Backup System**: Deleted images moved to str_imgbak instead of permanent deletion
- **Status Management**: Flexible status system for stories and comments
- **Validation**: Checks required fields (uid, nickname, body, status)

---

## chatCtl.go - Text Chat Controller

### Chat Controller Initialization
- `NewChatController()`: Creates ChatController instance with Redis connection for real-time messaging

### WebSocket Connection Management
- `HandleWebSocket()`: Handles WebSocket connection upgrade and client registration
- `registerClient()`: Registers new client to active connections map, closes old connection if exists
- `unregisterClient()`: Removes client from active connections and cleans up resources

### Message Processing Functions
- `readPump()`: Goroutine that reads messages from WebSocket connection with ping/pong timeout
- `writePump()`: Goroutine that writes messages to WebSocket connection with batching
- `handleMessage()`: Routes incoming messages by type (text-message, typing, read-receipt, call-request, etc.)
- `sendToUser()`: Sends message to specific user by userID

### Message Handler Functions
- `handleTextMessage()`: Processes text messages and forwards to recipient
- `handleTyping()`: Handles typing indicators and forwards to recipient
- `handleReadReceipt()`: Handles read receipts for message delivery confirmation

### Chat Room Management (HTTP API)
- `CreateChatRoom()`: Creates new chat room with name, creator, and privacy settings
- `GetChatRooms()`: Retrieves user's chat room list
- `GetChatHistory()`: Retrieves paginated chat history for a room
- `SendMessage()`: REST API endpoint for sending messages (alternative to WebSocket)

### Call Integration
- `SendCallNotification()`: Sends call-related notifications through chat WebSocket (called by SignalingController)
- **Call Message Types**: call-request, call-accept, call-reject

### Key Features
- **WebSocket-based**: Real-time bidirectional communication
- **Concurrent Connections**: Thread-safe multi-client connection handling
- **Message Types**: text-message, typing, read-receipt, call notifications
- **Ping/Pong**: 54s interval ping, 60s timeout for connection health
- **Message Batching**: Efficient batch transmission of queued messages
- **Buffer Management**: 256 message buffer per client with overflow protection

---

## Overall Structure Summary

### Main Controllers (18+ controllers)
- **Controller**: Core controller management (9 functions)
- **AccountController**: User account operations (18 functions)
- **StoryController**: Story and comment management (12 functions) ⭐ NEW
- **ChatController**: WebSocket text chat (11 functions) ⭐ NEW
- **SignalingController**: WebRTC P2P signaling (8 functions)
- **ContentController**: Content management (1 function)
- **HomeController**: Home screen data management (2 functions)
- **NotiController**: Notification and announcement management (9 functions)
- **ProfileController**: Profile management (1 function)
- **ItemController**: Item management (1 function)
- **HistoryController**: History tracking (1 function)
- **CheckInController**: Check-in functionality
- **CsCenterController**: Customer service center
- **InboxController**: Inbox management
- **MarriageController**: Marriage feature
- **SetController**: User settings
- **ShopController**: Shop functionality
- **VideoChatController**: Video chat features
- **VoiceChatController**: Voice chat features

### Support Modules
- **types.go**: Utility functions and constants (3 functions + constants)
- **ctl_test.go**: Comprehensive testing suite (15+ functions including story upload tests)
- **imgCtl**: Image processing and content moderation (4 functions)

### Key Features by Category
- **Authentication**: JWT tokens, login/logout, password management
- **User Management**: Registration, profile updates, account operations
- **Real-time Communication**: WebRTC signaling, WebSocket chat, call status
- **Story Sharing**: Image upload to Cloudflare, story management, comments ⭐ NEW
- **Text Chat**: WebSocket messaging, typing indicators, read receipts ⭐ NEW
- **Content Moderation**: NSFW detection, image processing
- **Notification System**: User notifications, announcements, read status management
- **Home Screen**: Integrated data delivery for video chat, voice chat, stories
- **Testing**: Comprehensive API testing, file uploads, endpoint validation
- **Security**: Encrypted data handling, token validation, STUN server testing

---

## Production Features
- **Error Handling**: Structured error responses with appropriate HTTP status codes
- **Logging**: Differentiated logging levels (error/warn) based on response severity
- **Pagination**: Configurable pagination for list endpoints
- **File Validation**: Image type and size validation for uploads
- **Session Management**: Redis-based session storage with heartbeat mechanisms
- **Monitoring**: WebRTC statistics and health check endpoints

---

## signaling.go - WebRTC Signaling Controller

### Signaling Controller
- `NewSignalingController()`: Creates SignalingController instance for WebRTC P2P signaling

### WebSocket Connection Management
- `HandleWebSocket()`: Handles WebSocket connection requests and upgrades to WebSocket protocol
- `registerClient()`: Registers new client to active connections map
- `unregisterClient()`: Removes client from active connections and cleanup resources

### Message Processing Functions
- `readPump()`: Reads messages from WebSocket connection (goroutine)
- `writePump()`: Writes messages to WebSocket connection (goroutine)
- `handleMessage()`: Processes received signaling messages by type
- `relayMessage()`: Relays signaling messages to target user
- `sendToClient()`: Sends message to specific client

### User List Management
- `broadcastUserList()`: Broadcasts connected user list to all clients
- `GetConnectedUsers()`: HTTP API endpoint to retrieve list of connected users

### Key Features
- **WebSocket-based Signaling**: Real-time P2P signaling server using WebSocket
- **Concurrent Connection Management**: Thread-safe multi-client connection handling
- **Message Relay**: Offer/Answer/ICE Candidate relay between peers
- **User List Synchronization**: Real-time user list updates on connect/disconnect
- **Ping/Pong Mechanism**: 54s interval ping, 60s timeout for connection health check
- **Goroutine-based**: Separate read/write goroutines per client

---
*Created: 2025-09-15*
*Last Updated: 2025-11-30*
*File Location: /home/jino/go/src/ms-gateway/controller/*

---

## Chat/Signaling 확장 (2026-03)

### 운영 정책 (Step A~D)

#### Step A: 송신자 정책
- **현재(테스트)**: `HandleWebSocket`에서 `userId` 쿼리 파라미터를 신뢰
- **서버 강제**: `handleMessage()`에서 `msg.From = client.userID`로 클라이언트 위변조 차단
- **JWT 전환 예정**: `HandleWebSocket` 상단 TODO 블록 활성화 시 토큰에서 uid 강제 추출

#### Step B: 수신 라우팅 정책
- **실시간**: 수신자가 Chat WS 연결 중이면 `sendToUser()` → 즉시 전달
- **오프라인**: `sendToUser()` 실패 시 → `IncrUnread()` 증가 + `FCMPusher.SendDMPush()` 발송
- **재접속**: `GetTotalUnread()` / `GetAllUnreadForUser()`로 unread 조회, `ResetUnread()`로 초기화

#### Step C: 룸 정책
- **DM 룸**: `ensureDMRoom()` → MySQL `dm_room` 테이블로 영구 관리
- **waitingRoom**: signaling 통화 대기 전용 (TTL 있음), DM 룸과 **절대 혼용 금지**
- chatCtl에서 `waitingRoom`/`WTRoom` 참조 0건으로 검증 완료

#### Step D: 통화 연계
- **DM 중 통화 요청** (`handleCallRequestFromDM`):
  1. 수신자가 signaling `waitingRoom`에 있으면 → `ForwardCallRequest()`로 즉시 전달
  2. Chat WS 온라인이면 → `call-incoming` DM 알림
  3. 완전 오프라인 → `FCMPusher.SendCallPush()` fallback
- **통화 취소** (`handleCallCancel`): 수신자 오프라인 시 FCM push fallback 포함
- **시그널링 취소 보강** (`SignalingController.handleCallCancel`): 수신자가 이미 통화방에 있는 경우 `partner-left` 이벤트 전달 + `returnToWaitingRoom()`으로 종료 처리

### ChatController (`chatCtl.go`)
- `ChatClient`에 `uid uint64` 필드 추가
- 연결/해제 시 Redis 온라인 상태 반영 (`SetOnline` / `DeleteOnline`)
- `handleTextMessage()`에서 DM 방 자동 확보, unread 증가, 오프라인 FCM fallback, `msg-ack` 전송
- `handleReadReceipt()`에서 unread 초기화(`ResetUnread`) 후 상대에게 전달
- 신규 통화 브릿지: `handleCallRequestFromDM`, `handleCallAcceptFromDM`, `handleCallCancel`
- 신규 공개 메서드: `IsUserOnline`, `SendDMNotification`, `GetTotalUnread`

### SignalingController (`signaling.go`)
- 대기실 메시지 타입에 `call-cancel` 추가
- 대기방 사용자 목록 프로토콜 분리: `join-waiting` 자동 목록 전송 제거, `get-user-list` 요청 시 반환
- `handleCallRequest()` 오프라인 fallback: Chat WS(`call-incoming`) -> FCM 순서 처리
- `handleCallResponse()` 수락 시 `partner` 정보 포함
- 신규 공개 메서드: `IsUserInWaitingRoom`, `ForwardCallRequest`

### FCMPusher (`fcmPusher.go`)
- `SendCallPush(callerNick, callerPic, callMode, did)`
- `SendDMPush(senderNick, content, did)`

### 클라이언트 테스트 페이지 (`client/dm_chat.html`)
- WS: `GET /chat/v01/ws?userId=` — 서버 `writePump`가 여러 JSON을 `\n`으로 묶어 보내므로, 수신 시 줄 단위로 분리 후 `JSON.parse`
- 방 준비: `POST /chat/v01/room`이 라우터에 등록되어 있으면 JWT와 함께 시도하고, 없거나 실패 시 `ensureDMRoom`과 동일한 `min(uid)_max(uid)` 규칙으로 `roomId` 계산
- REST: `GET /chat/v01/rooms`, `GET /inbox/v01/unread`는 Redis 등록 JWT 필수; 기본 서버 주소 예시는 `localhost:8080` (`conf/config.toml`의 `port`와 맞출 것)
- **쪽지함 API**: `GetChatRooms` → MySQL `dm_room` 행만 조회하므로, 상대 WS 접속만으로는 목록이 비어 있을 수 있음(첫 `text-message`로 `ensureDMRoom` 실행 후 DB 반영). 테스트 페이지는 DB 미등록 시 로컬 준비 방을 점선으로 합쳐 표시하고, `msg-ack` 후 목록을 다시 불러옴
