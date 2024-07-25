package scheduler

import (
	log "basesk/common/logger"
)

type Mailing struct {
}

func NewMailing() {
	log.Info("New mailing")
}

func (m *Mailing) SendMail() {
	log.Info("Send mail")
}
