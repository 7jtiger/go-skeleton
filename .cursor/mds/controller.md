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
- `LoginUser()`: Authenticates user login, creates JWT token, stores session in Redis, and initializes WebRTC session
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
- `GetWebRTCConfig()`: Retrieves WebRTC configuration for authenticated user from Redis
- `GetAvailableUsers()`: Returns list of users available for video calls based on current status
- `UpdateCallStatus()`: Updates user's call status (in call or available) with optional call partner info
- `UpdateWebRTCHeartbeat()`: Updates WebRTC session heartbeat to maintain active session status
- `GetWebRTCStats()`: Retrieves WebRTC system statistics for monitoring and admin purposes

### STUN Server Testing Functions
- `TestStunServers()`: Tests connectivity to recommended STUN servers and returns connection report
- `TestSpecificStunServer()`: Tests connectivity to specific STUN server URL and returns detailed results

---

## content.go - Content and Chat Room Management

### Content Controller
- `NewContentController()`: Creates ContentController instance for content management operations
- `GetTestContent()`: Simple test endpoint that returns success message for content controller validation

### Chat Room List Controller
- `NewChatRoomListController()`: Creates ChatRoomListController with Redis connection for real-time chat management
- `GetChatRoomList()`: Retrieves user's chat room list with pagination, sorted by activity and priority
- `JoinChatRoom()`: Adds user to chat room and updates user's chat room list
- `LeaveChatRoom()`: Removes user from chat room and updates user's chat room list
- `HandleChatRoomEvent()`: Processes chat room events (message, mention, urgent, read) and moves rooms to top
- `MarkAsRead()`: Marks chat room as read by setting unread count to 0 and updating priority
- `GetChatRoomStats()`: Retrieves chat room statistics including total rooms and unread message counts

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

### Main Controllers (8 controllers)
- **Controller**: Core controller management (9 functions)
- **AccountController**: User account operations (18 functions)
- **ContentController**: Content management (1 function)
- **ChatRoomListController**: Chat room operations (6 functions)
- **ProfileController**: Profile management (1 function)
- **ItemController**: Item management (1 function)
- **HistoryController**: History tracking (1 function)

### Support Modules
- **types.go**: Utility functions and constants (3 functions + constants)
- **ctl_test.go**: Comprehensive testing suite (12+ functions)
- **imgCtl**: Image processing and content moderation (4 functions)

### Key Features by Category
- **Authentication**: JWT tokens, login/logout, password management
- **User Management**: Registration, profile updates, account operations
- **Real-time Communication**: WebRTC configuration, chat rooms, call status
- **Content Moderation**: NSFW detection, image processing
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
*Created: 2025-09-15*
*File Location: /home/jino/go/src/ms-gateway/controller/*
