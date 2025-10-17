package hachecker

import (
	"fmt"
	"net"

	conf "ms-gateway/conf"

	log "ms-gateway/common/logger"
	utils "ms-gateway/common/utils"
	// "github.com/rs/zerolog/log"
)

type HAChecker struct {
	sPort     string
	mIP       string
	peerIP    string
	pPort     string
	curStatus string
	// realPIP   string
	bActive bool
	// lastNum     string
	// peerLastNum string
	peerStatus string
}

func NewHAChecker(cfg *conf.Config) *HAChecker {
	r := &HAChecker{
		sPort:     cfg.HAChecker.ServerPort,
		mIP:       utils.GetLocalIP(),
		peerIP:    cfg.HAChecker.PeerIP,
		pPort:     cfg.HAChecker.PeerPort,
		curStatus: "ready",
		bActive:   cfg.HAChecker.SetStatus == "active",
		// lastNum:     "0",
		// peerLastNum: "0",
		peerStatus: "ready",
	}

	go r.pingTCPServer()

	go r.checkPeerStatus()

	// go r.check() //only test

	return r
}

func (p *HAChecker) pingTCPServer() {
	listener, err := net.Listen("tcp", ":"+p.sPort)
	if err != nil {
		log.Warn("Error starting TCP server: ", err)
		return
	}
	defer listener.Close()

	log.Info("Starting HA Checker server on port ", p.sPort)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warn("Error accepting connection: ", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *HAChecker) handleConnection(conn net.Conn) {
	defer conn.Close()
	response := fmt.Sprintf("{\"sip\": \"%s\", \"stat\": \"%s\"}", p.mIP, p.curStatus)
	conn.Write([]byte(response + "\n"))
}

func (p *HAChecker) setServerStat(stat string) {
	p.curStatus = stat
	p.peerStatus = "ready"
	p.bActive = stat == "active"
}

func (p *HAChecker) GetCurStatus() bool {
	return p.bActive

}
