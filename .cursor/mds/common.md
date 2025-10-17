# Common Package Function Feature Summary

## common.go - Core Common Functions

### Data Type Conversion Functions
- `Hex2Int64()`: Converts hexadecimal string to unsigned 64-bit integer by removing "0x" prefix and parsing as base 16
- `GetAto64()`: Converts string to 64-bit integer with error handling, returning 0 on parse failure
- `GetFilterCount()`: Counts the number of parts when splitting a string by a specific filter delimiter
- `GetMiddlePath()`: Determines file path prefix based on content type (returns "noti/" for admnoti or "faq/" for admfaq)

### File Operations Functions
- `ReadFileLastNum()`: Reads file and returns 1 if successful, 0 if file read fails (simple file existence check)
- `WriteFileLastNum()`: Always returns true regardless of file read result (placeholder implementation)

### Utility Methods
- `StTest.Expose()`: Prints "inner expose" message for testing purposes

---

## logger/ - Logging System

### logger.go - Comprehensive Logging Functions

#### Logging Initialization
- `InitLogger()`: Initializes zap logger with file rotation, console output, and configurable log levels based on environment mode

#### Core Logging Functions
- `Debug()`: Logs debug-level messages by concatenating multiple interface{} parameters into a single string
- `Info()`: Logs info-level messages by concatenating multiple interface{} parameters into a single string
- `Warn()`: Logs warning-level messages by concatenating multiple interface{} parameters into a single string
- `Error()`: Logs error-level messages by concatenating multiple interface{} parameters into a single string
- `Errorf()`: Logs formatted error messages using printf-style formatting with arguments
- `Crit()`: Logs critical messages and triggers panic for severe errors

#### Gin Framework Integration
- `GinLogger()`: Creates Gin middleware for HTTP request logging with method, path, status, IP, user-agent, and response time
- `GinRecovery()`: Creates Gin middleware for panic recovery with optional stack trace logging and broken connection handling

---

## maps/ - Thread-Safe Map Collections

### genMap.go - Generic Map Operations

#### Map Management Functions
- `NewGenMap()`: Creates new thread-safe GenMap instance with mutex protection and 10,000 initial capacity
- `cleanPool()`: Clears existing map data and reinitializes with fresh GenMap instance
- `getLen()`: Calculates remaining capacity to next 1000-unit boundary for buffer management
- `GetSize()`: Returns current number of items in the map with nil safety check
- `Insert()`: Thread-safely inserts blockchain data using blockSymbol and blockNumber as composite key
- `TxInsert()`: Thread-safely inserts transaction data using hash as the key
- `GetDump()`: Thread-safely retrieves all map data, clears original map, and returns copy with count

### aggMap.go - Aggregation Map Operations

#### Map Management Functions
- `NewMap()`: Creates new thread-safe UMap instance with mutex protection and 200 initial capacity
- `cleanMap()`: Clears existing map data and reinitializes with fresh UMap instance
- `getLen()`: Calculates remaining capacity to next 1000-unit boundary for buffer management
- `MapSize()`: Returns current number of items in the map with nil safety check
- `Join()`: Thread-safely joins blockchain data using blockSymbol and blockNumber as composite key
- `InJoin()`: Thread-safely joins transaction data using hash as the key
- `MapSwap()`: Thread-safely swaps out all map data, clears original map, and returns copy with count

---

## types/ - Common Type Definitions

### type.go - Data Structure Definitions

#### Core Data Structures
- `Elem struct`: Element structure containing name, description, URL, status, and reserve fields for general usage
- `Notice struct`: Notice/announcement structure with index, view count, category, writer, timestamps, and body content
- `Body struct`: Content body structure containing title, creation date, content, and location information
- `RespHeader struct`: Response header structure containing result code, result string, and description for API responses

#### Result Code System
- `ResultCode type`: Custom integer type for standardized result codes
- **Constants**: Success (0), Failed (1), UserIDNotFound (13), AccessTokenInvalid (101), UserNotFound (102)

---

## utils/ - Utility Functions Collection

### converter.go - Data Conversion Utilities

#### JSON and Map Processing
- `GetValue()`: Extracts field value from map[string]interface{} by marshaling to JSON and unmarshaling back
- `GetJsonStr2Value()`: Extracts field value from JSON string by unmarshaling to map and accessing field
- `GetFieldValue()`: Extracts field value from string by finding field position and parsing until comma delimiter
- `GetFieldValueCount()`: Splits string by filter and returns element at specified count index
- `GetFilterCount()`: Counts number of parts when splitting string by filter delimiter
- `GetJsonValue()`: Extracts JSON field value by finding field position and parsing until quote delimiter
- `GetAto64()`: Converts string to 64-bit integer with error handling, returning 0 on failure

#### Utility Methods
- `StTest.Expose()`: Prints "inner expose" message for testing
- `Emt.GetIfValue()`: Returns empty string (placeholder implementation for interface value extraction)

### crypto.go - Cryptographic Functions

#### Random Generation
- `GenerateRandomBytes()`: Generates cryptographically secure random bytes of specified length using crypto/rand

#### AES-256 GCM Encryption
- `EncryptGCM()`: Encrypts data using AES-256 GCM mode with random nonce and returns base64-encoded ciphertext
- `DecryptGCM()`: Decrypts AES-256 GCM encrypted data by extracting nonce and verifying authentication tag

#### ChaCha20 Encryption
- `EncryptChaCha20()`: Encrypts data using ChaCha20 stream cipher with random nonce and base64 encoding
- `DecryptChaCha20()`: Decrypts ChaCha20 encrypted data by extracting nonce and performing stream decryption

### http.go - HTTP Client Functions

#### Form and JSON Requests
- `PostForm()`: Sends HTTP POST request with form data and unmarshals JSON response
- `Post()`: Sends HTTP POST request with JSON body, authorization header, and unmarshals response
- `PostWithHeader()`: Sends HTTP POST request with custom headers and returns response as map
- `PostHeaderStr()`: Sends HTTP POST request with string body and custom headers

#### GET Requests
- `Get()`: Sends HTTP GET request with query parameters and returns response body as string
- `GetWithHeader()`: Sends HTTP GET request with query parameters and custom headers

#### Testing and Alerts
- `GetCtxTest()`: Performs test GET request to specific endpoint with date parameters
- `PostTest()`: Performs test POST request with JSON body and logs results
- `SendChatAlert()`: Sends alert messages to Google Chat webhook based on environment mode
- `SendTelegramAlert()`: Sends formatted alert messages to Telegram bot with environment context

#### Utility Functions
- `GenUuid()`: Generates UUID v4 and returns as hexadecimal string
- `GetLocalIP()`: Returns first local network interface IP address

### HttpClientPool.go - Job Pool System

#### Pool Management
- `New()`: Creates job pool with specified number of worker routines and queue capacity
- `Shutdown()`: Gracefully shuts down job pool by stopping queue and worker routines
- `JQueue()`: Queues job for execution with priority option and waits for completion
- `QueuedJobs()`: Returns current number of queued jobs using atomic operations
- `ActRoutines()`: Returns current number of active worker routines using atomic operations

#### Queue Operations
- `Enqueue()`: Adds job to priority or normal queue based on priority flag
- `Dequeue()`: Removes job from queue (priority first) and returns to worker
- `GetDump()`: Retrieves job from dequeue channel for worker execution

#### Worker Management
- `queueRoutine()`: Main queue management goroutine handling enqueue/dequeue operations
- `jobRoutine()`: Worker goroutine that processes jobs safely with panic recovery
- `JobSafety()`: Executes job with panic recovery and routine counter management
- `catchPanic()`: Panic recovery function that captures stack trace and logs errors

### jwt.go - JWT Token Management

#### Token Creation and Verification
- `CreateJWTToken()`: Creates JWT token with user ID, expiration, and standard claims (issuer, audience, etc.)
- `VerifyJWTToken()`: Verifies JWT token signature and expiration, returns claims on success
- `VerifyJWTTokenWithAudience()`: Verifies JWT token with additional audience validation for enhanced security

### timeutil.go - Time Conversion Utilities

#### Unix Time Conversion
- `UnixToTime()`: Converts Unix timestamp (int64) to time.Time object
- `UnixToTimeStamp()`: Converts Unix timestamp (uint64) to formatted date-time string "YYYY-MM-DD HH:MM:SS"
- `StrUnixToTime()`: Converts Unix timestamp string to time.Time object with error handling

#### Date String Processing
- `StrToInt()`: Converts string to integer using strconv.Atoi
- `GetEndTime()`: Parses date string and returns Unix timestamp as big.Int, handles future dates
- `GetDurationTime()`: Parses date string and returns start/end time range for day calculations
- `ConvertStrToTime()`: Converts date string to time.Time with future date validation
- `StrDayToTime()`: Converts "YYYY-MM-DD" date string to time.Time at midnight
- `StrMonthToTime()`: Converts "YYYY-MM" month string to time.Time at beginning of month

### turn.go - TURN Server Management

#### TURN Credential Management
- `GenerateTurnCredentials()`: Generates time-windowed TURN credentials using HMAC-SHA1 and shared secret
- `ValidateTurnCredentials()`: Validates TURN credentials by checking timestamp expiry and HMAC verification

#### TURN Server Operations
- `CreateOptimizedTurnServer()`: Creates production-ready TURN server with performance and security optimizations
- `MonitorTurnServer()`: Monitors TURN server statistics and calls callback function with metrics
- `LogTurnServerStats()`: Logs TURN server statistics including allocations, uptime, and timestamps
- `GetTurnServerHealth()`: Returns TURN server health status and allocation count as map
- `OptimizeTurnServerForProduction()`: Applies production-optimized settings for timeouts, security, and rate limiting
- `CreateTurnAuthHandler()`: Creates authentication handler for different auth types (long-term, static, default)

### utils.go - General Utility Functions

#### System Information
- `HomeDir()`: Returns user home directory from environment or system user info
- `WorkingDir()`: Returns current working directory with macOS temp folder handling
- `Trace()`: Returns current file path, function name, and line number for debugging
- `Trace3()`: Returns caller information from 3 levels up the call stack

#### Communication Functions
- `SendMail()`: Sends email via SMTP using simple authentication and plain text body
- `SendGoMail()`: Sends HTML email via Gmail SMTP with gomail library and app password authentication
- `UploadFtp()`: Uploads file to FTP server with retry mechanism and connection management
- `AlertSlack()`: Sends alert message to Slack channel using webhook client

#### System Monitoring
- `MemUsage()`: Returns memory usage statistics (alloc, total, sys, heap, GC count) in MB
- `MemUsageString()`: Returns formatted string with current memory usage statistics

#### File Operations
- `Mkdirp()`: Creates directory structure recursively with 0777 permissions if it doesn't exist

#### Name Generation
- `GenDefNick()`: Generates random Korean nickname by combining adjectives and nouns from predefined arrays

### webrtc.go - WebRTC Connectivity Functions

#### STUN Server Testing
- `TestStunServer()`: Tests connectivity to single STUN server and returns public IP/port mapping
- `TestMultipleStunServers()`: Tests multiple STUN servers concurrently and returns connectivity report
- `parseStunURL()`: Parses STUN URL format (stun:host:port) and extracts host and port components
- `createStunBindingRequest()`: Creates STUN Binding Request packet with transaction ID and magic cookie
- `parseStunResponse()`: Parses STUN response packet and extracts public IP/port from mapped address attributes

#### WebRTC Configuration
- `GetRecommendedStunServers()`: Returns list of recommended Google STUN servers for WebRTC
- `ValidateWebRTCConfig()`: Validates WebRTC configuration format and ensures ICE servers are properly configured
- `CreateStunTestReport()`: Creates comprehensive STUN connectivity test report with success rates and details

---

## Overall Structure Summary

### Core Modules (12 files, ~80 functions)
- **common.go**: Basic utilities (7 functions)
- **logger/**: Comprehensive logging (8 functions)
- **maps/**: Thread-safe collections (14 functions)
- **types/**: Data structures (4 structures + constants)
- **utils/**: Extensive utility collection (50+ functions)

### Key Features by Category
- **Security**: AES-256 GCM, ChaCha20, JWT, TURN authentication
- **Networking**: HTTP clients, STUN testing, WebRTC validation
- **Concurrency**: Thread-safe maps, job pools, worker routines
- **Communication**: Email, Slack, Telegram, FTP
- **Time Management**: Unix conversion, date parsing, duration calculation
- **System Monitoring**: Memory usage, logging, health checks

---

## Production Readiness Features
- **Error Handling**: Comprehensive error wrapping and validation
- **Performance**: Connection pooling, concurrent processing, optimized algorithms
- **Security**: Multiple encryption methods, credential validation, rate limiting
- **Monitoring**: Detailed logging, statistics collection, health checks
- **Scalability**: Job pools, async processing, resource management

---
*Created: 2025-09-15*
*File Location: /home/jino/go/src/ms-gateway/common/*
