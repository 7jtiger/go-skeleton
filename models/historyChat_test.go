package models

import "testing"

func TestFormatDMPairRoomID_orderIndependent(t *testing.T) {
	a := uint64(4033287471439576593)
	b := uint64(8697414060736839837)
	want := "4033287471439576593_8697414060736839837"

	if got := FormatDMPairRoomID(a, b); got != want {
		t.Errorf("FormatDMPairRoomID(a,b) = %q, want %q", got, want)
	}
	if got := FormatDMPairRoomID(b, a); got != want {
		t.Errorf("FormatDMPairRoomID(b,a) = %q, want %q", got, want)
	}
}

func TestFormatDMPairRoomID_sameUID(t *testing.T) {
	uid := uint64(12345)
	want := "12345_12345"
	if got := FormatDMPairRoomID(uid, uid); got != want {
		t.Errorf("FormatDMPairRoomID(same) = %q, want %q", got, want)
	}
}
