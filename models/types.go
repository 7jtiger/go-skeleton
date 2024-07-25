package models

import (
	"time"

	"go-skeleton/protocol"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 체결 db document read 타입
type ConcludedDocument struct {
	ID          primitive.ObjectID `json:"_id"`
	Time        int64              `json:"time"`
	Seller      common.Address     `json:"seller"`
	Token       common.Address     `json:"token"`
	Buyer       common.Address     `json:"buyer"`
	Price       decimal.Decimal    `json:"price"`
	AmountOrTid decimal.Decimal    `json:"amountOrTid"`
}

// /// auth type ////////////////////////////////////////////////////////////////////////////////////
type VerifyTokenResp struct {
	*protocol.RespHeader
	Address   common.Address `json:"address"`
	UserID    string         `json:"userId"`
	ExpiredAt time.Time      `json:"expiredAt"`
}

type FindAddressResp struct {
	*protocol.RespHeader
	Address common.Address `json:"address"`
}

///// to-be deleted-////////////////////////////////////////////////////////////////////////////////////////////
