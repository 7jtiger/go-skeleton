package models

import (
	"database/sql"
	"errors"
	ptc "ms-gateway/protocol"
)

var ErrStoryLocked = errors.New("story content is locked")

// RequireStoryRead distinguishes a visible teaser from permission to read content.
func (p *StoryDB) RequireStoryRead(viewer uint64, idx int) (uint64, error) {
	var story ptc.StoryDetailResp
	if err := p.conndb.QueryRow("SELECT idx,uid,stat FROM story WHERE idx=?", idx).Scan(&story.Idx, &story.Uid, &story.Stat); err != nil {
		return 0, err
	}
	locked, err := p.storyAccess(&story, viewer)
	if err != nil {
		return 0, err
	}
	if locked {
		return 0, ErrStoryLocked
	}
	return story.Uid, nil
}

func (p *StoryDB) GetCommentStoryID(idx int, actor uint64, admin bool) (int, error) {
	if actor == 0 {
		return 0, sql.ErrNoRows
	}
	var storyID int
	err := p.conndb.QueryRow("SELECT str_idx FROM str_cmt WHERE idx=? AND (wuid=? OR ?)", idx, actor, admin).Scan(&storyID)
	return storyID, err
}
