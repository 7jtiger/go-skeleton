package scheduler

import (
	log "ms-gateway/common/logger"
)

type Mailing struct {
}

func NewMailing() {
	log.Info("New mailing")
}

func (m *Mailing) SendMail() {
	log.Info("Send mail")
}
