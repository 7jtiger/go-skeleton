package models

import (
	"database/sql"
	"encoding/json"
	"strings"

	ptc "ms-gateway/protocol"
)

func storyVisible(story *ptc.StoryDetailResp, viewer uint64) bool {
	switch story.Stat {
	case ptc.StoryStatPublic, ptc.StoryStatFollower, ptc.StoryStatPaid:
		return true
	case ptc.StoryStatPrivate, ptc.StoryStatReserved:
		return viewer != 0 && viewer == story.Uid
	default:
		return false
	}
}

func redactStory(story *ptc.StoryDetailResp, viewer uint64) {
	story.IsLocked = 0
	if viewer != 0 && viewer == story.Uid {
		return
	}
	if (story.Stat == ptc.StoryStatFollower && story.IsFollowed == 0) || (story.Stat == ptc.StoryStatPaid && story.IsPaid == 0) {
		story.IsLocked = 1
		// Media stays in the response; clients use IsLocked to control presentation.
		story.Body, story.CommentList, story.CmtCount = "", nil, 0
	}
}

func sqlPlaceholders(count int) string { return strings.TrimSuffix(strings.Repeat("?,", count), ",") }

// ApplyStoryAccessBatch resolves both relationship flags in three queries per page.
// It operates on request-local copies; originals can still be cached safely.
func (p *StoryDB) ApplyStoryAccessBatch(stories []ptc.StoryDetailResp, viewer uint64) ([]ptc.StoryDetailResp, error) {
	visible := make([]ptc.StoryDetailResp, 0, len(stories))
	authors := make(map[uint64]bool)
	ids := make([]interface{}, 0, len(stories))
	for _, story := range stories {
		if !storyVisible(&story, viewer) {
			continue
		}
		story.IsPaid, story.IsFollowed = 0, 0
		story.GoodByMe = 0
		visible = append(visible, story)
		authors[story.Uid] = true
		ids = append(ids, story.Idx)
	}
	if len(visible) == 0 {
		return visible, nil
	}
	if viewer != 0 {
		args := []interface{}{viewer}
		for uid := range authors {
			args = append(args, uid)
		}
		rows, err := p.conndb.Query("SELECT followee_uid FROM user_follow WHERE uid = ? AND stat = 1 AND followee_uid IN ("+sqlPlaceholders(len(authors))+")", args...)
		if err != nil {
			return nil, err
		}
		followed := make(map[uint64]bool)
		for rows.Next() {
			var uid uint64
			if err := rows.Scan(&uid); err != nil {
				rows.Close()
				return nil, err
			}
			followed[uid] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		args = append([]interface{}{viewer}, ids...)
		rows, err = p.conndb.Query("SELECT DISTINCT str_idx FROM str_pcs WHERE pcs_uid = ? AND psc_stat = 1 AND str_idx IN ("+sqlPlaceholders(len(ids))+")", args...)
		if err != nil {
			return nil, err
		}
		paid := make(map[int]bool)
		for rows.Next() {
			var idx int
			if err := rows.Scan(&idx); err != nil {
				rows.Close()
				return nil, err
			}
			paid[idx] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}

		rows, err = p.conndb.Query("SELECT story_idx FROM story_like WHERE uid=? AND stat=1 AND story_idx IN ("+sqlPlaceholders(len(ids))+")", args...)
		if err != nil {
			return nil, err
		}
		liked := map[int]bool{}
		for rows.Next() {
			var idx int
			if err := rows.Scan(&idx); err != nil {
				rows.Close()
				return nil, err
			}
			liked[idx] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for i := range visible {
			if liked[visible[i].Idx] {
				visible[i].GoodByMe = 1
			}
			if followed[visible[i].Uid] {
				visible[i].IsFollowed = 1
			}
			if paid[visible[i].Idx] {
				visible[i].IsPaid = 1
			}
		}
	}
	for i := range visible {
		redactStory(&visible[i], viewer)
	}
	return visible, nil
}

// fillStoryAuxiliary loads the latest two comments, total comments, author
// profiles and active follower counts in two queries (MySQL 8 window functions).
func (p *StoryDB) fillStoryAuxiliary(stories []*ptc.StoryDetailResp) error {
	if len(stories) == 0 {
		return nil
	}
	ids := make([]interface{}, 0, len(stories))
	byID := make(map[int][]*ptc.StoryDetailResp)
	for _, story := range stories {
		if _, exists := byID[story.Idx]; !exists {
			ids = append(ids, story.Idx)
		}
		byID[story.Idx] = append(byID[story.Idx], story)
		story.ContentTotal = len(story.StrImg)
		story.CommentList, story.CmtCount, story.FlwCount = nil, 0, 0
	}
	query := `SELECT idx,str_idx,wuid,COALESCE(nick,''),COALESCE(thumb_url,''),COALESCE(wgender,''),COALESCE(CAST(wage AS CHAR),''),COALESCE(warea,''),COALESCE(body,''),stat,at_create,at_update,total_count
 FROM (SELECT c.*, ROW_NUMBER() OVER (PARTITION BY str_idx ORDER BY at_create DESC,idx DESC) AS rn,
 COUNT(*) OVER (PARTITION BY str_idx) AS total_count FROM str_cmt c
 WHERE stat IN (0,1) AND str_idx IN (` + sqlPlaceholders(len(ids)) + `)) ranked
 WHERE rn <= 2 ORDER BY str_idx, rn`
	rows, err := p.conndb.Query(query, ids...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var c ptc.StrComment
		var total int
		if err := rows.Scan(&c.Idx, &c.StrIdx, &c.Wuid, &c.Nick, &c.ThumbUrl, &c.WGender, &c.WAge, &c.WArea, &c.Body, &c.Stat, &c.AtCreate, &c.AtUpdate, &total); err != nil {
			rows.Close()
			return err
		}
		for _, story := range byID[int(c.StrIdx)] {
			story.CommentList = append(story.CommentList, c)
			story.CmtCount = total
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// This connection uses sdb; LEFT JOIN preserves stories without a profile.
	rows, err = p.conndb.Query(`SELECT s.idx,COALESCE(p.nick,''),COALESCE(p.main_pic,''),
 COALESCE(p.gender,0),COALESCE(p.area,0),COALESCE(p.age,''),COALESCE(DATE_FORMAT(p.birth,'%Y-%m-%d'),''),
 (SELECT COUNT(*) FROM user_follow f WHERE f.stat=1 AND f.followee_uid=s.uid)
 FROM story s LEFT JOIN prf_info p ON p.uid=s.uid
 WHERE s.idx IN (`+sqlPlaceholders(len(ids))+`)`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var author ptc.StoryDetailResp
		var birth, profileAge string
		var areaCode int
		if err := rows.Scan(&author.Idx, &author.Nick, &author.MainPic, &author.Gender, &areaCode, &profileAge, &birth, &author.FlwCount); err != nil {
			return err
		}
		author.Area = ptc.GetAreaName(areaCode)
		author.Age = calcStoryAge(birth, profileAge)
		for _, story := range byID[author.Idx] {
			story.Nick, story.MainPic = author.Nick, author.MainPic
			story.Gender, story.Area, story.Age = author.Gender, author.Area, author.Age
			story.FlwCount = author.FlwCount
		}
	}
	return rows.Err()
}

// RefreshStoryList reloads author profiles, comments and follower counts on every response,
// including cache hits, then applies the current viewer permissions.
func (p *StoryDB) RefreshStoryList(stories []ptc.StoryDetailResp, viewer uint64) ([]ptc.StoryDetailResp, error) {
	visible := make([]ptc.StoryDetailResp, 0, len(stories))
	for _, story := range stories {
		if storyVisible(&story, viewer) {
			visible = append(visible, story)
		}
	}
	refs := make([]*ptc.StoryDetailResp, len(visible))
	for i := range visible {
		refs[i] = &visible[i]
	}
	if err := p.fillStoryAuxiliary(refs); err != nil {
		return nil, err
	}
	return p.ApplyStoryAccessBatch(visible, viewer)
}

// Read all rows before auxiliary queries, so each request uses only one SQL
// connection at a time instead of retaining the page cursor while acquiring more.
func (p *StoryDB) readStoryList(rows *sql.Rows, viewer uint64) ([]ptc.StoryDetailResp, error) {
	defer rows.Close()
	stories := make([]ptc.StoryDetailResp, 0)
	for rows.Next() {
		var story ptc.StoryDetailResp
		var birth, profileAge string
		var areaCode int
		var raw json.RawMessage
		var mainPic, body sql.NullString
		var price sql.NullInt64
		if err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &mainPic, &story.Gender, &areaCode, &birth, &profileAge, &story.Stat, &body, &story.QtLike, &story.QtView, &price, &raw, &story.AtCreate); err != nil {
			return nil, err
		}
		story.Area = ptc.GetAreaName(areaCode)
		story.Age = calcStoryAge(birth, profileAge)
		story.MainPic, story.Body, story.Price = mainPic.String, body.String, int(price.Int64)
		if err := json.Unmarshal(raw, &story.StrImg); err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return p.RefreshStoryList(stories, viewer)
}

func (p *StoryDB) GetStoryCacheEntries(ids []int) ([]StoryCacheEntry, error) {
	if len(ids) == 0 {
		return []StoryCacheEntry{}, nil
	}
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return p.loadStoryCacheEntries(storyCacheColumns+" WHERE idx IN ("+sqlPlaceholders(len(ids))+")", args...)
}

func (p *StoryDB) PoolStats() sql.DBStats { return p.conndb.Stats() }
