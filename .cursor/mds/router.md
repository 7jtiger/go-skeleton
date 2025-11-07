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
- `ValidateFileUpload()`: File upload validation middleware that verifies uploaded file count, size, and type
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
- `Idx()`: Main router function that initializes Gin engine and sets up all routes

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

### story/v01 - Story Feature (Lightweight Authentication Required)
- `GET /story/v01/home/:id`: Endpoint to retrieve user's story home
- `GET /story/v01/list/:stat/:id`: Endpoint to retrieve story list by status
- `GET /story/v01/detail/:idx`: Endpoint to retrieve specific story details

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

### chat/v01 - Chat Features (Security Headers)
- `GET /chat/v01/mlistvd`: Endpoint to retrieve male video chat list
- `GET /chat/v01/wlistvd`: Endpoint to retrieve female video chat list
- `GET /chat/v01/mlistvo`: Endpoint to retrieve male voice chat list
- `GET /chat/v01/wlistvo`: Endpoint to retrieve female voice chat list

### inbox/v01 - Inbox Management (Security Headers)
- `GET /inbox/v01/list`: Endpoint to retrieve inbox message list

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

### WebRTC Signaling WebSocket
- **URL**: `ws://server:port/webrtc/v01/ws?userId={userId}`
- **Protocol**: WebSocket
- **Authentication**: No Auth (개발용, 프로덕션에서는 JWT 권장)
- **Purpose**: Real-time P2P video chat signaling

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

---
*Last Updated: 2025-11-05*

