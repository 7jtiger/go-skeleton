package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"ms-gateway/conf"
	ptl "ms-gateway/protocol"
)

// Connection-local tables shadow real tables; no application records are changed.
func TestStoryFreshnessMySQL(t *testing.T) {
	path := os.Getenv("MS_GATEWAY_DB_CONFIG")
	if path == "" {
		t.Skip("set MS_GATEWAY_DB_CONFIG for temporary-table tests")
	}
	dsn, err := conf.NewConfig(path).MySQLDSN("sdb")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`CREATE TEMPORARY TABLE story (idx INT PRIMARY KEY,uid BIGINT UNSIGNED,nick VARCHAR(30) DEFAULT '',stat INT,main_pic VARCHAR(128),body TEXT,qt_like INT DEFAULT 0,qt_view INT DEFAULT 0,price INT,str_img JSON,at_create DATETIME DEFAULT CURRENT_TIMESTAMP,at_update DATETIME DEFAULT CURRENT_TIMESTAMP,cntt_rvsn TINYINT UNSIGNED DEFAULT 1)`,
		`CREATE TEMPORARY TABLE prf_info (uid BIGINT UNSIGNED PRIMARY KEY,nick VARCHAR(30),main_pic VARCHAR(128),gender INT,area INT,age VARCHAR(3),birth DATE)`,
		`CREATE TEMPORARY TABLE str_cmt (idx INT AUTO_INCREMENT PRIMARY KEY,str_idx INT,wuid BIGINT UNSIGNED,nick VARCHAR(30),thumb_url VARCHAR(256),wgender VARCHAR(2),wage VARCHAR(45),warea VARCHAR(45),body TEXT,stat INT,at_create DATETIME,at_update DATETIME)`,
		`CREATE TEMPORARY TABLE user_follow (uid BIGINT UNSIGNED,followee_uid BIGINT UNSIGNED,stat INT)`,
		`CREATE TEMPORARY TABLE str_pcs (str_idx INT,pcs_uid BIGINT UNSIGNED,psc_stat INT)`,
		`CREATE TEMPORARY TABLE story_like (story_idx INT,uid BIGINT UNSIGNED,stat INT)`,
	} {
		exec(q)
	}
	s := &StoryDB{conndb: db}
	img := `{"1":{"type":"img","url":"https://example.test/image"}}`
	video := `{"1":{"type":"vdo","url":"https://example.test/video","thumb1":"https://example.test/thumb"}}`
	for id := 1; id <= 110; id++ {
		stat, media := 0, img
		if id > 5 { // More than one full page of text or locked media precedes readable media.
			switch id % 3 {
			case 0:
				media = "{}"
			case 1:
				stat = 1
			case 2:
				stat = 2
			}
		}
		if id == 5 {
			media = video
		}
		exec("INSERT INTO story (idx,uid,stat,str_img) VALUES (?,10,?,?)", id, stat, media)
	}
	exec("INSERT INTO story (idx,uid,stat,str_img) VALUES (111,99,0,?)", img)
	t.Run("author_profile_refreshes_cached_and_list_responses", func(t *testing.T) {
		cached, err := s.GetStoryCacheEntry(5)
		if err != nil {
			t.Fatal(err)
		}
		wantAge := calcStoryAge("1990-02-03", "-2")
		for _, nick := range []string{"current", "updated"} {
			exec("REPLACE INTO prf_info VALUES (10,?,'profile.jpg',1,7,'-2','1990-02-03')", nick)
			list, err := s.RefreshStoryList([]ptl.StoryDetailResp{cached.Story}, 20)
			if err != nil {
				t.Fatal(err)
			}
			if len(list) != 1 || list[0].Nick != nick || list[0].MainPic != "profile.jpg" || list[0].Gender != 1 || list[0].Area != "daegu/gyeongbuk" || list[0].Age != wantAge {
				t.Fatalf("wrong cached author: %+v", list)
			}
			check := func(story ptl.StoryDetailResp) {
				t.Helper()
				if story.Nick != nick || story.MainPic != "profile.jpg" || story.Gender != 1 || story.Area != "daegu/gyeongbuk" || story.Age != wantAge {
					t.Fatalf("wrong joined author: %+v", story)
				}
			}
			page, _, err := s.GetStoryPage(20, 10, 1, 1)
			if err != nil || len(page) != 1 {
				t.Fatalf("page: %v %v", page, err)
			}
			check(page[0].Story)
			detail, err := s.GetStoryDetail(20, 5)
			if err != nil {
				t.Fatal(err)
			}
			check(*detail)
			raw, err := json.Marshal(detail)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["area"] != "daegu/gyeongbuk" {
				t.Fatalf("wrong detail response area: %s", raw)
			}
			if fields["age"] != wantAge {
				t.Fatalf("wrong detail response age: %s", raw)
			}
			if _, exists := fields["birth"]; exists {
				t.Fatalf("detail response exposes birth: %s", raw)
			}
			all, _, err := s.GetStoryList(20, 1, 200)
			if err != nil {
				t.Fatal(err)
			}
			for _, story := range *all {
				if story.Uid == 10 {
					check(story)
				}
			}
			exec("INSERT INTO user_follow VALUES (20,10,1)")
			following, _, err := s.GetFlwStoryList(20, 1, 1)
			if err != nil || len(*following) != 1 {
				t.Fatalf("following: %v %v", following, err)
			}
			check((*following)[0])
			condition, _, err := s.GetCondStoryList(20, []string{"uid = ?"}, "idx DESC", []interface{}{10, 1, 0})
			if err != nil || len(*condition) != 1 {
				t.Fatalf("condition: %v %v", condition, err)
			}
			check((*condition)[0])
			exec("INSERT INTO str_pcs VALUES (5,20,1)")
			purchased, _, err := s.GetPurchaseList(20, 1, 1)
			if err != nil || len(*purchased) != 1 {
				t.Fatalf("purchase: %v %v", purchased, err)
			}
			check((*purchased)[0])
			exec("DELETE FROM str_pcs")
			exec("DELETE FROM user_follow")

			compact, err := s.GetDefStoryList(20, 10, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(*compact) != 1 || (*compact)[0].Nick != nick || (*compact)[0].MainPic != "profile.jpg" || (*compact)[0].Gender != 1 || (*compact)[0].Area != "daegu/gyeongbuk" || (*compact)[0].Age != wantAge {
				t.Fatalf("wrong list author: %+v", compact)
			}
		}
		exec("DELETE FROM prf_info")
		list, err := s.RefreshStoryList([]ptl.StoryDetailResp{cached.Story}, 20)
		if err != nil || len(list) != 1 || list[0].Nick != "" || list[0].MainPic != "" || list[0].Age != "" {
			t.Fatalf("missing profile: %+v err=%v", list, err)
		}
	})
	t.Run("locked_responses_preserve_media", func(t *testing.T) {
		exec("INSERT INTO story (idx,uid,stat,str_img) VALUES (300,10,2,?)", video)
		defer exec("DELETE FROM story WHERE idx=300")
		check := func(story ptl.StoryDetailResp) {
			t.Helper()
			if story.IsLocked != 1 || story.StrImg["1"].URL != "https://example.test/video" || story.StrImg["1"].Thumb != "https://example.test/thumb" {
				t.Fatalf("locked media missing: %+v", story)
			}
		}
		detail, err := s.GetStoryDetail(20, 300)
		if err != nil {
			t.Fatal(err)
		}
		check(*detail)
		cached, err := s.GetStoryCacheEntry(300)
		if err != nil {
			t.Fatal(err)
		}
		refreshed, err := s.RefreshStoryList([]ptl.StoryDetailResp{cached.Story}, 20)
		if err != nil || len(refreshed) != 1 {
			t.Fatalf("cache: %v %v", refreshed, err)
		}
		check(refreshed[0])
		exec("INSERT INTO user_follow VALUES (20,10,1)")
		for _, fetch := range []func() (*[]ptl.StoryDetailResp, int, error){
			func() (*[]ptl.StoryDetailResp, int, error) { return s.GetStoryList(20, 1, 200) },
			func() (*[]ptl.StoryDetailResp, int, error) {
				return s.GetCondStoryList(20, nil, "idx DESC", []interface{}{200, 0})
			},
			func() (*[]ptl.StoryDetailResp, int, error) { return s.GetFlwStoryList(20, 1, 200) },
		} {
			list, _, err := fetch()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, story := range *list {
				if story.Idx == 300 {
					check(story)
					found = true
				}
			}
			if !found {
				t.Fatal("locked story missing")
			}
		}
		exec("DELETE FROM user_follow")
		exec("UPDATE story SET stat=1 WHERE idx=300")
		exec("INSERT INTO str_pcs VALUES (300,20,1)")
		defer exec("DELETE FROM str_pcs")
		list, _, err := s.GetPurchaseList(20, 1, 10)
		if err != nil || len(*list) != 1 {
			t.Fatalf("purchase: %v %v", list, err)
		}
		check((*list)[0])
	})

	t.Run("recent_media_skips_locked_and_text_across_pages", func(t *testing.T) {
		images, thumbs, err := s.GetStoryFirstPicUrls(20, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(*images) != 4 {
			t.Fatalf("expected four media entries, got %v", *images)
		}
		for _, id := range []int{5, 4, 3, 2} {
			if (*images)[id] == "" {
				t.Fatalf("missing %d: %v", id, *images)
			}
		}
		if (*thumbs)[5] != "https://example.test/thumb" {
			t.Fatalf("missing thumbnail: %v", *thumbs)
		}
	})
	t.Run("fewer_than_four_and_empty_profiles", func(t *testing.T) {
		for owner, want := range map[uint64]int{99: 1, 999: 0} {
			images, _, err := s.GetStoryFirstPicUrls(20, owner)
			if err != nil || len(*images) != want {
				t.Fatalf("owner=%d expected=%d err=%v", owner, want, err)
			}
		}
	})
	t.Run("follow_and_purchase_unlock_recent_media", func(t *testing.T) {
		exec("INSERT INTO user_follow VALUES (20,10,1)")
		exec("INSERT INTO str_pcs VALUES (110,20,1)")
		images, _, err := s.GetStoryFirstPicUrls(20, 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []int{110, 109, 106, 103} {
			if (*images)[id] == "" {
				t.Fatalf("missing unlocked %d: %v", id, *images)
			}
		}
		exec("DELETE FROM user_follow")
		exec("DELETE FROM str_pcs")
	})
	t.Run("cached_content_refreshes_comment_create_edit_delete", func(t *testing.T) {
		cached, err := s.GetStoryCacheEntry(5)
		if err != nil {
			t.Fatal(err)
		}
		refresh := func(wantCount int, wantBody string) {
			t.Helper()
			// Round-trip like Redis and reuse the SAME pre-mutation content snapshot.
			raw, err := json.Marshal(cached)
			if err != nil {
				t.Fatal(err)
			}
			var entry StoryCacheEntry
			if err := json.Unmarshal(raw, &entry); err != nil {
				t.Fatal(err)
			}
			list, err := s.RefreshStoryList([]ptl.StoryDetailResp{entry.Story}, 20)
			if err != nil {
				t.Fatal(err)
			}
			if len(list) != 1 || list[0].CmtCount != wantCount || len(list[0].CommentList) != wantCount {
				t.Fatalf("unexpected comments: %+v", list)
			}
			if wantCount > 0 && list[0].CommentList[0].Body != wantBody {
				t.Fatalf("stale comment: %+v", list[0].CommentList)
			}
		}
		refresh(0, "")
		id, err := s.SetStrComment(&ptl.StrComment{StrIdx: 5, Wuid: 20, Nick: "writer", Body: "before", Stat: 1})
		if err != nil {
			t.Fatal(err)
		}
		refresh(1, "before")
		rawCache, err := s.GetStoryCacheEntry(5)
		if err != nil {
			t.Fatal(err)
		}
		if rawCache.Story.CmtCount != 0 || len(rawCache.Story.CommentList) != 0 {
			t.Fatal("comments must not be stored in raw content cache")
		}
		if _, err := s.UpdateStrBodyComment(int(id), 20, false, "after"); err != nil {
			t.Fatal(err)
		}
		refresh(1, "after")
		if _, err := s.UpdateStrStatComment(int(id), 20, false, 4); err != nil {
			t.Fatal(err)
		}
		refresh(0, "")
		exec("INSERT INTO user_follow VALUES (20,10,1)")
		list, err := s.RefreshStoryList([]ptl.StoryDetailResp{cached.Story}, 20)
		if err != nil || len(list) != 1 || list[0].FlwCount != 1 {
			t.Fatalf("stale follower count: %+v %v", list, err)
		}
		exec("DELETE FROM user_follow")
	})
	t.Run("condition_count_and_scope_match", func(t *testing.T) {
		exec("INSERT INTO story (idx,uid,stat,str_img) VALUES (200,10,4,'{}'),(201,10,5,'{}'),(202,10,3,'{}')")
		ownPrivate, total, err := s.GetCondStoryList(10, []string{"stat = ?"}, "idx DESC", []interface{}{3, 200, 0})
		if err != nil || total != 1 || len(*ownPrivate) != 1 || (*ownPrivate)[0].Idx != 202 {
			t.Fatalf("owner private filter: %+v count=%d err=%v", ownPrivate, total, err)
		}
		for _, status := range []int{6, 0, 1, 2, 3} {
			conds := []string{}
			args := []interface{}{}
			if status != 6 {
				conds = append(conds, "stat = ?")
				args = append(args, status)
			}
			args = append(args, 200, 0)
			list, total, err := s.GetCondStoryList(20, conds, "idx DESC", args)
			if err != nil {
				t.Fatal(err)
			}
			if total != len(*list) {
				t.Fatalf("total=%d list=%d", total, len(*list))
			}
			for _, story := range *list {
				if story.Stat < 0 || story.Stat > 2 || (status != 6 && story.Stat != status) {
					t.Fatalf("wrong status: %d", story.Stat)
				}
				if story.Stat != 0 && (story.IsLocked != 1 || len(story.CommentList) != 0 || story.Body != "") {
					t.Fatalf("locked content exposed: %+v", story)
				}
			}
		}
	})
	t.Run("joined_condition_sort_and_purchase", func(t *testing.T) {
		for _, order := range []string{"new", "old", "cmt", "god", "view"} {
			if _, _, err := s.GetCondStoryList(20, nil, ptl.GetOrderQuery(order), []interface{}{10, 0}); err != nil {
				t.Fatalf("order %s: %v", order, err)
			}
		}
		exec("INSERT INTO str_pcs VALUES (5,20,1)")
		list, _, err := s.GetPurchaseList(20, 1, 10)
		if err != nil || len(*list) != 1 {
			t.Fatalf("purchase: %v %v", list, err)
		}
		exec("DELETE FROM str_pcs")
	})

	t.Run("price_and_status_require_owner", func(t *testing.T) {
		before, err := s.GetStoryCacheEntry(5)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			idx   int
			actor uint64
		}{{5, 20}, {999, 10}, {5, 0}} {
			if _, err := s.UpdatePriceStat(tc.idx, tc.actor, 2, 100); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("unauthorized price update: %v", err)
			}
		}
		for _, tc := range []struct{ stat, price int }{{6, 100}, {2, -1}} {
			if _, err := s.UpdatePriceStat(5, 10, tc.stat, tc.price); err == nil {
				t.Fatal("invalid update accepted")
			}
		}
		unchanged, err := s.GetStoryCacheEntry(5)
		if err != nil || unchanged.Story.Stat != before.Story.Stat || unchanged.Story.Price != before.Story.Price || unchanged.ContentRevision != before.ContentRevision {
			t.Fatalf("rejected update changed story: %+v %v", unchanged, err)
		}
		for _, price := range []int{100, 0} {
			if n, err := s.UpdatePriceStat(5, 10, 2, price); err != nil || n != 1 {
				t.Fatalf("owner update: %d %v", n, err)
			}
			after, err := s.GetStoryCacheEntry(5)
			if err != nil || after.Story.Stat != 2 || after.Story.Price != price || before.Matches(after) {
				t.Fatalf("update/cache: %+v %v", after, err)
			}
		}
	})

	t.Run("story_updates_require_owner", func(t *testing.T) {
		for _, tc := range []struct {
			idx   int
			actor uint64
		}{{5, 20}, {999, 10}, {5, 0}} {
			if _, err := s.UpdateStrBody(tc.idx, tc.actor, "unauthorized"); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("body unauthorized/missing: %v", err)
			}
			if _, err := s.UpdateStoryStat(tc.idx, tc.actor, 3); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("stat unauthorized/missing: %v", err)
			}
		}
		if n, err := s.UpdateStrBody(5, 10, "owner update"); err != nil || n != 1 {
			t.Fatalf("body owner update: %d %v", n, err)
		}
		if n, err := s.UpdateStoryStat(5, 10, 2); err != nil || n != 1 {
			t.Fatalf("stat owner update: %d %v", n, err)
		}
		var body string
		var stat int
		if err := db.QueryRow("SELECT body,stat FROM story WHERE idx=5").Scan(&body, &stat); err != nil || body != "owner update" || stat != 2 {
			t.Fatalf("persisted body=%q stat=%d err=%v", body, stat, err)
		}
		exec("UPDATE story SET cntt_rvsn=255 WHERE idx=5")
		// Force an actual zero-row-change result regardless of timestamp precision.
		result, err := db.Exec("UPDATE story SET body=body WHERE idx=5 AND uid=10")
		if err != nil {
			t.Fatal(err)
		}
		if n, err := s.storyMutationResult(result, 5, 10); err != nil || n != 0 {
			t.Fatalf("owned no-op misclassified: %d %v", n, err)
		}
		if _, err := s.UpdateStrBody(5, 10, "owner update"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.UpdateStoryStat(5, 10, 2); err != nil {
			t.Fatal(err)
		}
	})

}
