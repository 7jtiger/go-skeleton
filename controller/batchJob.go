package controller

import (
	haredis "ms-gateway/common/ha-redis"
	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	"ms-gateway/models"
	"strings"
	"time"
)

const (
	interval1H = 60 * time.Minute
	// totalTxInterval      = 60 * time.Minute
	// coinPriceInterval    = 1 * time.Minute
	// tatalPriceInterval   = 1 * time.Minute
	// pushInterval         = 60 * time.Second
	// exchangeRateInterval = 10 * time.Second
	// coinPriceInterval    = 10 * time.Second
	maxRetries = 3
	retryDelay = 5 * time.Second
)

const (
	HOUR int = 1
	DAY  int = 24
)

type CronJobController struct {
	ctl    *Controller
	cfg    *conf.Config
	adb    *models.AccountDB
	rdb    *models.RedisDB
	hch    *haredis.HAChecker
	start  chan struct{}
	stop   chan struct{}
	errors chan error
}

func NewCronJobController(h *Controller, hch *haredis.HAChecker, rep *models.Repositories) (*CronJobController, error) {
	r := &CronJobController{
		ctl:    h,
		cfg:    h.cfg,
		hch:    hch,
		start:  make(chan struct{}, 1),
		stop:   make(chan struct{}, 1),
		errors: make(chan error, 1),
	}

	if err := rep.Get(&r.adb, &r.rdb); err != nil {
		return nil, err
	}

	if strings.EqualFold(r.cfg.Server.CronJob, "on") {
		go r.cronJob()
	}

	return r, nil
}

func (p *CronJobController) Start() error {
	select {
	case p.start <- struct{}{}:
		log.Info("Success, Send Start Signal")
	default:
		log.Warn("CronJobController already started")
	}
	return nil
}

func (p *CronJobController) cronJob() {
	// 초기화 시도
	if err := p.initializeData(); err != nil {
		log.Info("Failed to initialize data:", err)
		// return
	}

	tick1HTicker := time.NewTicker(interval1H)
	//	tickTotalTx := time.NewTicker(totalTxInterval)
	defer tick1HTicker.Stop()
	// defer tickTotalTx.Stop()
	for {
		select {
		case <-p.stop:
			log.Info("CronJob shutting down gracefully")
			return
		case <-tick1HTicker.C:
			log.Info("CronJob 1H interval")
			// go p.update1HData()
		case err := <-p.errors:
			log.Error("Background task error:", err)
		}
	}
}

func (p *CronJobController) initializeData() error {
	return nil
}

func (p *CronJobController) safeExchangeRateUpdate() {
	if !p.hch.IsLeader() {
		return
	}

}
