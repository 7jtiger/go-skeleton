# Account Controller Function Summary

## Structure and Initialization Functions
- `NewAccountController()`: Constructor function that initializes AccountController instance and sets up required dependencies

## Basic Service Functions
- `GetVersion()`: API endpoint function that returns service version information (0.9.1)

## User Account Validation Functions
- `CheckID()`: Function that checks if user ID is duplicated and validates availability
- `CheckEmail()`: Function that checks if user email is duplicated and validates availability

## User Authentication Functions
- `RegistUserInfo()`: Function that registers new users and sets up default profile information
- `LoginUser()`: Function that handles user login, generates JWT tokens, stores session in Redis, initializes WebRTC session, and returns WebRTC configuration
- `genLoginUserToken()`: Internal function that generates JWT access/refresh tokens and stores them in Redis with user information
- `RefreshToken()`: Function that reissues access/refresh tokens using refresh token from `Authorization: Bearer ...` header or JSON body
- `extractRefreshToken()`: Helper that safely reads refresh token without treating empty request body as JSON parse failure (`EOF`)
- `LogoutUser()`: Function that handles user logout, deletes JWT token from Redis, and updates last access time

## User Account Management Functions
- `LeaveUser()`: Function that handles user withdrawal
- `DeleteUser()`: Function that deletes user accounts but actually changes status to inactive (status=4)

## Account Recovery and Change Functions
- `FindID()`: Function that finds user ID using name and birth date
- `FindPW()`: Function that handles password recovery using ID, email, and birth date
- `ChangePW()`: Function that verifies existing password and changes to new password

## User Information Management Functions
- `ModifyUserInfo()`: Function that modifies user information (pw, area, nick, email, etc.)
- `GetUserInfo()`: Function that retrieves user information by user ID
- `ModifyMainPic()`: Function that uploads and modifies user main profile image

## WebRTC Related Functions
- `GetWebRTCConfig()`: Function that retrieves WebRTC configuration information for authenticated users
- `GetAvailableUsers()`: Function that retrieves list of users available for video calls
- `UpdateCallStatus()`: Function that updates user call status (in call/waiting)
- `UpdateWebRTCHeartbeat()`: Function that updates WebRTC session heartbeat to maintain session
- `GetWebRTCStats()`: Function that retrieves WebRTC system statistics information

## STUN Server Test Functions
- `TestStunServers()`: Function that tests connectivity of default STUN servers and checks public IP
- `TestSpecificStunServer()`: Function that tests connectivity of specific STUN server

---

*Last Updated: 2025-11-30*

