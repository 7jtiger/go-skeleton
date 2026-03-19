package utils

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/big"
	"net"

	"github.com/google/uuid"
)

func GenUuid() string {
	uuid := uuid.New()

	uuid4 := hex.EncodeToString(uuid[:])
	return uuid4
}

func GenRandomUID() (uint64, error) {
	// 64비트 (8바이트) 난수 생성
	bytes := make([]byte, 8)
	_, err := crand.Read(bytes)
	if err != nil {
		return 0, err
	}

	// 바이트를 big.Int로 변환
	uid := new(big.Int).SetBytes(bytes)

	// MySQL BIGINT UNSIGNED 범위 제한 (2^63-1)
	maxBigInt := new(big.Int).SetUint64(1<<63 - 1) // 9223372036854775807
	uid.Mod(uid, maxBigInt)

	return uid.Uint64(), nil
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	return addrs[0].(*net.IPNet).IP.String()
}
