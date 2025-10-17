# Protocol Package Function Feature Summary

## userptl.go - User Protocol Structures

### Request Structures
- `RegistReq struct`: User registration request structure containing ID, password, user info, personal details, and profile settings
- `LoginReq struct`: User login request structure containing ID and password for authentication

### Response Structures
- `UserInfoResp struct`: User information response structure containing personal details, profile info, and join date

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

### types.go (7 structures + 2 functions + 1 data map)
- **Response Management**: Header creation functions (2 functions)
- **Core Responses**: Standard response formats (3 structures)
- **Utility Types**: Pagination, encryption, custom writer (4 structures)
- **Configuration**: Language support mapping (1 data map)

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
- **System Response**: Standardized success/error response handling
- **Security**: Encrypted data transmission and token validation
- **Pagination**: List response management with page controls
- **Internationalization**: Multi-language response support

---
*Created: 2025-09-15*
*File Location: /home/jino/go/src/ms-gateway/protocol/*
