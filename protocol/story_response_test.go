package protocol

import (
	"encoding/json"
	"testing"
)

func TestStoryResponseNumericFlags(t *testing.T) {
	for _, flag := range []int{0, 1} {
		for _, response := range []interface{}{
			StoryDetailResp{IsLocked: flag, GoodByMe: flag},
			StoryListResp{IsLocked: flag, GoodByMe: flag},
		} {
			raw, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"is_locked", "good_byme"} {
				if fields[key] != float64(flag) {
					t.Fatalf("%T: %s must be numeric %d: %s", response, key, flag, raw)
				}
			}
			for _, key := range []string{"like_me", "liked_by_me"} {
				if _, exists := fields[key]; exists {
					t.Fatalf("unexpected legacy field %s: %s", key, raw)
				}
			}
		}
	}
}

func TestStoryResponseAgeOnly(t *testing.T) {
	for _, response := range []interface{}{
		StoryDetailResp{Age: ""},
		StoryListResp{Age: ""},
	} {
		raw, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]interface{}
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["age"] != "" {
			t.Fatalf("missing response age: %s", raw)
		}
		if _, exists := fields["birth"]; exists {
			t.Fatalf("birth must not be exposed: %s", raw)
		}
	}
}
