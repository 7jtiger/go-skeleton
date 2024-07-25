package scheduler

import (
	logs "basesk/common/logger"
	"basesk/conf"
	"basesk/models"
	"fmt"
	"strings"
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
}

var SchHandler *Schedule

type Schedule struct {
	cfg *conf.Config
	// rep  *models.Repositories
	Desc string
	Item map[string]*item
}

func getDuration(start int) time.Duration {
	var rettime time.Duration
	t := time.Now()
	if start == 0 { // 1day 00:00:00 start
		n := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
		rettime = n.Sub(t)
	} else if start == 1 { //**:**:00 start
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
		rettime = n.Sub(t)
	} else if start == 5 { //duration 5min
		g := 5 - (t.Minute() % int(5))
		if g == 0 {
			g = 5
		}
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+g, 0, 0, t.Location())
		rettime = n.Sub(t)
	} else { // + start secont start
		n := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()+start, 0, t.Location())
		rettime = n.Sub(t)
	}

	return rettime
}

func NewScheduler(cfg *conf.Config, rep *models.Repositories) (*Schedule, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	SchHandler = &Schedule{
		cfg:  cfg,
		Desc: "test",
		Item: make(map[string]*item),
	}

	//init
	cr := make(map[string]*item)
	for _, ejob := range cfg.Works {
		ex := strings.ToLower(ejob.Execute)
		tick := getDuration(ejob.Start)
		if ex == "run" || ex == "exe" || ex == "o" {
			it := &item{
				name:   ejob.Name,
				desc:   ejob.Desc,
				args:   ejob.Args,
				delay:  time.Duration(ejob.Duration) * time.Second,
				ticker: *time.NewTicker(tick),
				quit:   make(chan int),
			}
			SchHandler.Scheduler(it)
			SchHandler.Item[ejob.Name] = it
		} else if ex == "cron" {
			cr[ejob.Name] = &item{
				name: ejob.Name,
				desc: ejob.Desc,
				args: ejob.Args,
			}
		}
	}

	return SchHandler, nil
}

func (s *Schedule) Scheduler(it *item) {
	go func() {
		defer func() {
			s.close(it)
			fmt.Println("Scheduler stopped!!")
		}()

		firstExec := true
		for {
			select {
			case <-it.ticker.C:
				if firstExec {
					it.ticker.Stop()
					it.ticker = *time.NewTicker(it.delay)
					firstExec = false
				}

				s.task(it)
				// break
			case <-it.quit:
				fmt.Println("item ticker stopped!!")
				s.close(it)
				// break
			}
		}

	}()
}

func (s *Schedule) close(item *item) {
	item.ticker.Stop()
}

func (s *Schedule) task(it *item) {
	tname := strings.ToLower(it.name)
	t := time.Now()
	//lower cast
	if tname == "depositcheck" {
		logs.Info("task", "Store", t)
		// go s.FirstChecker()
	} else if tname == "makeexceldaily" {
		logs.Info("task", "SchduleMakeExcel", t)
		go tmp("test")
	} else if tname == "1min" {
		go tmp(it.desc)
	} else {
		go tmp(it.desc)
	}
}

func tmp(sz string) {
	//fmt.Println("tmp work -- ", sz)
}
