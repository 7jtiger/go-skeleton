package models

import (
	ptl "ms-gateway/protocol"
	"testing"
)

func TestStoryStatusVisibility(t *testing.T) {
	for _, tc := range []struct {
		stat    int
		viewer  uint64
		paid    int
		visible bool
		locked  int
	}{
		{0, 20, 0, true, 0}, {1, 20, 0, true, 0}, {1, 10, 0, true, 0},
		{2, 20, 0, true, 1}, {2, 20, 1, true, 0}, {2, 10, 0, true, 0},
		{3, 20, 1, false, 0}, {3, 10, 0, true, 0}, {4, 10, 0, false, 0}, {6, 10, 0, false, 0},
	} {
		story := ptl.StoryDetailResp{Uid: 10, Stat: tc.stat, IsPaid: tc.paid, IsFollowed: 1, Body: "private content"}
		if got := storyVisible(&story, tc.viewer); got != tc.visible {
			t.Fatalf("visibility stat=%d viewer=%d: %v", tc.stat, tc.viewer, got)
		}
		if tc.visible {
			redactStory(&story, tc.viewer)
			if story.IsLocked != tc.locked {
				t.Fatalf("lock stat=%d: %d", tc.stat, story.IsLocked)
			}
			if tc.locked == 1 && story.Body != "" {
				t.Fatal("locked content exposed")
			}
		}
	}
}

func TestFLWRequiresActiveFollow(t *testing.T) {
	for _, follow := range []int{0, 1} {
		story := ptl.StoryDetailResp{Uid: 10, Stat: ptl.StoryStatFollower, IsFollowed: follow, IsPaid: 1, Body: "content"}
		redactStory(&story, 20)
		if story.IsLocked != 1-follow {
			t.Fatalf("follow=%d locked=%d", follow, story.IsLocked)
		}
		if follow == 0 && story.Body != "" {
			t.Fatal("non-follower received content")
		}
	}
}
