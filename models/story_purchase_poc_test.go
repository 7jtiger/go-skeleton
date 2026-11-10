package models

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"ms-gateway/conf"
)

func TestStoryPurchasePOCMySQL(t *testing.T) {
	path := os.Getenv("MS_GATEWAY_DB_CONFIG")
	if path == "" {
		t.Skip("set MS_GATEWAY_DB_CONFIG for isolated temporary-table verification")
	}
	cfg := conf.NewConfig(path)
	dsn, err := cfg.MySQLDSN("adb")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	// Verify the deployed block schema without reading user records.
	var columns int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='user_block' AND COLUMN_NAME='tid'").Scan(&columns); err != nil || columns != 1 {
		t.Fatalf("expected user_block.tid: %d %v", columns, err)
	}
	for _, q := range []string{
		"CREATE TEMPORARY TABLE user_block (uid BIGINT UNSIGNED,tid BIGINT UNSIGNED,reason VARCHAR(128),stat INT,at_create DATETIME,at_update DATETIME,UNIQUE KEY(uid,tid))",
		"CREATE TEMPORARY TABLE story (idx INT PRIMARY KEY,uid BIGINT UNSIGNED,stat INT)",
		"CREATE TEMPORARY TABLE str_pcs (idx INT AUTO_INCREMENT PRIMARY KEY,str_idx INT,tid BIGINT UNSIGNED,pcs_uid BIGINT UNSIGNED,at_create DATETIME,at_update DATETIME,qt_payed INT,psc_stat INT)",
		"INSERT INTO story VALUES (1,20,2)",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	a := &AccountDB{conndb: db}
	s := &StoryDB{conndb: db}
	if _, err := db.Exec("INSERT INTO story VALUES (20,20,0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BuyStory(10, 20); !errors.Is(err, ErrStoryPurchaseNotPaid) {
		t.Fatalf("public story must not be purchased: %v", err)
	}
	if _, err := a.SetBlockUser(20, 10, "test"); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]uint64{{10, 20}, {20, 10}} {
		blocked, err := a.IsBlockedPair(pair[0], pair[1])
		if err != nil || !blocked {
			t.Fatalf("bidirectional block: %v %v", blocked, err)
		}
	}
	if _, err := a.SetUnblock(20, 10); err != nil {
		t.Fatal(err)
	}
	blocked, err := a.IsBlockedPair(10, 20)
	if err != nil || blocked {
		t.Fatalf("unblocked: %v %v", blocked, err)
	}
	for i := 0; i < 2; i++ {
		status, err := s.BuyStory(10, 1)
		if err != nil || status != 1 {
			t.Fatalf("POC purchase: %d %v", status, err)
		}
	}
	var count, paid, status int
	if err := db.QueryRow("SELECT COUNT(*),MAX(qt_payed),MAX(psc_stat) FROM str_pcs WHERE str_idx=1 AND pcs_uid=10").Scan(&count, &paid, &status); err != nil || count != 1 || paid != 0 || status != 1 {
		t.Fatalf("purchase record: count=%d paid=%d status=%d err=%v", count, paid, status, err)
	}
	entitlement, err := s.IsPcsPaid(1, 10)
	if err != nil || entitlement != 1 {
		t.Fatalf("purchase entitlement: %d %v", entitlement, err)
	}
}
