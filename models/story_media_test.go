package models

import (
	"encoding/json"
	"strconv"
	"testing"

	ptc "ms-gateway/protocol"
)

func TestParseStoryMedia_Unified(t *testing.T) {
	raw := `{
	  "1": {"type": "img", "url": "https://imagedelivery.net/.../pic1/public"},
	  "2": {"type": "vdo", "url": "https://customer-xxx.cloudflarestream.com/abc/manifest/video.m3u8", "thumb": "https://imagedelivery.net/.../thumb2/public"},
	  "3": {"type": "img", "url": "https://imagedelivery.net/.../pic3/public"}
	}`
	media, err := ParseStoryMedia(raw)
	if err != nil {
		t.Fatalf("ParseStoryMedia: %v", err)
	}
	if media["1"].Type != ptc.StoryMediaTypeImg || media["2"].Thumb == "" || media["3"].URL == "" {
		t.Fatalf("unexpected media: %+v", media)
	}
	if MediaMType(media) != 2 {
		t.Fatalf("expected mtype=2, got %d", MediaMType(media))
	}
}

func TestParseStoryMedia_RejectsLegacyFlat(t *testing.T) {
	raw := `{"1":"https://v.mp4","thumb1":"https://t/public"}`
	if _, err := ParseStoryMedia(raw); err == nil {
		t.Fatal("expected error for legacy flat format")
	}
}

func TestGetFirstPicUrl_VdoUsesThumb(t *testing.T) {
	raw := `{"1":{"type":"vdo","url":"https://v.m3u8","thumb":"https://thumb/public"}}`
	url, kind, err := GetFirstPicUrl(raw)
	if err != nil {
		t.Fatalf("GetFirstPicUrl: %v", err)
	}
	if kind != 1 || url != "https://thumb/public" {
		t.Fatalf("want thumb url kind=1, got %q kind=%d", url, kind)
	}
}

func TestBuildStrImgForDB(t *testing.T) {
	media := ptc.StoryStrImg{
		"3": {Type: ptc.StoryMediaTypeImg, URL: "https://c/public", Thumb: "ignored"},
		"1": {Type: ptc.StoryMediaTypeVdo, URL: "https://v.m3u8", Thumb: "https://t/public"},
	}
	raw, mtype, err := BuildStrImgForDB(media)
	if err != nil {
		t.Fatalf("BuildStrImgForDB: %v", err)
	}
	if mtype != 2 {
		t.Fatalf("expected mtype=2, got %d", mtype)
	}
	out, err := ParseStoryMedia(string(raw))
	if err != nil {
		t.Fatalf("ParseStoryMedia: %v", err)
	}
	if len(out) != 2 || out["1"].Type != ptc.StoryMediaTypeVdo || out["2"].Type != ptc.StoryMediaTypeImg {
		t.Fatalf("canonical slots: %+v", out)
	}
	if out["2"].Thumb != "" {
		t.Fatalf("img slot must not keep thumb: %+v", out["2"])
	}

	// DB에 저장되는 JSON이 스펙 키를 갖는지 확인
	var generic map[string]map[string]string
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal generic: %v", err)
	}
	if _, ok := generic["1"]["thumb"]; !ok {
		t.Fatal("vdo slot must include thumb")
	}
	if _, ok := generic["2"]["thumb"]; ok {
		t.Fatal("img slot must omit thumb")
	}
}

func TestValidateStoryMedia_MaxSlots(t *testing.T) {
	media := ptc.StoryStrImg{}
	for i := 1; i <= MaxStoryMediaSlots+1; i++ {
		media[strconv.Itoa(i)] = ptc.StoryMediaItem{Type: ptc.StoryMediaTypeImg, URL: "https://a/" + strconv.Itoa(i)}
	}
	if err := ValidateStoryMedia(media); err == nil {
		t.Fatal("expected error when media exceeds max slots")
	}

	okMedia := ptc.StoryStrImg{}
	for i := 1; i <= MaxStoryMediaSlots; i++ {
		okMedia[strconv.Itoa(i)] = ptc.StoryMediaItem{Type: ptc.StoryMediaTypeImg, URL: "https://a/" + strconv.Itoa(i)}
	}
	if err := ValidateStoryMedia(okMedia); err != nil {
		t.Fatalf("5 slots should pass: %v", err)
	}
}

func TestReindexStoryMediaAfterDelete(t *testing.T) {
	media := ptc.StoryStrImg{
		"1": {Type: ptc.StoryMediaTypeImg, URL: "a"},
		"2": {Type: ptc.StoryMediaTypeVdo, URL: "b", Thumb: "tb"},
		"3": {Type: ptc.StoryMediaTypeImg, URL: "c"},
	}
	out := ReindexStoryMediaAfterDelete(media, "2")
	if len(out) != 2 || out["1"].URL != "a" || out["2"].URL != "c" {
		t.Fatalf("reindex failed: %+v", out)
	}
}
