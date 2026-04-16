# Protocol Package Function Feature Summary

## userptl.go - User Protocol Structures

### Request Structures
- `RegistReq struct`: User registration request structure containing ID, password, user info, personal details, and profile settings
- `LoginReq struct`: User login request structure containing ID and password for authentication
- `RefreshTokenReq struct`: Refresh token reissue request structure with `refTok` field (body fallback when Authorization header is absent)

### Response Structures
- `UserInfoResp struct`: User information response structure containing personal details, profile info, and join date
- `RefreshTokenResp struct`: Token refresh response structure containing renewed access/refresh token pair and uid

---

## history.go - History and Notification Protocol Structures

### Notification Structures
- `Noti struct`: User notification structure containing index, notification type, title, message, timestamp, status, and sender information (uid, url, nickname)

### Announcement Structures
- `Announcement struct`: Public announcement structure containing index, title, body content, URL link, timestamp, and status

### Database Schema Comments
- `noti_his` table: Personal user notification history with 9 fields (idx, uid, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick)
- `anuc_his` table: Public announcement history with 5 fields (idx, an_title, an_body, an_url, at_msg)

---

## home.go - Home Screen Protocol Structures

### Request Structures
- `HomeDataReq struct`: Home screen data request structure containing user ID

### Video Chat Structures
- `VideoChat struct`: Video chat user information containing ID, VCID, profile picture, intro, gender, nickname, area, and age

### Voice Chat Structures
- `VoiceChat struct`: Voice chat user information containing ID, VOID, profile picture, intro, gender, nickname, area, and age

### Story Structures
- `Story struct`: Story information containing user ID, story ID, and story picture URL

### Response Structures
- `HomeDataResp struct`: Comprehensive home screen response containing notification status, message count, check-in status, video chat list, voice chat list, story list, terms URL, and policy URL

---

## types.go - Common Protocol Types and Functions

### Response Management Functions
- `NewRespHeader()`: Constructor function that creates standardized response header with result code, result string, and description
- `NewRespDataHeader()`: Constructor function that creates data response header with result code, result string, and data payload

### Core Response Structures
- `RespHeader struct`: Standard response header structure containing result code, result string, and description for all API responses
- `OkResp struct`: Simple success response structure that embeds RespHeader for basic acknowledgment responses
- `RespDataHeader struct`: Data response structure that includes result information and actual data payload for API responses

### Utility Structures
- `Pagination struct`: Pagination information structure containing page number, limit, and total count for list responses
- `DefaultPaginationQuery struct`: Default pagination query parameters structure with form binding for page and limit values
- `AesDataForm struct`: AES encrypted data wrapper structure containing encrypted string data for secure transmission
- `BodyWriter struct`: Custom response writer structure that extends gin.ResponseWriter with buffer for response manipulation

### Configuration Data
- `LangCode map`: Language code validation map supporting multiple languages (ko, en, cn, tss, es, ja) for internationalization

---

## result_code.go - Result Code Management

### Result Code Definition
- `ResultCode type`: Custom integer type for standardized API result codes and error handling

### Result Code Translation Function
- `toString()`: Method that converts ResultCode integer values to human-readable string representations

### Result Code Constants (25 constants)
#### General Status Codes
- `Success (0)`: Operation completed successfully
- `Failed (1)`: General request failure
- `IpInvalid (2)`: Invalid IP address in request

#### User Management Error Codes
- `UserIDNotFound (13)`: User ID does not exist in system
- `IDDuplicate (14)`: User ID already exists (duplicate)
- `EmailDuplicate (15)`: Email address already exists (duplicate)

#### Authentication & Authorization Codes
- `AccessTokenInvalid (101)`: Invalid or expired access token
- `InvalidParam (102)`: Invalid request parameters provided

#### User Registration Codes
- `UserAlreadyExists (103)`: User already exists in system
- `UserRegistFailed (104)`: User registration process failed
- `UserRegistSuccess (105)`: User registration completed successfully

#### System Processing Codes
- `JsonParseFailed (106)`: JSON parsing or format error

#### User Authentication Codes
- `UserLoginFailed (107)`: User login process failed
- `UserLogoutFailed (108)`: User logout process failed

#### Password Management Codes
- `UserChangePWFailed (109)`: Password change operation failed
- `UserChangePWSuccess (110)`: Password change completed successfully

#### Account Recovery Codes
- `UserFindIDFailed (111)`: User ID recovery process failed
- `UserFindPWFailed (112)`: Password recovery process failed

#### Account Management Codes
- `UserLeaveFailed (113)`: User account withdrawal failed
- `UserInfoFailed (114)`: User information retrieval failed
- `UserDeleteFailed (115)`: User account deletion failed

---

## Overall Structure Summary

### userptl.go (3 structures)
- **Request Structures**: Registration and login request formats (2 structures)
- **Response Structures**: User information response format (1 structure)

### history.go (2 structures)
- **Notification Structures**: Personal user notifications (1 structure)
- **Announcement Structures**: Public announcements (1 structure)
- **Story Comment Structures**: `StrComment` 확장 필드 포함 (`thumb_url`, `wgender`, `wage(string)`, `warea`)

### home.go (5 structures)
- **Request Structures**: Home screen data request (1 structure)
- **Response Structures**: Comprehensive home screen data (1 structure)
- **Feature Structures**: Video chat, voice chat, story (3 structures)

### types.go (7 structures + 2 functions + 1 data map)
- **Response Management**: Header creation functions (2 functions)
- **Core Responses**: Standard response formats (3 structures)
- **Utility Types**: Pagination, encryption, custom writer (4 structures)
- **Configuration**: Language support mapping (1 data map)

### DM/통화 확장 구조체 (2026-03)
- `PartnerInfo`: 통화/DM 상대 프로필 정보(UID, Nick, MainPic, ThumbPic, SPIntro, Gender, Age, Area)
- `DMRoomResp`: 쪽지함 방 응답 포맷(roomId, partner, unread, atCreate, atUpdate)
- `ChatMessage` 확장 필드:
  - `callMode`: `video|audio|text`
  - `msgId`: 클라이언트 메시지 식별자
  - `partner`: 통화 수락 시 상대방 정보 포함
  - `unread`: 서버 ack 시 unread 수 전달

### result_code.go (1 type + 1 function + 25 constants)
- **Type Definition**: Custom result code type (1 type)
- **Translation**: Code to string conversion (1 function)
- **Status Codes**: Comprehensive error and success codes (25 constants)

---

## Key Features
- **Standardized Responses**: Consistent API response structure across all endpoints
- **Comprehensive Error Handling**: Detailed result codes for different failure scenarios
- **Security Support**: AES encryption wrapper for secure data transmission
- **Internationalization**: Multi-language support with language code validation
- **Flexible Pagination**: Configurable pagination for list-based API responses
- **Type Safety**: Strong typing for all protocol structures and result codes

---

## Protocol Categories
- **User Management**: Registration, login, profile, account operations
- **Notification System**: Personal notifications and public announcements
- **Home Screen**: Video chat, voice chat, story data integration
- **System Response**: Standardized success/error response handling
- **Security**: Encrypted data transmission and token validation
- **Pagination**: List response management with page controls
- **Internationalization**: Multi-language response support

---
*Created: 2025-09-15*
*Last Updated: 2025-11-05*
*File Location: /home/jino/go/src/ms-gateway/protocol/*
