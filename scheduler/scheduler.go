package scheduler

import (
	"context"
	"fmt"
	haredis "ms-gateway/common/ha-redis"
	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	"ms-gateway/models"

	"strings"
	"sync"
	"time"
)

//get config
//sort enable item
//

type item struct {
	name   string
	desc   string
	args   string
	delay  time.Duration
	ticker time.Ticker
	quit   chan int
	once   sync.Once
}

var SchHandler *Schedule

type Schedule struct {
	sync.RWMutex
	cfg *conf.Config
	hch *haredis.HAChecker

	Desc string
	Item map[string]*item
	adb  *models.AccountDB
	// sdb  *models.ScanDB
	// cdb  *models.ContractDB
	rdb *models.RedisDB
	// stdb *models.StakingDB

	firstStat string
	ctx       context.Context
	cancel    context.CancelFunc
	quit      chan struct{}
	stopOnce  sync.Once
}

func getDuration(start int) time.Duration {
	var rettime time.Duration
	t := time.Now()
	switch start {
	case 0: // 1day 00:00:00 start
		n := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
		rettime = n.Sub(t)
	case 1: // **:**:00 start
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
		rettime = n.Sub(t)
	case 5: // duration 5min
		g := 5 - (t.Minute() % int(5))
		if g == 0 {
			g = 5
		}
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+g, 0, 0, t.Location())
		rettime = n.Sub(t)
	default: // + start second start
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()+start, 0, t.Location())
		rettime = n.Sub(t)
	}

	return rettime
}

func NewScheduler(cfg *conf.Config, hch *haredis.HAChecker, rep *models.Repositories) (*Schedule, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if strings.EqualFold(cfg.Server.SchWorker, "off") {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Schedule{
		cfg:    cfg,
		hch:    hch,
		Desc:   "scheduler",
		Item:   make(map[string]*item),
		ctx:    ctx,
		cancel: cancel,
		quit:   make(chan struct{}),
	}

	if err := rep.Get(&s.adb, &s.rdb); err != nil {
		return nil, fmt.Errorf("db connect error: %w", err)
	}

	for _, ejob := range cfg.Works {
		if err := s.addJob(ejob); err != nil {
			s.Stop()
			return nil, fmt.Errorf("failed to add job %s: %w", ejob.Name, err)
		}
	}

	go s.initExec()
	return s, nil
}

func (s *Schedule) addJob(ejob conf.Works) error {
	s.Lock()
	defer s.Unlock()

	ex := strings.ToLower(ejob.Execute)
	if ex != "run" && ex != "exe" && ex != "o" && ex != "0" {
		return nil
	}

	tick := getDuration(ejob.Start)
	it := &item{
		name: ejob.Name,
		desc: ejob.Desc,
		args: ejob.Args,
		// delay:  time.Duration(ejob.Duration) * time.Second,
		delay:  time.Duration(ejob.Duration),
		ticker: *time.NewTicker(tick),
		quit:   make(chan int),
	}

	s.Item[ejob.Name] = it
	go s.runJob(it)
	log.Info("Job added: ", ejob.Name, " in ", tick, " desc : ", ejob.Desc, " args : ", ejob.Args)
	return nil
}

func (s *Schedule) getBaseDay() string {
	return s.firstStat
}

func (s *Schedule) initExec() {
	if strings.EqualFold(s.cfg.Server.Mode, "dev") {
		log.Info("Scheduler initialized in dev mode")
	}

	log.Info("Scheduler initialized in prod mode")
}

func (s *Schedule) setBaseDay(stat string) {
	s.firstStat = stat
}

func (s *Schedule) runJob(it *item) {
	defer func() {
		if r := recover(); r != nil {
			log.Info("Recovered from panic in job %s: %v", it.name, r)
		}
		s.close(it)
	}()

	// firstExec := true
	for {
		select {
		case <-it.ticker.C:
			// if firstExec {
			// 	it.ticker.Stop()
			// 	it.ticker = *time.NewTicker(it.delay)
			// 	firstExec = false
			// }

			if err := s.task(it); err != nil {
				log.Info("Error executing job %s: %v", it.name, err)
			}

		case <-it.quit:
			return

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Schedule) task(it *item) error {
	tname := strings.ToLower(it.name)
	// t := time.Now()
	switch tname {
	case "month11check":
		log.Info("task11 ", "Store ")
	case "preweekcheck":
		log.Info("task22 ", "SchdulePreWeekCheck ")
	case "msg_remove":
		log.Info("msg_remove ", "Remove messages older than 24 hours")
		return s.MsgDeleter()
	default:
		return nil
	}

	return nil
}

func (s *Schedule) close(item *item) {
	if item == nil {
		return
	}
	item.once.Do(func() {
		item.ticker.Stop()
		close(item.quit)
	})
}

func (s *Schedule) Stop() {
	s.stopOnce.Do(func() {
		s.Lock()
		defer s.Unlock()

		if s.cancel != nil {
			s.cancel()
		}
		close(s.quit)
		for _, item := range s.Item {
			s.close(item)
		}
	})
}
