# Models Package Function Feature Summary

## model.go - Repository Management System

### Core Interfaces and Types
- `IRepository interface`: Common interface defining Start(), Terminate(), Close(), and Ping() methods for all repository types
- `RepositoryConstructor type`: Function type for creating repository instances with config and root repositories
- `Repositories struct`: Central repository manager with thread-safe map for storing repository instances

### Repository Management Functions
- `NewModel()`: Creates and initializes all repository instances (AccountDB, HistoryDB, ItemDB, RedisDB) with connection validation
- `Register()`: Registers repository constructor by calling it and storing the instance in type-indexed map
- `Get()`: Thread-safe retrieval of repository instances by type, returns error if repository not found

### Design Pattern Features
- **Dependency Injection**: Centralized repository creation and management
- **Type Safety**: Uses reflection to ensure type-safe repository retrieval
- **Thread Safety**: RWMutex protection for concurrent access to repository map
- **Auto-Start**: Automatically starts all registered repositories on initialization

---

## account_db.go - Account Database Repository

### Database Connection and Management
- `NewAccountDB()`: Constructor that establishes MySQL account database connection with pooling configuration
- `Start()`: Initializes account database service
- `Terminate()`: Safely closes database connection and cleanup resources
- `Close()`: Closes database connection
- `Ping()`: Checks database connection health
- `heartbeat()`: Goroutine that periodically pings database every 2 minutes

### User Existence Check Functions
- `IsExistID()`: Checks if user ID already exists in database
- `IsExistEmail()`: Checks if email already exists in database

### User Authentication Functions
- `RegistUser()`: Registers new user with encrypted password, email, and name using ChaCha20
- `LoginUser()`: Verifies password and returns user information on successful login
- `LogoutUser()`: Updates last access time on user logout

### User Account Management
- `LeaveUser()`: Soft deletes user account by setting stat=4 (inactive)
- `DeleteUser()`: Soft deletes user account by changing status to 4

### Account Recovery Functions
- `FindID()`: Searches user IDs by name and birth date, returns up to 10 results
- `FindPW()`: Validates password recovery request using ID, email, and birth date
- `ChangePW()`: Changes password after verifying current password and encrypting new one

### User Information Management
- `ModifyUserInfo()`: Updates user information by category (nickname, area, email)
- `GetUserInfo()`: Retrieves user information with decryption of sensitive fields

### Database Schema
- **user_info**: Main user table with 25 fields including authentication, profile, and status information
- **pf_info**: Profile information table with follower/following lists and additional details
- **acc_info**: Bank account information table for payment processing

### Key Features
- **Security**: ChaCha20 encryption for passwords, emails, and names
- **Connection Pooling**: 300 max connections, 30 idle connections, 3-minute lifetime
- **Soft Delete**: Status-based deletion instead of hard delete
- **Health Monitoring**: Periodic connection health checks

---

## history_db.go - History Database Repository

### Database Connection and Management
- `NewHistoryDB()`: Constructor that establishes MySQL history database connection
- `Start()`: Initializes history database service
- `Terminate()`: Safely closes database connection and cleanup resources
- `Close()`: Closes database connection
- `Ping()`: Checks database connection health
- `heartbeat()`: Goroutine that periodically pings database every 2 minutes

### Notification Management Functions
- `GetNotiAllList()`: Retrieves all personal notifications for a user, ordered by index descending
- `GetNotiCount()`: Counts unread notifications (stat=0) for a specific user
- `GetNotiDetail()`: Retrieves detailed information for a specific notification by index
- `SetNotiRead()`: Marks a notification as read by setting stat to 1

### Announcement Management Functions
- `SaveAnnouncement()`: Saves new public announcement with title, body, URL, and timestamp
- `GetAnnouncementList()`: Retrieves latest 20 announcements with automatic 'new' badge (stat=1) for items within 7 days
- `GetAnnouncementDetail()`: Retrieves specific announcement detail with automatic 'new' badge calculation

### Database Schema
- **buy_his**: Purchase history table
- **by_flw**: Follower flow tracking table
- **call_his**: Call history table with duration and payment information
- **chat_his**: Chat history table with message count tracking
- **declare_his**: Report/declaration history table
- **flw_list**: Follower list table with JSON array storage
- **pay_his**: Payment history table
- **noti_his**: Personal notification history with 9 fields
- **anuc_his**: Public announcement history with 5 fields

### Key Features
- **Notification System**: Personal user notifications with read/unread status
- **Announcement System**: Public announcements with time-based 'new' badge
- **Auto-Status Calculation**: Automatic status calculation based on timestamp
- **Connection Pooling**: Same pooling configuration as AccountDB

---

## item_db.go - Item Database Repository

### Database Connection and Management
- `NewItemDB()`: Constructor that establishes MySQL item database connection
- Basic IRepository interface implementation for connection management

### Key Features
- **Item Management**: Handles item/product-related database operations
- **Connection Pooling**: Standard MySQL connection pooling configuration

---

## redis_db.go - Redis Database Repository

### Database Connection and Management
- `NewRedisDB()`: Constructor that establishes Redis connection with context support
- WebRTC session and user status management through Redis

### Key Features
- **Session Storage**: JWT token and WebRTC session management
- **Real-time Status**: User availability and call status tracking
- **Cache Layer**: High-performance data caching for frequently accessed data

---

## types.go - Common Model Types

### Shared Data Structures
- Common types and constants used across all repository implementations
- Error definitions and utility types for database operations

---

## Overall Structure Summary

### Repository Pattern Implementation
- **4 Main Repositories**: AccountDB, HistoryDB, ItemDB, RedisDB
- **Centralized Management**: Single Repositories manager for all database connections
- **Type-Safe Access**: Reflection-based type-safe repository retrieval
- **Health Monitoring**: Automatic connection health checks every 2 minutes

### Database Technologies
- **MySQL**: Primary relational database for user, history, and item data
- **Redis**: In-memory database for sessions and real-time status

### Design Principles
- **Interface-Based Design**: All repositories implement IRepository interface
- **Dependency Injection**: Centralized creation and management
- **Thread Safety**: Mutex-protected concurrent access
- **Connection Pooling**: Optimized database connection management
- **Soft Delete Pattern**: Status-based deletion for data retention

### Security Features
- **ChaCha20 Encryption**: Encryption for sensitive user data
- **Prepared Statements**: SQL injection prevention
- **Connection Pooling**: Resource management and performance

### Key Features by Category
- **User Management**: Registration, authentication, profile management
- **Notification System**: Personal notifications and public announcements
- **History Tracking**: Call, chat, payment, and purchase history
- **Real-time Features**: WebRTC session management, user status tracking
- **Security**: Data encryption, soft delete, connection health monitoring

---
*Created: 2025-10-20*
*Last Updated: 2025-11-05*
*File Location: /home/jino/go/src/ms-gateway/models/*

