# Conf Package Function Feature Summary

## config.go - Configuration Management

### Configuration Structures

#### ServerConf struct
- `Name string`: Server name identifier
- `Mode string`: Server running mode (dev, prod, test)
- `Port string`: Server listening port number
- `HCheck string`: Health check endpoint path
- `HCPort string`: Health check port
- `BaseKey string`: Base encryption key
- `JWTSecret string`: JWT token signing secret key

#### Works struct
- `Name string`: Task name identifier
- `Desc string`: Task description
- `Execute string`: Execution flag (run/exe/o for immediate, cron for cron-style)
- `Duration int`: Task interval duration in seconds
- `Start int`: Start time configuration (0=daily, 1=hourly, 5=5min interval, custom seconds)
- `Args string`: Task arguments

#### Config struct - Server Section
- `Server.Name string`: Server name
- `Server.Mode string`: Running mode (dev/prod/test)
- `Server.Port string`: Listening port
- `Server.HCheck string`: Health check path
- `Server.HCPort string`: Health check port
- `Server.BaseKey string`: Base encryption key
- `Server.JWTSecret string`: JWT secret key

#### Config struct - WebRTC Section (STUN only)
- `WebRTC.SignalingServerUrl string`: WebRTC signaling server URL
- `WebRTC.StunServers []string`: Array of STUN server addresses (Google STUN servers)
- `WebRTC.MaxVideoWidth int`: Maximum video resolution width
- `WebRTC.MaxVideoHeight int`: Maximum video resolution height
- `WebRTC.MaxVideoFrameRate int`: Maximum video frame rate
- `WebRTC.VideoBitrate int`: Video bitrate setting
- `WebRTC.AudioBitrate int`: Audio bitrate setting
- `WebRTC.SessionTimeout int`: Session timeout in seconds
- `WebRTC.HeartbeatInterval int`: Heartbeat interval in seconds

**Note**: TURN server support has been removed to avoid relay traffic overhead. Only STUN servers are used for NAT traversal.

#### Config struct - HAChecker Section
- `HAChecker.Checker bool`: HA checker activation flag
- `HAChecker.ServerPort string`: HA server port
- `HAChecker.PeerIP string`: Peer server IP address
- `HAChecker.PeerPort string`: Peer server port
- `HAChecker.SetStatus string`: Initial HA status (active/ready/inactive)

#### Config struct - Other Sections
- `DB map[string]map[string]interface{}`: Database configurations by name
- `Works []Works`: Scheduled task configurations array
- `LogInfo.Fpath string`: Log file path
- `LogInfo.MaxAgeHour int`: Maximum log file age in hours
- `LogInfo.RotateHour int`: Log rotation interval in hours
- `WhiteList.Ips []string`: Allowed IP addresses for admin operations

### Configuration Functions
- `NewConfig(fpath string)`: Loads configuration from specified TOML file path using naoina/toml library, panics on error

---

## config.toml - Configuration File Structure

### Configuration Sections

#### [server]
- Server identity, mode, ports, and security keys

#### [webrtc]
- WebRTC signaling, STUN servers (Google), media quality settings, session management
- **TURN server configuration removed** - uses only STUN for NAT traversal to avoid relay traffic costs

#### [hachecker]
- High availability checker configuration with peer server details

#### [db.adb], [db.hdb], [db.idb], [db.rdb]
- Database connection settings for different database types
  - Account database (adb)
  - History database (hdb)
  - Item database (idb)
  - Redis database (rdb)

#### [loginfo]
- Logging configuration with file paths and rotation settings

#### [whitelist]
- IP addresses allowed for admin operations

#### [[works]]
- Multiple scheduled task configurations
- Each task with name, description, execution mode, start time, and duration

---

## Key Features
- **TOML Format**: Human-readable configuration file format using naoina/toml
- **Environment Separation**: Support for dev, prod, test modes
- **WebRTC Support**: Comprehensive WebRTC and TURN server configuration
- **Database Flexibility**: Multiple database configurations with flexible interface{} values
- **Security Configuration**: IP whitelist, JWT settings, encryption keys
- **HA Support**: High availability checker configuration with peer management
- **Scheduler Integration**: Task scheduling configuration built-in
- **Media Quality Control**: Configurable video/audio bitrate and resolution limits
- **Session Management**: Timeout and heartbeat interval configuration
- **Type Safety**: Strong typing for all configuration fields

---

## Production Features
- **STUN Servers**: Google STUN servers for NAT traversal (no relay traffic)
- **Health Monitoring**: HA checker for service availability
- **Logging Management**: Rotation and retention policies
- **Resource Limits**: Connection pooling and session management

---

## Usage Pattern
```go
cfg := conf.NewConfig("./conf/config.toml")

// Access configuration
serverPort := cfg.Server.Port
jwtSecret := cfg.Server.JWTSecret
stunServers := cfg.WebRTC.StunServers  // Google STUN servers only
dbConfig := cfg.DB["adb"]
haStatus := cfg.HAChecker.SetStatus
```

---
*Created: 2025-10-20*
*Last Updated: 2025-11-07*
*File Location: /home/jino/go/src/ms-gateway/conf/*
*Note: TURN server support removed on 2025-11-07 to avoid relay traffic overhead*

