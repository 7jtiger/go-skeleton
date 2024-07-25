package maps

import (
	"fmt"
	"sync"

	com "go-skeleton/common"
)

type GenMap struct {
	gm    map[string]string
	count int
	lock  *sync.Mutex
}

const UNIT uint = 1000

func NewGenMap() *GenMap {
	return &GenMap{
		gm:    make(map[string]string, 10000),
		lock:  &sync.Mutex{},
		count: 0,
	}
}

func (m *GenMap) cleanPool() *GenMap {
	m.gm = nil
	m.count = 0
	return NewGenMap()
}

/*
func (m *GenMap) buffering(mt *GenMap, nlen int) *GenMap {
	var swap map[string]string
	swap = mt.gm
	mt.gm = nil
	mt.gm = make(map[string]string, mt.count+1000)
	//copy(swap, m.gm)

	mt.gm = swap
	mt.count = len(mt.gm)

	return mt
}
*/

func (m *GenMap) getLen(nlen int) int {
	var res int = 1000 - (nlen % 1000)
	return res
}

func (m *GenMap) GetSize() int {
	if m.gm == nil {
		return 0
	}

	return len(m.gm)
}

func (m *GenMap) Insert(bkData string) {
	//thread-safe : prelock wait for unlock
	m.lock.Lock()
	defer m.lock.Unlock()

	sName := com.GetJsonValue(bkData, "blockSymbol")
	sNum := com.GetJsonValue(bkData, "blockNumber")
	if len(sName) <= 0 || len(sNum) <= 0 {
		return
	}

	skey := fmt.Sprintf("%s_%s", sName, sNum)

	m.gm[skey] = bkData
	m.count = len(m.gm)
}

func (m *GenMap) TxInsert(bkData string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	sHash := com.GetJsonValue(bkData, "hash")
	if len(sHash) <= 0 {
		return
	}

	m.gm[sHash] = bkData
	m.count = len(m.gm)
}

func (m *GenMap) GetDump() (*map[string]string, int) {
	if m.gm == nil {
		return nil, 0
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	mc := make(map[string]string, len(m.gm))

	mc = m.gm

	m.gm = make(map[string]string, 1000)
	m.count = 0

	return &mc, len(mc)
}
