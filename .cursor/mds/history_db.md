# History DB Function Feature Summary

## Database Connection and Management Functions
- `NewHistoryDB()`: Constructor function that establishes MySQL history database connection and initializes HistoryDB instance
- `Start()`: Function to start history database service (currently no implementation)
- `Terminate()`: Function to safely terminate history database connection and clean up resources
- `Close()`: Function to close history database connection
- `Ping()`: Function to check history database connection status
- `heartbeat()`: Background function that periodically checks history database connection status

## Notification Management Functions
- `GetNotiAllList()`: Function to retrieve all personal notifications for a user, ordered by index descending
- `GetNotiCount()`: Function to count unread notifications (stat=0) for a specific user
- `GetNotiDetail()`: Function to retrieve detailed information for a specific notification by index
- `SetNotiRead()`: Function to mark a notification as read by setting stat to 1

## Announcement Management Functions
- `SaveAnnouncement()`: Function to save new public announcement with title, body, URL, and timestamp
- `GetAnnouncementList()`: Function to retrieve latest 20 announcements with automatic 'new' badge (stat=1) for items within 7 days
- `GetAnnouncementDetail()`: Function to retrieve specific announcement detail with automatic 'new' badge calculation based on 7-day window


CREATE TABLE `buy_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `pay_amt` double DEFAULT NULL,
  `tot_point` double DEFAULT NULL,
  `balance` double DEFAULT NULL,
  `at_buy` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


CREATE TABLE `by_flw` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `tot_count` int DEFAULT NULL,
  `by_flow` json DEFAULT NULL,
  `upd_cnt` int DEFAULT NULL,
  `at_upd` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `call_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `tid` varchar(45) DEFAULT NULL,
  `tnick` varchar(45) DEFAULT NULL,
  `tage` int DEFAULT NULL,
  `st_call` tinyint(1) DEFAULT NULL,
  `st_cnt` tinyint(1) DEFAULT NULL,
  `st_view` tinyint(1) DEFAULT NULL,
  `at_call` date DEFAULT NULL,
  `sec_duration` int DEFAULT NULL,
  `paid_point` double DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`),
  UNIQUE KEY `tid_UNIQUE` (`tid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `chat_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `tid` varchar(45) DEFAULT NULL,
  `tnick` varchar(45) DEFAULT NULL,
  `tage` int DEFAULT NULL,
  `st_chat` tinyint(1) DEFAULT NULL,
  `st_cnt` tinyint(1) DEFAULT NULL,
  `at_chat` date DEFAULT NULL,
  `st_view` tinyint(1) DEFAULT NULL,
  `chat_cnt` int DEFAULT NULL COMMENT 'chat count',
  `paid_point` double DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`),
  UNIQUE KEY `tid_UNIQUE` (`tid`),
  UNIQUE KEY `tnick_UNIQUE` (`tnick`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `declare_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `ty_declare` tinyint(1) DEFAULT NULL,
  `dc_detail` varchar(1000) DEFAULT NULL,
  `at_declare` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `flw_list` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `tot_count` int DEFAULT NULL,
  `flw_list` json DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `pay_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `ty_pay` tinyint(1) DEFAULT NULL,
  `charge_fee` double DEFAULT NULL,
  `amount` double DEFAULT NULL,
  `ex_point` double DEFAULT NULL,
  `tot_point` double DEFAULT NULL,
  `at_paid` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `noti_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint unsigned NOT NULL,
  `nt_type` tinyint(4) unsigned zerofill NOT NULL,
  `nt_title` varchar(20) NOT NULL,
  `nt_msg` varchar(50) NOT NULL,
  `at_noti` datetime NOT NULL,
  `stat` tinyint(1) unsigned zerofill NOT NULL,
  `frm_uid` bigint unsigned NOT NULL,
  `frm_url` varchar(100) NOT NULL,
  `frm_nick` varchar(15) NOT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='noti list'

CREATE TABLE `anuc_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `an_title` varchar(45) NOT NULL,
  `an_body` tinytext,
  `an_url` varchar(150) DEFAULT NULL,
  `at_msg` datetime NOT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

---

## DM Room 추가 (2026-03)

### 신규 구조체
- `DMRoomRow`: `dm_room` 테이블 로우 매핑 구조체

### 신규 함수
- `CreateDMRoom(uid uint64, tUser *ptl.UserInfoResp)`: DM 방 생성 (room_id는 uid 정렬로 자동 생성)
- `GetDMRoom(ridx int64)`: idx 기준 단건 조회
- `GetDMRoomByPair(uid, tid uint64)`: 사용자 쌍 **양방향** 조회 (`(uid=A AND tid=B) OR (uid=B AND tid=A)`)
- `GetDMRoomByRid(rid string)`: room_id 문자열 기준 단건 조회
- `GetDMRoomsByUser(uid uint64)`: 사용자 참여 DM 방 목록 조회 (uid 또는 tid로 참여한 방 UNION, 최신 업데이트순)
- `SoftDeleteDMRoom(ridx int64)`: idx 기준 soft delete(`st_chat=0`)
- `SoftDeleteDMRoomsByUser(uid, tid uint64)`: room_id 기준 soft delete
- `ActivateDMRoomByPair(uid, tid uint64)`: 기존 soft-deleted DM 방 **양방향** 재활성화(`st_chat=1`)
- `UdtDMPaid(ridx int64, point float64)`: DM 방 paid_point 누적 업데이트

### 구현 규칙
- `room_id` 생성 시 `min(uid,tid)_max(uid,tid)` 정규화 적용 (`getRoomID`)
- 삭제는 hard delete가 아닌 `st_chat` 기반 soft delete 사용
- **양방향 조회**: `GetDMRoomByPair`, `ActivateDMRoomByPair`는 A→B, B→A 양방향 검색 지원

CREATE TABLE `app_push` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `msg` varchar(256) NOT NULL,
  `type` int NOT NULL DEFAULT '0' COMMENT 'type=0: nomal push\ntype=1: noti push',
  `did` varchar(145) NOT NULL,
  `stat` int NOT NULL DEFAULT '0' COMMENT 'stat=0 : before send\nstat=1 : sand',
  `at_reg` date DEFAULT NULL,
  `at_sent` date DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `inmsg_push` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `msg` varchar(256) NOT NULL,
  `type` int NOT NULL DEFAULT '0' COMMENT 'type=0 : video call noti\ntype=1 : audio call noti\ntype=2: text chat noti',
  `did` varchar(145) NOT NULL,
  `stat` int NOT NULL DEFAULT '0' COMMENT 'stat=0: before recv\nstat=1: sand\nstat=2: accepted\nstat=3: disagree',
  `at_send` date DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

---

## Structure Information
- **HistoryDB**: Database access layer structure that manages history-related data

---

## Key Features
- **Connection Management**: MySQL database connection pooling setup (maximum 300 connections, 30 idle connections)
- **Health Check**: Database connection status monitoring every 2 minutes
- **Resource Management**: Channel-based resource cleanup for safe termination
- **Logging**: Log recording during connection setup and termination
- **Notification System**: Personal user notifications with read/unread status management
- **Announcement System**: Public announcements with automatic 'new' badge calculation (7-day window)
- **Auto-Status Calculation**: Automatic stat field calculation based on timestamp comparison

---
*Created: 2025-09-15*
*Last Updated: 2025-11-05*
*File Location: /home/jino/go/src/ms-gateway/models/history_db.go*

---

## Database Configuration
- **Connection Setup**: MySQL database connection using `hdb` configuration
- **Connection Options**: Time parsing enabled with `parseTime=true` option
- **Pooling Settings**: 
  - Maximum idle connections: 30
  - Maximum open connections: 300
  - Connection maximum lifetime: 3 minutes

