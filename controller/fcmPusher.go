package controller

import (
	"context"
	"database/sql"
	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	"ms-gateway/hachecker"
	"ms-gateway/models"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type SendFCMElem struct {
	NotiTitle string
	NotiBody  string
	Cate      string
	Did       string
}

type FCMPusher struct {
	ctl       *Controller
	cfg       *conf.Config
	hch       *hachecker.HAChecker
	fcmClient *messaging.Client
	pushDB    *models.ItemDB
	accountDB *models.AccountDB
	historyDB *models.HistoryDB

	sendmsg chan *SendFCMElem
	stop    chan chan<- struct{}
}

const (
	NOTI_ITEM = 0
	PRIV_ITEM = 1
)

func NewFCMPusher(root *Controller, hch *hachecker.HAChecker, rep *models.Repositories) (*FCMPusher, error) {
	ctx := context.Background()

	// Firebase Admin SDK 초기화
	opt := option.WithCredentialsFile(root.cfg.ApiInfo.FcmKeyPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}

	// FCM 클라이언트 생성
	fcmClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	r := &FCMPusher{
		ctl:       root,
		cfg:       root.cfg,
		hch:       hch,
		fcmClient: fcmClient,
		sendmsg:   make(chan *SendFCMElem, 1000),
		stop:      make(chan chan<- struct{}),
	}

	if err := rep.Get(&r.pushDB, &r.accountDB, &r.historyDB); err != nil {
		log.Crit("newFcmSender", "error", err)
		return nil, err
	}

	go r.loop()
	return r, nil
}

func (p *FCMPusher) Terminate() {
	fin := make(chan struct{})
	p.stop <- fin
	<-fin

	if p.ctl != nil {
		p.stop <- fin
	}

	log.Info("terminated instance FCMPusher")
}

func (p *FCMPusher) loop() {

	if !p.hch.GetCurStatus() {
		for {
			if p.hch.GetCurStatus() {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	tickNotiItem := time.NewTicker(5 * time.Minute)
	tickTgtItem := time.NewTicker(1 * time.Second)
	tickDelItem := time.NewTicker(24 * 30 * time.Hour)
	defer tickNotiItem.Stop()
	defer tickTgtItem.Stop()
	defer tickDelItem.Stop()

	for {
		select {
		case fin := <-p.stop:
			fin <- struct{}{}
			return
		case e := <-p.sendmsg:
			p.sendFCM(e.NotiTitle, e.NotiBody, e.Cate, e.Did)
			time.Sleep(1 * time.Second)
		case <-tickNotiItem.C: // 5m
			// log.Info("send Noti Item")
			p.getNotiItem()
		case <-tickTgtItem.C: // 1s
			// log.Info("send Tgt Item")
			p.getTgtItem()
		case <-tickDelItem.C: // 24h
			log.Info("sand Del Item")
		}
	}
}

// func (p *FCMPusher) WriteSender(notiTitle string, notiBody string, cate int, did string) {
func (p *FCMPusher) WriteSender(notiTitle, notiBody, cate, did string) {
	p.sendmsg <- &SendFCMElem{
		NotiTitle: notiTitle,
		NotiBody:  notiBody,
		Cate:      cate,
		Did:       did,
	}
}

/*
// func (p *FCMSender) sendFCM(notiTitle, notiBody, did, id string) {
func (p *FCMPusher) sendFCM(notiTitle, notiBody, cate, did string) {
	data := map[string]string{
		"title": notiTitle,
		"body":  notiBody,
		"cate":  cate,
		"id":    did,
		"sum":   "send by Gurufin",
	}
	ids := []string{did}

	p.fcmClient.SetMsgData(data)
	p.fcmClient.Message.RegistrationIds = ids

	// 푸쉬 팝업에 표시되는 정보.
	notiItem := fcm.NotificationPayload{
		Title: notiTitle,
		Body:  notiBody,
		Sound: "default",
	}

	p.fcmClient.SetNotificationPayload(&notiItem)
	status, err := p.fcmClient.Send()
	if err != nil {
		log.Error("FCM push failed", "status", status, "error", err)
		return
	}

	log.Info("FCM push successful", "status", status)
}
*/

func (p *FCMPusher) sendFCM(notiTitle, notiBody, cate, did string) {
	ctx := context.Background()

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: notiTitle,
			Body:  notiBody,
		},
		Data: map[string]string{
			"category":     cate,
			"click_action": "OPEN_ACTIVITY_1",
			"type":         "test",
			"id":           "1",
			"sum":          "send by Gurufin",
		},
		Token: did, // 단일 디바이스 토큰
	}

	// 메시지 전송
	var response *messaging.SendResponse
	// var err error
	for rtry := 0; rtry < 3; rtry++ {
		resp, err := p.fcmClient.Send(ctx, message)
		if err == nil {
			break
		}
		// time.Sleep(1 * time.Second) // 리트라이 간 대기 시간 추가

		// 토큰이 더 이상 유효하지 않은 경우 처리
		if rtry == 3 {
			log.Info("FCM push failed", " retryCount ", rtry+1, " error ", err, " response ", resp)
			if messaging.IsRegistrationTokenNotRegistered(err) {
				// TODO: 필요한 경우 무효한 토큰 제거 로직 추가
				log.Info("Invalid token", "token", did)
				return
			}
		}
	}

	log.Info("FCM push successful", "messageId", response)
	//TODO : db 업데이트 처리
}

// 여러 did 를 묶어서 처리
func (p *FCMPusher) SendFCMs(item *models.NotiItem, cate int, tokens []string) error {
	ctx := context.Background()

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: item.Title,
			Body:  item.Msg,
		},
		Data: map[string]string{
			"category": strconv.Itoa(cate),
			"itemId":   item.Idx,
		},
		Tokens: tokens,
	}

	// 멀티캐스트 메시지 전송
	response, err := p.fcmClient.SendMulticast(ctx, message)
	if err != nil {
		log.Error("SendFCMs", "error", err.Error())
		return err
	}

	// 실패한 토큰 처리
	if response.FailureCount > 0 {
		var failedTokens []string
		for idx, resp := range response.Responses {
			if !resp.Success {
				retryCount := 0
				for retryCount < 3 {
					retryResponse, retryErr := p.fcmClient.SendMulticast(ctx, message)
					if retryErr == nil && retryResponse.Responses[idx].Success {
						break
					}
					retryCount++
				}
				//실패는 없다고 처리 완료후 notiitem 업데이트
				/* 				if retryCount == 3 {
					failedTokens = append(failedTokens, tokens[idx])
					if messaging.IsRegistrationTokenNotRegistered(resp.Error) {
						// 유효하지 않은 토큰 제거 로직
						p.accountDB.RemoveInvalidToken(tokens[idx])
					}
				} */
			}
		}
		log.Info("SendFCMs", "failureCount", response.FailureCount, "failedTokens", failedTokens)
	}

	return nil
}

// ///////////////////////////////////////////////////////////
// process
// push_items stat 0 기준 조회 0 : 미처리
// sendFCM 전달
// push_items stat 1 로 업데이트
// historydb - txs tx 저장

// cate 1 : txs
// cate <= 2 : contract
func (p *FCMPusher) getTgtItem() {
	// log.Info("getTgtItem start")
	items := p.pushDB.GetPushItem()
	if items == nil {
		log.Error("Failed to get push items")
		return
	}

	for _, item := range *items {
		if item.Cate >= 2 {
			continue
		}

		sendOnOff, err := p.accountDB.GetFromDidSendOnOff(item.Did)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("No rows found ", " did ", item.Did, " error ", err)
				if _, err := p.pushDB.SendUpdateItem(PRIV_ITEM, item.Idx); err != nil {
					log.Error("Failed to update push item", "error", err)
					continue
				}
				break
			}
			log.Error("Failed to get send on off", "error", err)
			continue
		} else if !sendOnOff {
			if _, err := p.pushDB.SendUpdateItem(PRIV_ITEM, item.Idx); err != nil {
				log.Error("Failed to update push item", "error", err)
				continue
			}
			break
		}

		title, body, cateStr := p.getMsgTitle(item.Cate, item.Chain, item.Amount.String())
		p.WriteSender(title, body, cateStr, item.Did)

		if _, err := p.pushDB.SendUpdateItem(PRIV_ITEM, item.Idx); err != nil {
			log.Error("Failed to update push item", "error", err)
			continue
		}

		if err := p.historyDB.SaveTxsHistory(&item, cateStr); err != nil {
			log.Error("Failed to save transaction history", "error", err)
		}
	}
}

func (p *FCMPusher) getMsgTitle(cate int, chain, amount string) (string, string, string) {
	if strings.Contains(chain, p.cfg.Server.GuruChainID) {
		chain = "GXN"
	} else if strings.Contains(chain, p.cfg.Server.UsdxChainID) {
		chain = "USGX"
	}
	/* if strings.Contains(chain, "3110") || strings.Contains(chain, "3111") {
		chain = "Guru"
	} else if strings.Contains(chain, "5110") || strings.Contains(chain, "5111") {
		chain = "USDX"
	} */

	amtEth := utils.ToEther(amount)
	amtEth = amtEth.RoundFloor(6)

	var title, body, cateStr string
	switch cate {
	case 0:
		title = "Send " + chain
		body = "Send " + amtEth.String() + " " + chain
		cateStr = "SEND"
	case 1:
		title = "Receive " + chain
		body = "Receive " + amtEth.String() + " " + chain
		cateStr = "RECV"
	case 2: // NFT
		title = "NFT Transfer" + chain
		body = "NFT " + amtEth.String() + " " + chain
		cateStr = "NFT"
	}
	// case 3: // lock
	// 	title = "NFT Transfer" + chain
	// 	body = "" + amtEth.String() + " " + chain
	// 	cateStr = "NFT"
	//2 : UNLOCK

	return title, body, cateStr
}

// ///////////////////////////////////////////////////////////
// 전체 노티 푸쉬
// 1. massage
// 2. notices
// 3. event
// 1110 14 m n e
// 1100 12 m n
// 1010 10 m e
// 1000 8  m
// 0110 6  n e
// 0100 4  n
// 0010 2  e
// 0000 0

const (
	MSG  = 1
	NOTI = 2
	EVNT = 3
)

func (p *FCMPusher) getNotiItem() {
	log.Info("getNotiItem start")
	// get table from push_notiitem
	// 1. get cate 1 : massage send, set 14, 12, 10, 8 from accoutdb.user_pref
	// get accountdb.user_pref did
	// 2. get cate 2 : notice send, set 14, 12, 6, 4 from accoutdb.user_pref
	// get accountdb.user_pref did
	// 3. get cate 3 : event send, set 14, 10, 6, 2 from accoutdb.user_pref
	// get accountdb.user_pref did

	categories := []int{MSG, NOTI, EVNT}

	for _, cate := range categories {
		item := p.pushDB.GetNotiItem(cate)
		if item != nil {
			if err := p.CollectNotiSend(item, cate); err != nil {
				log.Error("fcmpool", "getNotiItem", err.Error())
			}

			idx, _ := strconv.Atoi(item.Idx)
			p.pushDB.SendUpdateItem(NOTI_ITEM, idx)
		}
	}
}

func (p *FCMPusher) CollectNotiSend(item *models.NotiItem, cate int) error {
	dids, err := p.accountDB.GetTargetDids(cate)
	if err != nil {
		return err
	}

	// 배치 처리 (Firebase 권장 최대 500개)
	const batchSize = 500
	ntot := len(*dids)

	for i := 0; i < ntot; i += batchSize {
		end := i + batchSize
		if end > ntot {
			end = ntot
		}
		batch := (*dids)[i:end]

		if err := p.SendFCMs(item, cate, batch); err != nil {
			log.Info("CollectNotiSend", "error", err.Error())
			continue
		}

		// API 속도 제한 방지를 위한 대기
		time.Sleep(1 * time.Second)
	}
	//TODO : db 업데이트 처리

	return nil
}
