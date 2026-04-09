CREATE TABLE `dm_room` (
  `idx`       INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id`   VARCHAR(64)  NOT NULL,
  `user1_uid` BIGINT UNSIGNED NOT NULL,
  `user2_uid` BIGINT UNSIGNED NOT NULL,
  `stat`      TINYINT NOT NULL DEFAULT 1 COMMENT '0=deleted, 1=active',
  `at_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `at_update` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `uk_room_id` (`room_id`),
  UNIQUE KEY `uk_pair` (`user1_uid`, `user2_uid`),
  INDEX `ix_user1` (`user1_uid`),
  INDEX `ix_user2` (`user2_uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
