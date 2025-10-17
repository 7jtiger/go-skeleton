# Account DB Function Summary

## Database Connection and Management Functions
- `NewAccountDB()`: Constructor function that establishes MySQL database connection and initializes AccountDB instance
- `Start()`: Function to start database service (currently no implementation)
- `Terminate()`: Function to safely terminate database connection and clean up resources
- `Close()`: Function to close database connection
- `Ping()`: Function to check database connection status
- `heartbeat()`: Background function that periodically checks database connection status

## User Existence Check Functions
- `IsExistID()`: Function to check if user ID already exists in the database
- `IsExistEmail()`: Function to check if user email already exists in the database

## User Authentication Functions
- `RegistUser()`: Function to encrypt and register new user information in the database
- `LoginUser()`: Function to verify password during user login and return user information
- `LogoutUser()`: Function to update last access time during user logout

## User Account Management Functions
- `LeaveUser()`: Function to change account status to inactive (stat=4) when user withdraws
- `DeleteUser()`: Function to delete user account but actually changes status to inactive (stat=4)

## Account Recovery and Change Functions
- `FindID()`: Function to search and return user ID list using name and birth date
- `FindPW()`: Function to check if password recovery is possible using ID, email, and birth date
- `ChangePW()`: Function to verify existing password and change to new encrypted password

## User Information Management Functions
- `ModifyUserInfo()`: Function to modify user information (nickname, area, email) by category
- `GetUserInfo()`: Function to query encrypted user information by user ID and return decrypted data

## Utility Functions
- `updatedLastest()`: Internal function to update user's last update time to current time


CREATE TABLE `user_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `sid` varchar(45) DEFAULT NULL,
  `uid` varchar(45) NOT NULL,
  `did` varchar(45) DEFAULT NULL,
  `pid` varchar(45) DEFAULT NULL,
  `email` varchar(45) DEFAULT NULL,
  `pw_hash` varchar(100) DEFAULT NULL,
  `join_plf` tinyint(1) DEFAULT NULL,
  `name` varchar(45) DEFAULT NULL,
  `nick` varchar(45) DEFAULT NULL,
  `gender` tinyint(1) DEFAULT NULL,
  `age` int DEFAULT NULL,
  `birthday` date DEFAULT NULL,
  `area` varchar(45) DEFAULT NULL,
  `stat` tinyint NOT NULL DEFAULT '0',
  `st_title` varchar(45) DEFAULT NULL,
  `block_user` json DEFAULT NULL,
  `hold_point` double DEFAULT NULL,
  `hold_cash` double DEFAULT NULL,
  `main_pic` varchar(256) DEFAULT NULL,
  `sub_pic` json DEFAULT NULL,
  `thmb_pic` varchar(128) DEFAULT NULL,
  `sp_intro` varchar(128) DEFAULT NULL,
  `at_join` date DEFAULT NULL,
  `at_upd` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`),
  UNIQUE KEY `sid_UNIQUE` (`sid`),
  UNIQUE KEY `did_UNIQUE` (`did`),
  UNIQUE KEY `pid_UNIQUE` (`pid`),
  UNIQUE KEY `email_UNIQUE` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=20 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


CREATE TABLE `pf_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(45) DEFAULT NULL,
  `sp_intro` varchar(128) DEFAULT NULL,
  `intro` varchar(256) DEFAULT NULL,
  `fw_cnt` int DEFAULT NULL,
  `fwing_cnt` int DEFAULT NULL,
  `fw_list` json DEFAULT NULL,
  `fwing_list` json DEFAULT NULL,
  `birthday` date DEFAULT NULL,
  `pf_pic` json DEFAULT NULL,
  `sub_pic` json DEFAULT NULL,
  `at_upd` date DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


CREATE TABLE `acc_info` (
  `idx` int NOT NULL,
  `uid` varchar(45) DEFAULT NULL,
  `aname` varchar(45) DEFAULT NULL COMMENT 'account name',
  `bname` varchar(45) DEFAULT NULL COMMENT 'bank name',
  `acc_num` varchar(45) DEFAULT NULL COMMENT 'account number',
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`),
  UNIQUE KEY `acc_num_UNIQUE` (`acc_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


---

## Key Features
- **Security**: Uses ChaCha20 encryption to encrypt sensitive information such as passwords, emails, and names
- **Connection Management**: Database connection pooling and connection status monitoring through heartbeat
- **Soft Delete**: Handles user deletion through status change rather than actual deletion

