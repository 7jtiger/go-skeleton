package protocol

import "testing"

func TestStoryStatusCodes(t *testing.T) {
	for input, want := range map[string]int{"pub": 0, "flw": 1, "pay": 2, "prv": 3, "all": 6, "0": 0, "1": 1, "2": 2, "3": 3, "6": 6, "ALL": 6} {
		got, err := GetStatCode(input)
		if err != nil || got != want {
			t.Fatalf("%s: got %d %v, want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"lmt", "del", "rsv", "4", "5", "limit", ""} {
		if _, err := GetStatCode(input); err == nil {
			t.Fatalf("accepted unsupported filter %q", input)
		}
	}
	for stat := 0; stat <= 5; stat++ {
		if !IsValidStoryStat(stat) {
			t.Fatalf("invalid stored status %d", stat)
		}
	}
	if IsValidStoryStat(6) {
		t.Fatal("ALL must not be persisted")
	}
}
