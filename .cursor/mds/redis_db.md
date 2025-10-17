# Redis DB Function Feature Summary

## Database Connection and Management Functions
- `NewRedisDB()`: Constructor function that establishes Redis client connection and initializes RedisDB instance
- `Start()`: Function to start Redis service (currently no implementation)
- `Close()`: Function to close Redis client connection
- `Ping()`: Function to check Redis connection status
- `Terminate()`: Function to log Redis database termination process

## Basic Cache Management Functions
- `SetCache()`: Function to store key-value pairs in Redis with 30-minute TTL
- `SetCacheMintime()`: Function to store key-value pairs with specified TTL in minutes
- `SetBytes()`: Function to store byte data in Redis (currently no implementation - commented out)
- `GetCache()`: Function to retrieve value by key from Redis
- `DeleteCache()`: Function to delete key from Redis
- `IncCount()`: Function to increment key value by 1 and set 30-minute TTL in Redis
- `SetNXCache()`: Function to set value only if key does not exist

## Legacy Hash Management Functions (Commented Out)
- `HSetMember()`: Function to store key-value pairs in "member" hash (DEPRECATED - commented out)
- `HGetMember()`: Function to retrieve value by key from "member" hash (DEPRECATED - commented out)
- `HSetAccess()`: Function to store JWT session information in "access" hash (DEPRECATED - commented out)
- `HGetAccess()`: Function to retrieve JWT session information from "access" hash (DEPRECATED - commented out)
- `HGetAllAddress()`: Function to retrieve all keys from "gember" hash (DEPRECATED - commented out)
- `HDelMember()`: Function to delete key from "gember" hash (DEPRECATED - commented out)

## Active JWT Token Management Functions
- `HSetJWTAccess()`: Function to store JWT token and user ID in AUTH:ACCESS hash with 24-hour TTL
- `HGetJWTAccess()`: Function to retrieve user ID by JWT token from AUTH:ACCESS hash
- `DeleteJWTToken()`: Function to delete JWT token and remove from user's active token list
- `DeleteAllUserTokens()`: Function to delete all active tokens for a specific user (uses old jwt:sessions format)

## User Information Management Functions
- `HSetUserInfo()`: Function to store encrypted user information in USER:INFO hash using ChaCha20 encryption
- `GetUserInfo()`: Function to retrieve and decrypt user information by user ID
- `GetUser()`: Function to retrieve basic user information from user:info key
- `SaveUser()`: Function to save user information (DEPRECATED - commented out)
- `GetUserActiveTokens()`: Function to retrieve user's active token list (DEPRECATED - commented out)
- `CleanupExpiredTokens()`: Scheduler function to clean up expired JWT tokens (DEPRECATED - commented out)

## Chat Room Management Functions
- `SetChatRoom()`: Function to create new chat room and store information in Redis
- `GetChatRoom()`: Function to retrieve chat room information by room ID
- `GetChatRooms()`: Function to retrieve all active chat rooms ordered by creation time
- `ActivateChatRoom()`: Function to change chat room to active status
- `DeactivateChatRoom()`: Function to change chat room to inactive status
- `GetUserChatRoom()`: Function to retrieve user's chat room ID
- `DeleteChatRoom()`: Function for room owner only to delete chat room and all related data
- `ListActiveChatRooms()`: Function to retrieve active chat rooms ordered by recent activity

## Chat Message Management Functions
- `SaveChatMessage()`: Function to save chat message and update chat room activity time
- `GetChatMessages()`: Function to retrieve messages from specific chat room with pagination

## Chat Room Participant Management Functions
- `AddUserToChatRoom()`: Function to add user to chat room participant list
- `RemoveUserFromChatRoom()`: Function to remove user from chat room participant list

## User Connection Status Management Functions
- `UpdateUserConnectionStatus()`: Function to update user's connection status
- `IsUserConnected()`: Function to check user's connection status

## Utility Functions
- `ScanKeys()`: Function to search for keys matching pattern using Redis SCAN command

## User Information Management Functions
- `GetUser()`: Function to retrieve user information
- `SaveUser()`: Function to save user information
- `GetUserActiveTokens()`: Function to retrieve user's active token list

## Chat Room List Management Functions
- `AddUserToChatRoomList()`: Function to add room to user's chat room list
- `MoveRoomToTop()`: Function to move chat room to top of list based on event type
- `GetUserChatRoomList()`: Function to retrieve user's chat room list with pagination in latest order
- `GetUserChatRoomListWithScores()`: Debugging function to retrieve chat room list with scores
- `RemoveUserFromChatRoomList()`: Function to remove room from user's chat room list
- `UpdateRoomUnreadCount()`: Function to update unread message count for chat room
- `GetChatRoomListCount()`: Function to retrieve total count of user's chat room list
- `GetUnreadTotalCount()`: Function to retrieve user's total unread message count
- `CleanupInactiveRooms()`: Function to clean up chat rooms inactive for certain period

## WebRTC Management Functions
- `GetWebRTCConfig()`: Function to return WebRTC configuration information with hardcoded default values
- `GetWebRTCConfigFromConf()`: Function to create WebRTC configuration referencing config file (DEPRECATED - commented out)
- `getTurnCredential()`: Internal function to generate TURN server credentials (currently empty implementation)
- `SetUserWebRTCSession()`: Function to save user WebRTC session information and update active user list
- `GetUserWebRTCSession()`: Function to retrieve user WebRTC session information
- `GetAvailableUsersForCall()`: Function to retrieve list of users available for calls (excluding self)
- `UpdateUserCallStatus()`: Function to update user's call status
- `CleanupInactiveSessions()`: Function to clean up WebRTC sessions inactive for more than 30 minutes
- `GetWebRTCStats()`: Function to retrieve WebRTC-related statistics (session count, active users, ongoing calls)

---

## Overall Structure Summary

### 📊 Function Statistics by Category
- **Database Connection and Management**: 5 functions
- **Basic Cache Management**: 7 functions
- **Legacy Hash Management**: 6 functions (DEPRECATED - commented out)
- **Active JWT Token Management**: 4 functions
- **User Information Management**: 6 functions (3 deprecated)
- **Chat Room Management**: 9 functions
- **Chat Message Management**: 2 functions
- **Chat Room Participant Management**: 2 functions
- **User Connection Status Management**: 2 functions
- **Utility**: 1 function
- **Chat Room List Management**: 9 functions
- **WebRTC Management**: 9 functions (1 deprecated)

### 🎯 Key Features
- **Real-time Chat**: Complete support for chat rooms, messages, and participant management
- **JWT Authentication**: Token-based user authentication and session management with ChaCha20 encryption
- **WebRTC Communication**: WebRTC configuration and session management for video calls
- **Cache Optimization**: Efficient data management with various TTL settings and hash-based operations
- **Real-time Notifications**: Chat room priority and unread message management
- **Data Security**: ChaCha20 encryption for sensitive user information storage

### 📈 Code Changes Summary
- **Deprecated Functions**: Legacy JWT and hash management functions commented out for cleanup
- **Enhanced Security**: Added ChaCha20 encryption for user information storage
- **Simplified JWT Management**: Streamlined JWT token handling with AUTH:ACCESS hash
- **Code Cleanup**: Removed duplicate structures and unused functions for better maintainability

---
*Created: 2025-09-15*
*Updated: 2025-09-15*
*File Location: /home/jino/go/src/ms-gateway/models/redis_db.go*
*Total Active Functions: 52 (10 deprecated/commented)*
*Total Lines: ~1,500*
