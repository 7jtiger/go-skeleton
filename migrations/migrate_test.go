package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io"
	"ms-gateway/conf"
	"os"
	"testing"
	"time"
)

func TestBundledStatements(t *testing.T) {
	for _, s := range steps {
		b, err := files.ReadFile(s.file)
		if err != nil {
			t.Fatal(err)
		}
		if s.statement >= len(statements(string(b))) {
			t.Fatalf("invalid migration statement: %s", s.file)
		}
	}
}

func TestProjectMigrationsMySQL(t *testing.T) {
	path := os.Getenv("MS_GATEWAY_DB_CONFIG")
	if path == "" {
		t.Skip("set MS_GATEWAY_DB_CONFIG to test migrations on an isolated schema")
	}
	dsn, err := conf.NewConfig(path).MySQLDSN("sdb")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("ms_gateway_migration_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + name + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	cfg.DBName = name
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		"CREATE TABLE story (idx INT PRIMARY KEY,qt_good INT DEFAULT 21,stat TINYINT DEFAULT 0,type INT DEFAULT 2,str_img JSON) ENGINE=InnoDB",
		"CREATE TABLE str_pcs (idx INT NOT NULL PRIMARY KEY,str_idx INT,pcs_uid BIGINT,qt_payed BIGINT) ENGINE=InnoDB",
		"CREATE TABLE str_cmt (idx INT PRIMARY KEY,str_idx INT,stat INT,at_create DATETIME) ENGINE=InnoDB",
		"INSERT INTO story (idx) VALUES (7)", "INSERT INTO str_pcs VALUES (10,7,20,100)",
	} {
		if _, err = db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO story (idx,type,str_img) VALUES (8,2,'{"1":{"type":"img","url":"a"},"2":{"type":"vdo","url":"b","thumb1":"c"}}'),(9,2,'{}')`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = Run(ctx, db, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND COLUMN_NAME='purchase_status'").Scan(&count); err != nil || count != 0 {
		t.Fatal("plan changed schema", err)
	}
	if err = Run(ctx, db, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	var statusCode int
	if err = db.QueryRow("SELECT stat FROM story WHERE idx=7").Scan(&statusCode); err != nil || statusCode != 0 {
		t.Fatalf("public status migration: %d %v", statusCode, err)
	}
	var likes int
	if err = db.QueryRow("SELECT qt_like FROM story WHERE idx=7").Scan(&likes); err != nil || likes != 21 {
		t.Fatalf("like count not preserved: %d %v", likes, err)
	}
	var mixed, text int
	if err = db.QueryRow("SELECT type FROM story WHERE idx=8").Scan(&mixed); err != nil || mixed != 3 {
		t.Fatalf("mixed migration: %d %v", mixed, err)
	}
	if err = db.QueryRow("SELECT type FROM story WHERE idx=9").Scan(&text); err != nil || text != 2 {
		t.Fatalf("text migration: %d %v", text, err)
	}
	var status, revision int
	if err = db.QueryRow("SELECT psc_stat FROM str_pcs WHERE idx=10").Scan(&status); err != nil || status != 0 {
		t.Fatalf("legacy payment auto-completed: %d %v", status, err)
	}
	if err = db.QueryRow("SELECT cntt_rvsn FROM story WHERE idx=7").Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("revision: %d %v", revision, err)
	}
	if _, err = db.Exec("UPDATE str_pcs SET psc_stat=1 WHERE idx=10"); err != nil {
		t.Fatal(err)
	}
	if err = Run(ctx, db, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT psc_stat FROM str_pcs WHERE idx=10").Scan(&status); err != nil || status != 1 {
		t.Fatalf("repeat reset entitlement: %d %v", status, err)
	}
	if err = db.QueryRow("SELECT stat FROM story WHERE idx=7").Scan(&statusCode); err != nil || statusCode != 0 {
		t.Fatalf("repeat remapped status: %d %v", statusCode, err)
	}
	if _, err = db.Exec("INSERT INTO str_pcs (str_idx,pcs_uid) VALUES (7,21)"); err != nil {
		t.Fatal("auto increment not working", err)
	}
	t.Run("restore_previous_mapping_once", func(t *testing.T) {
		for _, q := range []string{
			"DELETE FROM gateway_schema_migrations WHERE name='20261029_story_status_restore'",
			"INSERT INTO gateway_schema_migrations (name) VALUES ('20261028_story_status')",
			"UPDATE story SET stat=CASE idx WHEN 7 THEN 1 WHEN 8 THEN 2 WHEN 9 THEN 3 END",
		} {
			if _, err := db.Exec(q); err != nil {
				t.Fatal(err)
			}
		}
		for repeat := 0; repeat < 2; repeat++ {
			if err := Run(ctx, db, true, io.Discard); err != nil {
				t.Fatal(err)
			}
			for id, want := range map[int]int{7: 0, 8: 1, 9: 2} {
				var got int
				if err := db.QueryRow("SELECT stat FROM story WHERE idx=?", id).Scan(&got); err != nil || got != want {
					t.Fatalf("id=%d got=%d want=%d err=%v", id, got, want, err)
				}
			}
		}
	})

}
