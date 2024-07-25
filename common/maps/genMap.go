package maps

import (
	"fmt"
	"sync"

	com "basesk/common"
)

type UMap struct {
	um    map[string]string
	count int
	lock  *sync.Mutex
}

func NewMap() *UMap {
	return &UMap{
		um:    make(map[string]string, 200),
		lock:  &sync.Mutex{},
		count: 0,
	}
}

func (m *UMap) cleanMap() *UMap {
	m.um = nil
	m.count = 0
	return NewMap()
}

/*
func (m *GenMap) buffering(mt *GenMap, nlen int) *GenMap {
	var swap map[string]string
	swap = mt.um
	mt.um = nil
	mt.um = make(map[string]string, mt.count+1000)
	//copy(swap, m.um)

	mt.um = swap
	mt.count = len(mt.um)

	return mt
}
*/

func (m *UMap) getLen(nlen int) int {
	var res int = 1000 - (nlen % 1000)
	return res
}

func (m *UMap) MapSize() int {
	if m.um == nil {
		return 0
	}

	return len(m.um)
}

func (m *UMap) Join(bkData string) {
	//thread-safe : prelock wait for unlock
	m.lock.Lock()
	defer m.lock.Unlock()

	sName := com.GetJsonValue(bkData, "blockSymbol")
	sNum := com.GetJsonValue(bkData, "blockNumber")
	if len(sName) <= 0 || len(sNum) <= 0 {
		return
	}

	skey := fmt.Sprintf("%s_%s", sName, sNum)

	m.um[skey] = bkData
	m.count = len(m.um)
}

func (m *UMap) InJoin(bkData string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	sHash := com.GetJsonValue(bkData, "hash")
	if len(sHash) <= 0 {
		return
	}

	m.um[sHash] = bkData
	m.count = len(m.um)
}

func (m *UMap) MapSwap() (*map[string]string, int) {
	if m.um == nil {
		return nil, 0
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	mc := make(map[string]string, len(m.um))

	mc = m.um

	m.um = make(map[string]string, 1000)
	m.count = 0

	return &mc, len(mc)
}
