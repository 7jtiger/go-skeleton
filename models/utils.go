package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	ptc "ms-gateway/protocol"
)

// MaxStoryMediaSlots is the max number of media items in story.str_img.
const MaxStoryMediaSlots = 5

// ParseStoryMedia parses story.str_img JSON as StoryStrImg only.
// Expected shape:
//
//	{
//	  "1": {"type":"img","url":"https://imagedelivery.net/.../pic1/public"},
//	  "2": {"type":"vdo","url":"https://customer-xxx.cloudflarestream.com/.../manifest/video.m3u8","thumb1":"https://imagedelivery.net/.../thumb2/public"},
//	  "3": {"type":"img","url":"https://imagedelivery.net/.../pic3/public"}
//	}

/*
func ParseStoryMedia(strImg string) (ptc.StoryStrImg, error) {
	strImg = strings.TrimSpace(strImg)
	if strImg == "" || strImg == "null" {
		return nil, fmt.Errorf("empty str_img")
	}

	var media ptc.StoryStrImg
	if err := json.Unmarshal([]byte(strImg), &media); err != nil {
		return nil, fmt.Errorf("invalid media JSON: %w", err)
	}
	if err := ValidateStoryMedia(media); err != nil {
		return nil, err
	}
	return media, nil
} */
/*
// LoadStoryStrImg loads DB str_img bytes into StoryStrImg.
// Empty object "{}" is allowed (all slots deleted).
func LoadStoryStrImg(raw json.RawMessage) (ptc.StoryStrImg, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == "{}" {
		return ptc.StoryStrImg{}, nil
	}
	// return ParseStoryMedia(s)
	var media ptc.StoryStrImg
	if err := json.Unmarshal([]byte(s), &media); err != nil {
		return nil, fmt.Errorf("invalid media JSON: %w", err)
	}
	if err := ValidateStoryMedia(media); err != nil {
		return nil, err
	}
	return media, nil

}
*/

// BuildStrImgForDB builds canonical story.str_img JSON for DB storage.
// Slots are renumbered 1..N; img keeps type+url, vdo keeps type+url+thumb1.
func BuildStrImgForDB(media ptc.StoryStrImg) (json.RawMessage, int, error) {
	if _, err := ValidateStoryMedia(media); err != nil {
		return nil, 0, err
	}

	nums := make([]int, 0, len(media))
	for k := range media {
		n, err := strconv.Atoi(k)
		if err != nil || n <= 0 {
			return nil, 0, fmt.Errorf("media key %q must be positive numeric slot", k)
		}
		nums = append(nums, n)
	}
	sort.Ints(nums)

	canonical := make(ptc.StoryStrImg, len(nums))
	for i, n := range nums {
		src := media[strconv.Itoa(n)]
		item := ptc.StoryMediaItem{
			Type: src.Type,
			URL:  src.URL,
		}
		if src.Type == ptc.StoryMediaTypeVdo {
			item.Thumb = src.Thumb
		}
		canonical[strconv.Itoa(i+1)] = item
	}

	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal str_img: %w", err)
	}
	return raw, MediaMType(canonical), nil
}

// MediaMType returns story.type: 0=img only, 1=vdo only, 2=txt, 3=mixed.
func MediaMType(media ptc.StoryStrImg) int {
	if len(media) == 0 {
		return 2
	}
	hasImg, hasVdo := false, false
	for _, item := range media {
		switch item.Type {
		case ptc.StoryMediaTypeImg:
			hasImg = true
		case ptc.StoryMediaTypeVdo:
			hasVdo = true
		}
	}
	switch {
	case hasImg && hasVdo:
		return 3
	case hasVdo:
		return 1
	default:
		return 0
	}
}

// ValidateStoryMedia checks media slots; an empty map denotes a text story.
// Returns: 0 (img only), 1 (vdo only), 2 (text), 3 (mixed), and error if invalid
func ValidateStoryMedia(media ptc.StoryStrImg) (int, error) {
	if len(media) == 0 {
		return 2, nil
	}
	if len(media) > MaxStoryMediaSlots {
		return 0, fmt.Errorf("media allows at most %d items", MaxStoryMediaSlots)
	}
	hasImg, hasVdo := false, false
	for k, item := range media {
		n, err := strconv.Atoi(k)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("media key %q must be positive numeric slot", k)
		}
		if n > MaxStoryMediaSlots {
			return 0, fmt.Errorf("media key %q exceeds max slot %d", k, MaxStoryMediaSlots)
		}
		if item.URL == "" {
			return 0, fmt.Errorf("media[%s].url is required", k)
		}
		switch item.Type {
		case ptc.StoryMediaTypeImg:
			hasImg = true
		case ptc.StoryMediaTypeVdo:
			hasVdo = true
			if item.Thumb == "" {
				return 0, fmt.Errorf("media[%s].thumb1 is required for vdo", k)
			}
		default:
			return 0, fmt.Errorf("media[%s].type must be img or vdo", k)
		}

	}

	switch {
	case hasImg && hasVdo:
		return 3, nil
	case hasVdo:
		return 1, nil
	default:
		return 0, nil
	}
}

// GetFirstPicUrl returns representative list URL and type (0=img, 1=vdo).
// Call only with media already filtered by viewer access. This parser does not authorize.
func GetFirstPicUrl(strImg string) (*ptc.StoryMediaItem, int, error) {
	// media, err := ParseStoryMedia(strImg)
	// var StoryMediaItem
	var media map[string]ptc.StoryMediaItem
	if err := json.Unmarshal([]byte(strImg), &media); err != nil {
		return nil, 0, fmt.Errorf("invalid media JSON: %w", err)
	}

	// media, err := LoadStoryStrImg(json.RawMessage(strImg))
	// if err != nil {
	// 	return nil, err
	// }

	minNum := 0
	hasNum := false
	for k := range media {
		n, convErr := strconv.Atoi(k)
		if convErr != nil || n <= 0 {
			continue
		}
		if !hasNum || n < minNum {
			minNum = n
			hasNum = true
		}
	}
	if !hasNum {
		return nil, 0, fmt.Errorf("no media slot found")
	}

	var item ptc.StoryMediaItem
	var ok bool
	if item, ok = media[strconv.Itoa(minNum)]; ok {
		return &item, len(media), nil
	}

	return nil, 0, fmt.Errorf("no media slot found")
}

// ReindexStoryMediaAfterDelete removes slot key and renumbers remaining 1..N.
func ReIdxAfterDel(media ptc.StoryStrImg, delKey string) ptc.StoryStrImg {
	delete(media, delKey)

	nums := make([]int, 0, len(media))
	for k := range media {
		n, err := strconv.Atoi(k)
		if err != nil || n <= 0 {
			continue
		}
		nums = append(nums, n)
	}
	sort.Ints(nums)

	out := make(ptc.StoryStrImg, len(nums))
	for i, n := range nums {
		out[strconv.Itoa(i+1)] = media[strconv.Itoa(n)]
	}
	return out
}

// Owner-scoped profile summaries use the same visibility and content policy as detail lists.
// calcStoryAge converts profile birth and age to the response age.
// birth is YYYY-MM-DD; prfAge is an offset like "-2", "2", "+3", or "0".
func calcStoryAge(birth, prfAge string) string {
	if birth == "" {
		return ""
	}
	born, err := time.Parse("2006-01-02", birth)
	if err != nil {
		return ""
	}

	now := time.Now()
	age := now.Year() - born.Year()
	if now.Month() < born.Month() || (now.Month() == born.Month() && now.Day() < born.Day()) {
		age--
	}

	adj := 0
	if prfAge != "" {
		n, err := strconv.Atoi(prfAge)
		if err != nil {
			return ""
		}
		adj = n
	}

	return strconv.Itoa(age + adj)
}
