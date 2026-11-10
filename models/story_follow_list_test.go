package models

import (
	"database/sql"
	"os"
	"testing"

	"ms-gateway/conf"
)

func TestStoryFollowListMySQL(t *testing.T) {
	path := os.Getenv("MS_GATEWAY_DB_CONFIG")
	if path == "" {
		t.Skip("set MS_GATEWAY_DB_CONFIG for temporary-table tests")
	}
	cfg := conf.NewConfig(path)
	dsn, err := cfg.MySQLDSN("sdb")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	s := &StoryDB{conndb: db, cfg: cfg}
	table, err := s.accountUserInfoTable()
	if err != nil {
		t.Fatal(err)
	}
	// Validate SELECT access and the real column names using the sdb connection.
	rows, err := db.Query("SELECT uid,nick,thmb_pic FROM " + table + " LIMIT 0")
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	for _, q := range []string{
		"CREATE TEMPORARY TABLE user_follow (uid BIGINT UNSIGNED,followee_uid BIGINT UNSIGNED,stat INT,at_update DATETIME)",
		"CREATE TEMPORARY TABLE " + table + " (uid BIGINT UNSIGNED PRIMARY KEY,nick VARCHAR(30),thmb_pic VARCHAR(256))",
		"INSERT INTO " + table + " VALUES (10,'follower','https://example.test/thumb'),(30,'inactive','hidden'),(40,NULL,NULL)",
		"INSERT INTO user_follow VALUES (10,20,1,'2026-10-29 12:00:00'),(30,20,0,'2026-10-29 11:00:00'),(20,40,1,'2026-10-29 10:00:00'),(50,20,1,'2026-10-29 09:00:00')",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	followers, total, err := s.GetFollowerList(20, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(*followers) != 2 {
		t.Fatalf("followers total=%d rows=%d", total, len(*followers))
	}
	if (*followers)[0].Uid != 10 || (*followers)[0].Nick != "follower" || (*followers)[0].ThumbPic != "https://example.test/thumb" {
		t.Fatalf("wrong follower metadata: %+v", (*followers)[0])
	}
	if (*followers)[1].Uid != 50 || (*followers)[1].Nick != "" || (*followers)[1].ThumbPic != "" {
		t.Fatal("missing user metadata must preserve relation")
	}
	following, total, err := s.GetFollowingList(20, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(*following) != 1 || (*following)[0].Uid != 40 || (*following)[0].Nick != "" || (*following)[0].ThumbPic != "" {
		t.Fatalf("following: %+v total=%d", following, total)
	}
	followers, total, err = s.GetFollowerList(20, 2, 1)
	if err != nil || total != 2 || len(*followers) != 1 || (*followers)[0].Uid != 50 {
		t.Fatalf("pagination: %+v total=%d err=%v", followers, total, err)
	}
	following, total, err = s.GetFollowingList(999, 1, 10)
	if err != nil || total != 0 || len(*following) != 0 {
		t.Fatalf("empty list: %+v total=%d err=%v", following, total, err)
	}
}
