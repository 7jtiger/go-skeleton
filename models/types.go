package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// 체결 db document read 타입
// type ConcludedDocument struct {
// 	ID          primitive.ObjectID `json:"_id"`
// 	Time        int64              `json:"time"`
// 	Seller      common.Address     `json:"seller"`
// 	Token       common.Address     `json:"token"`
// 	Buyer       common.Address     `json:"buyer"`
// 	Price       decimal.Decimal    `json:"price"`
// 	AmountOrTid decimal.Decimal    `json:"amountOrTid"`
// }

// DMRoomRow chat_his 테이블 로우 (API 응답 roomId=idx, DB room_id=minUid_maxUid)
type DMRoomRow struct {
	Idx       int64     `json:"idx"`
	UID       uint64    `json:"uid"`
	TID       uint64    `json:"tid"`
	RoomID    string    `json:"room_id"` // chat_his.room_id = FormatDMPairRoomID(uid, tid)
	STChat    int       `json:"st_chat"`
	AtCrtCHAT time.Time `json:"at_crtchat"`
	AtUpdate  time.Time `json:"at_update"`
	PaidPoint float64   `json:"paid_point"`
}

// DM 채팅방 참여 상태 (chat_his.st_chat)
const (
	STChatBothLeft = 0
	STChatBothIn   = 1
	STChatUIDLeft  = 2
	STChatTIDLeft  = 3
)

const PartnerLeftMsg = "상대방이 대화를 종료하였습니다."

const dmRoomSelectCols = "idx, uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point"

// FormatDMPairRoomID DM chat_his.room_id (작은 uid + "_" + 큰 uid).
// 예: 4033287471439576593_8697414060736839837
func FormatDMPairRoomID(uid, tid uint64) string {
	lo, hi := uid, tid
	if lo > hi {
		lo, hi = hi, lo
	}
	return strconv.FormatUint(lo, 10) + "_" + strconv.FormatUint(hi, 10)
}

// IsUserInDMRoom viewer가 목록/방에 노출될 수 있는지
func IsUserInDMRoom(st int, uid, tid, viewer uint64) bool {
	switch st {
	case STChatBothLeft:
		return false
	case STChatBothIn:
		return viewer == uid || viewer == tid
	case STChatUIDLeft:
		return viewer == tid
	case STChatTIDLeft:
		return viewer == uid
	default:
		return false
	}
}

// IsPartnerLeft viewer 기준 상대방이 나간 상태인지
func IsPartnerLeft(st int, uid, tid, viewer uint64) bool {
	if viewer == uid {
		return st == STChatTIDLeft
	}
	if viewer == tid {
		return st == STChatUIDLeft
	}
	return false
}

// PartnerUIDOfRoom viewer의 상대 UID
func PartnerUIDOfRoom(room *DMRoomRow, viewer uint64) uint64 {
	if viewer == room.UID {
		return room.TID
	}
	return room.UID
}

func nextSTChatOnLeave(st int, leaverUID uint64, room *DMRoomRow) (int, error) {
	if leaverUID == room.UID {
		switch st {
		case STChatBothIn:
			return STChatUIDLeft, nil
		case STChatTIDLeft:
			return STChatBothLeft, nil
		case STChatUIDLeft, STChatBothLeft:
			return st, nil
		}
	} else if leaverUID == room.TID {
		switch st {
		case STChatBothIn:
			return STChatTIDLeft, nil
		case STChatUIDLeft:
			return STChatBothLeft, nil
		case STChatTIDLeft, STChatBothLeft:
			return st, nil
		}
	}
	return 0, fmt.Errorf("user %d is not a participant of room %d", leaverUID, room.Idx)
}

func nextSTChatOnRejoin(st int, joinerUID uint64, room *DMRoomRow) int {
	if joinerUID == room.UID {
		switch st {
		case STChatBothLeft:
			return STChatTIDLeft
		case STChatUIDLeft:
			return STChatBothIn
		default:
			return st
		}
	}
	if joinerUID == room.TID {
		switch st {
		case STChatBothLeft:
			return STChatUIDLeft
		case STChatTIDLeft:
			return STChatBothIn
		default:
			return st
		}
	}
	return st
}

type dmRoomScanner interface {
	Scan(dest ...any) error
}

func scanDMRoomFields(sc dmRoomScanner) (*DMRoomRow, error) {
	var dm DMRoomRow
	var roomID sql.NullString
	var atCrt, atUpd sql.NullTime
	var paid sql.NullFloat64
	if err := sc.Scan(&dm.Idx, &dm.UID, &dm.TID, &roomID, &dm.STChat, &atCrt, &atUpd, &paid); err != nil {
		return nil, err
	}
	if roomID.Valid {
		dm.RoomID = roomID.String
	}
	if atCrt.Valid {
		dm.AtCrtCHAT = atCrt.Time
	}
	if atUpd.Valid {
		dm.AtUpdate = atUpd.Time
	}
	if paid.Valid {
		dm.PaidPoint = paid.Float64
	}
	return &dm, nil
}

func scanDMRoomRow(row *sql.Row) (*DMRoomRow, error) {
	return scanDMRoomFields(row)
}

// /// auth type ////////////////////////////////////////////////////////////////////////////////////

///// to-be deleted-////////////////////////////////////////////////////////////////////////////////////////////
