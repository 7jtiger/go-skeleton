// Package migrations applies only the gateway's story schema changes.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"strings"
)

//go:embed *.sql
var files embed.FS

type step struct {
	file, kind, table, name, expected string
	statement                         int
}

const revisionComment = "콘텐츠 변경 버전: 본문·상태·미디어 수정 시 증가, 같은 초 변경의 캐시 검증용. 0=미확인, 1~254=버전, 255=포화 시 캐시 사용 안 함"

const storyStatusComment = "stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)"

var steps = []step{
	{"20261028_story_qt_like.sql", "like_column", "story", "qt_like", "", 0},
	{"20261027_story_purchase_status.sql", "legacy_purchase", "str_pcs", "purchase_status", "tinyint|NO|0|", 0},
	{"20261027_story_content_revision.sql", "legacy_revision", "story", "content_revision", "bigint unsigned|NO|1|", 0},
	{"20261028_story_cntt_rvsn.sql", "revision", "story", "cntt_rvsn", "tinyint unsigned|NO|1|", 0},
	{"20261027_story_view_history.sql", "table", "story_view_history", "", "", 0},
	{"20261027_story_batch_indexes.sql", "index", "str_cmt", "idx_str_cmt_story_stat_time", "str_idx,stat,at_create,idx", 0},
	{"20261027_story_batch_indexes.sql", "index", "str_pcs", "idx_str_pcs_buyer_status_story", "pcs_uid,purchase_status,str_idx", 1},
	{"20261028_story_purchase_request.sql", "auto", "str_pcs", "idx", "", 0},
	{"20261028_story_purchase_request.sql", "index", "str_pcs", "idx_str_pcs_story_buyer", "str_idx,pcs_uid,idx", 1},
	{"20261028_story_psc_stat.sql", "column", "str_pcs", "psc_stat", "tinyint|NO|0|", 0},
	{"20261028_story_media_type.sql", "media_type", "story", "type", "", 0},
	{"20261028_story_media_type.sql", "media_type_comment", "story", "type", "", 1},
	{"20261029_story_status_restore.sql", "status_mapping", "story", "stat_mapping", "", 0},
	{"20261029_story_status_restore.sql", "status_comment", "story", "stat", "", 3},
}

func statements(data string) []string {
	var clean []string
	for _, line := range strings.Split(data, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			clean = append(clean, line)
		}
	}
	var out []string
	for _, s := range strings.Split(strings.Join(clean, "\n"), ";") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Run checks the complete plan before making changes. DDL auto-commits in MySQL;
// successful steps are detected by their actual schema on subsequent runs.
func Run(ctx context.Context, db *sql.DB, apply bool, out io.Writer) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if apply {
		var lock sql.NullInt64
		if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT('ms-gateway:',DATABASE(),':migrate'),10)").Scan(&lock); err != nil {
			return err
		}
		if !lock.Valid || lock.Int64 != 1 {
			return fmt.Errorf("could not acquire migration lock")
		}
		defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(CONCAT('ms-gateway:',DATABASE(),':migrate'))")
		if _, err = conn.ExecContext(ctx, "SET SESSION lock_wait_timeout=10"); err != nil {
			return err
		}
	}
	// All tables updated by the gateway's transactions must support rollback.
	for _, table := range []string{"story", "str_pcs", "str_cmt"} {
		var engine string
		if err = conn.QueryRowContext(ctx, "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table).Scan(&engine); err != nil {
			return fmt.Errorf("required table %s: %w", table, err)
		}
		if !strings.EqualFold(engine, "InnoDB") {
			return fmt.Errorf("%s must use InnoDB; found %s", table, engine)
		}
	}
	var pending []step
	for _, s := range steps {
		exists, err := s.check(ctx, conn)
		if err != nil {
			return err
		}
		if exists {
			fmt.Fprintf(out, "OK %s.%s\n", s.table, s.label())
		} else {
			fmt.Fprintf(out, "PENDING %s: %s.%s\n", s.file, s.table, s.label())
			pending = append(pending, s)
		}
	}
	if !apply {
		fmt.Fprintf(out, "Plan: %d pending steps (use -apply to execute)\n", len(pending))
		return nil
	}
	for _, s := range pending {
		data, err := files.ReadFile(s.file)
		if err != nil {
			return err
		}
		queries := statements(string(data))
		if s.statement >= len(queries) {
			return fmt.Errorf("missing statement in %s", s.file)
		}
		if s.kind == "status_mapping" {
			if _, err := conn.ExecContext(ctx, queries[0]); err != nil {
				return err
			}
			tx, err := conn.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			for _, query := range queries[1:3] {
				if _, err := tx.ExecContext(ctx, query); err != nil {
					tx.Rollback()
					return err
				}
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			fmt.Fprintf(out, "APPLIED story.stat_mapping\n")
			continue
		}
		selected := queries[s.statement : s.statement+1]
		if s.kind == "revision" {
			selected = queries
		}
		for _, query := range selected {
			if s.kind == "index" && s.table == "str_pcs" {
				var count int
				if err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='str_pcs' AND COLUMN_NAME='psc_stat'").Scan(&count); err != nil {
					return err
				}
				if count > 0 {
					query = strings.ReplaceAll(query, "purchase_status", "psc_stat")
				}
			}
			if _, err = conn.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("apply %s.%s: %w", s.table, s.label(), err)
			}
		}
		ok, err := s.check(ctx, conn)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("verification failed for %s.%s", s.table, s.label())
		}
		fmt.Fprintf(out, "APPLIED %s.%s\n", s.table, s.label())
	}
	fmt.Fprintf(out, "Schema verified: %d steps applied\n", len(pending))
	return nil
}
func (s step) label() string {
	if s.kind == "table" {
		return "table"
	}
	return s.name
}
func (s step) check(ctx context.Context, c *sql.Conn) (bool, error) {
	switch s.kind {
	case "status_mapping":
		var count int
		if err := c.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='gateway_schema_migrations'").Scan(&count); err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
		err := c.QueryRowContext(ctx, "SELECT COUNT(*) FROM gateway_schema_migrations WHERE name='20261029_story_status_restore'").Scan(&count)
		return count == 1, err
	case "status_comment":
		var comment string
		err := c.QueryRowContext(ctx, "SELECT COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='story' AND COLUMN_NAME='stat'").Scan(&comment)
		return comment == storyStatusComment, err

	case "like_column":
		var oldCount, newCount int
		err := c.QueryRowContext(ctx, `SELECT COALESCE(SUM(COLUMN_NAME='qt_good'),0),COALESCE(SUM(COLUMN_NAME='qt_like'),0) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='story'`).Scan(&oldCount, &newCount)
		if err != nil {
			return false, err
		}
		if oldCount+newCount != 1 {
			return false, fmt.Errorf("story must have exactly one of qt_good or qt_like before migration")
		}
		return newCount == 1, nil

	case "media_type_comment":
		var comment string
		err := c.QueryRowContext(ctx, "SELECT COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='story' AND COLUMN_NAME='type'").Scan(&comment)
		return comment == "0=img, 1=vdo, 2=txt, 3=mix, 4=rsv", err

	case "media_type":
		var count int
		err := c.QueryRowContext(ctx, `SELECT COUNT(*) FROM story WHERE type=2 AND JSON_SEARCH(str_img,'one','img',NULL,'$.*.type') IS NOT NULL AND JSON_SEARCH(str_img,'one','vdo',NULL,'$.*.type') IS NOT NULL`).Scan(&count)
		return count == 0, err

	case "legacy_purchase":
		var count int
		if err := c.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='str_pcs' AND COLUMN_NAME='psc_stat'").Scan(&count); err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
		s.kind = "column"
		return s.check(ctx, c)

	case "legacy_revision":
		var count int
		if err := c.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='story' AND COLUMN_NAME='cntt_rvsn'").Scan(&count); err != nil {
			return false, err
		}
		if count != 0 {
			return true, nil
		}
		s.kind = "column"
		return s.check(ctx, c)
	case "revision":
		s.kind = "column"
		ok, err := s.check(ctx, c)
		if err != nil || !ok {
			return ok, err
		}
		var comment string
		if err = c.QueryRowContext(ctx, "SELECT COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='story' AND COLUMN_NAME='cntt_rvsn'").Scan(&comment); err != nil {
			return false, err
		}
		if comment != revisionComment {
			return false, fmt.Errorf("unexpected story.cntt_rvsn comment")
		}
		return true, nil

	case "column", "auto":
		var typ, nullable, def, extra string
		err := c.QueryRowContext(ctx, `SELECT COLUMN_TYPE,IS_NULLABLE,COALESCE(COLUMN_DEFAULT,'NULL'),EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?`, s.table, s.name).Scan(&typ, &nullable, &def, &extra)
		if err == sql.ErrNoRows {
			if s.kind == "auto" {
				return false, fmt.Errorf("missing %s.idx", s.table)
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if s.kind == "auto" {
			if typ != "int" || nullable != "NO" {
				return false, fmt.Errorf("unexpected %s.idx definition: %s %s", s.table, typ, nullable)
			}
			return strings.Contains(extra, "auto_increment"), nil
		}
		got := strings.Join([]string{typ, nullable, def, extra}, "|")
		if got != s.expected {
			return false, fmt.Errorf("incompatible %s.%s definition: %s", s.table, s.name, got)
		}
		return true, nil
	case "index":
		var cols sql.NullString
		err := c.QueryRowContext(ctx, `SELECT GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=?`, s.table, s.name).Scan(&cols)
		if err != nil {
			return false, err
		}
		if !cols.Valid {
			return false, nil
		}
		if strings.ReplaceAll(cols.String, "psc_stat", "purchase_status") != s.expected {
			return false, fmt.Errorf("incompatible index %s.%s: %s", s.table, s.name, cols.String)
		}
		return true, nil
	case "table":
		var engine string
		err := c.QueryRowContext(ctx, "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", s.table).Scan(&engine)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !strings.EqualFold(engine, "InnoDB") {
			return false, fmt.Errorf("%s must use InnoDB", s.table)
		}
		for name, want := range map[string]string{"story_idx": "int unsigned|NO|NULL|", "uid": "bigint unsigned|NO|NULL|", "last_counted_at": "datetime(6)|YES|NULL|"} {
			ok, err := (step{kind: "column", table: s.table, name: name, expected: want}).check(ctx, c)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, fmt.Errorf("missing %s.%s", s.table, name)
			}
		}
		ok, err := (step{kind: "index", table: s.table, name: "PRIMARY", expected: "story_idx,uid"}).check(ctx, c)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, fmt.Errorf("missing %s primary key", s.table)
		}
		return true, nil
	}
	return false, fmt.Errorf("unknown migration kind %s", s.kind)
}
