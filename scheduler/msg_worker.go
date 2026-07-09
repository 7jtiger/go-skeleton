package scheduler

import (
	"time"
)

// MsgDeleter removes DM messages older than 3 days from Redis.
func (s *Schedule) MsgDeleter() error {
	_, err := s.rdb.ExpiredMsg(3 * 24 * time.Hour)
	if err != nil {
		return err
	}

	/* 	log.Info("msg worker complete - scanned rooms: %d, deleted messages: %d",
		result.ScannedRooms,
		result.DeletedMsgs,
	) */
	return nil
}
