# Controller Package Function Feature Summary

## controller.go - Main Controller Core

### Controller Initialization
- `NewCTL()`: Creates main controller instance, initializes sub-controllers (Account, Profile), and sets up repository connections

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

## Overall Structure Summary

### Main Controllers (10+ controllers)
- **Controller**: Core controller management (9 functions)
- **AccountController**: User account operations (18 functions)
- **ContentController**: Content management (1 function)
- **HomeController**: Home screen data management (2 functions)
- **NotiController**: Notification and announcement management (9 functions)
- **ProfileController**: Profile management (1 function)
- **ItemController**: Item management (1 function)
- **HistoryController**: History tracking (1 function)
- **CheckInController**: Check-in functionality (placeholder)
- **CsCenterController**: Customer service center (placeholder)
- **InboxController**: Inbox management (placeholder)
- **MarriageController**: Marriage feature (placeholder)
- **SettingController**: User settings (placeholder)
- **ShopController**: Shop functionality (placeholder)
- **StoryListController**: Story list management (placeholder)
- **VideoChatController**: Video chat features (placeholder)
- **VoiceChatController**: Voice chat features (placeholder)

### Support Modules
- **types.go**: Utility functions and constants (3 functions + constants)
- **ctl_test.go**: Comprehensive testing suite (12+ functions)
- **imgCtl**: Image processing and content moderation (4 functions)

### Key Features by Category
- **Authentication**: JWT tokens, login/logout, password management
- **User Management**: Registration, profile updates, account operations
- **Real-time Communication**: WebRTC configuration, chat rooms, call status
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
