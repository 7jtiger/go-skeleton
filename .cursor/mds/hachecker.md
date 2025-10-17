# HAChecker Package Function Feature Summary

## ping_server.go - HA Server Management

### HA Checker Core Structure
- `HAChecker struct`: High Availability checker structure containing server port, IPs, peer configuration, status tracking, and activity flags

### Initialization and Setup
- `NewHAChecker()`: Creates HA checker instance with configuration, starts TCP ping server and peer status monitoring goroutines

### TCP Server Management
- `pingTCPServer()`: Starts TCP listener on configured port to handle incoming health check connections from peer servers
- `handleConnection()`: Handles individual TCP connections by sending JSON response with server IP and current status

### Status Management
- `setServerStat()`: Updates current server status, resets peer status to ready, and sets active flag based on status
- `GetCurStatus()`: Returns current active status as boolean indicating if server is in active HA state

---

## dot_sender.go - Peer Status Monitoring

### Peer Communication Functions
- `checkPeerStatus()`: Continuously monitors peer server status by establishing TCP connections and parsing JSON responses
- `setPeerLastNumber()`: Updates peer status information and determines active state based on peer and current server status

---

## Overall Structure Summary

### Core Components (2 files, 6 functions)
- **ping_server.go**: HA server management (5 functions)
- **dot_sender.go**: Peer monitoring (2 functions)

### Key Features
- **High Availability**: Dual-server configuration with active/standby status management
- **Health Monitoring**: Continuous peer status checking with 2-second intervals
- **TCP Communication**: JSON-based status exchange between HA pair servers
- **Automatic Failover**: Dynamic active/standby switching based on peer availability
- **Status Tracking**: Real-time monitoring of both local and peer server states

---

## HA Architecture

### Server States
- **active**: Server is actively handling requests
- **ready**: Server is ready but in standby mode
- **inactive**: Server is not responding or unavailable

### Communication Protocol
- **TCP Connections**: Direct TCP communication on configured ports
- **JSON Messages**: Structured status exchange with server IP and status
- **Heartbeat Interval**: 2-second polling cycle for peer status checks

### Failover Logic
- **Peer Unavailable**: Local server becomes active when peer connection fails
- **Peer Active**: Local server remains standby when peer is active
- **Dual Active Prevention**: Logic prevents both servers from being active simultaneously

---

## Production Features
- **Fault Tolerance**: Automatic detection and handling of peer server failures
- **Network Monitoring**: Continuous TCP connection health checks
- **Configuration Driven**: Server ports and peer IPs configurable via config file
- **Graceful Handling**: Proper error handling for network failures and JSON parsing
- **Resource Management**: Proper connection cleanup and goroutine management

---

## Use Cases
- **Load Balancer Backend**: HA pair for backend services with automatic failover
- **Database Clustering**: Master/slave configuration with automatic promotion
- **Microservice Resilience**: Service instance management in distributed systems
- **Network Service HA**: Router, gateway, or proxy server high availability

---
*Created: 2025-09-15*
*File Location: /home/jino/go/src/ms-gateway/hachecker/*
