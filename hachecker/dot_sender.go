package hachecker

import (
	"bufio"
	"encoding/json"

	// log "ms-gateway/common/logger"
	"net"
	"time"

	log "ms-gateway/common/logger"
)

func (p *HAChecker) checkPeerStatus() {
	for {
		conn, err := net.Dial("tcp", p.peerIP+":"+p.pPort)
		if err != nil {
			p.setServerStat("active")
			time.Sleep(2 * time.Second)
			continue
		}

		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Warn("Error reading response: ", err)
		} else {
			var result map[string]string
			if err := json.Unmarshal([]byte(message), &result); err != nil {
				log.Warn("Error parsing response: ", err)
			} else {
				p.setPeerLastNumber(result["lnum"], result["stat"])
				// log.Info("Peer message : ", result)
			}
		}
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func (p *HAChecker) setPeerLastNumber(sNum, sStat string) {
	// p.peerLastNum = sNum
	p.peerStatus = sStat
	if sStat == "active" {
		if p.curStatus == "active" {
			p.bActive = true
		} else {
			p.bActive = false
		}
	}
}
