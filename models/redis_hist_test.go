package models

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestRedisDB(t *testing.T) *RedisDB {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	return &RedisDB{client: client, ctx: ctx}
}

func TestGetChatMessagesByCursor_fromIndex(t *testing.T) {
	rdb := newTestRedisDB(t)
	ctx := rdb.ctx
	roomID := "test-hist-cutoff-room"
	listKey := chatMsgListKey(roomID)
	_ = rdb.client.Del(ctx, listKey)

	msgs := []ChatMessageData{
		{ID: "m1", Content: "old1", Timestamp: time.Now()},
		{ID: "m2", Content: "old2", Timestamp: time.Now()},
		{ID: "m3", Content: "new1", Timestamp: time.Now()},
	}
	for _, m := range msgs {
		raw, _ := json.Marshal(m)
		if err := rdb.client.RPush(ctx, listKey, raw).Err(); err != nil {
			t.Fatal(err)
		}
	}
	defer rdb.client.Del(ctx, listKey)

	// fromIndex=2 → m3만 보임
	total, got, next, hasMore, err := rdb.GetChatMessagesByCursor(roomID, "", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("visible total want 1 got %d", total)
	}
	if len(got) != 1 || got[0].ID != "m3" {
		t.Fatalf("messages want [m3] got %+v", got)
	}
	if hasMore {
		t.Fatal("hasMore should be false")
	}
	if next != "m3" {
		t.Fatalf("nextCursor want m3 got %s", next)
	}

	// fromIndex=0 → 전체 3건
	totalAll, gotAll, _, _, err := rdb.GetChatMessagesByCursor(roomID, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if totalAll != 3 || len(gotAll) != 3 {
		t.Fatalf("all messages want 3 got total=%d len=%d", totalAll, len(gotAll))
	}
}
