package scheduler

import (
	log "ms-gateway/common/logger"
	"time"
)

func (s *Schedule) MsgDeleter() error {
	result, err := s.rdb.Expired24hMsg(24 * time.Hour)
	if err != nil {
		return err
	}

	log.Info("msg worker complete - scanned rooms: %d, deleted messages: %d",
		result.ScannedRooms,
		result.DeletedMsgs,
	)
	return nil
}
