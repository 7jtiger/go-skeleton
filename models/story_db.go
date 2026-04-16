package models

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	ptl "ms-gateway/protocol"
)

/*
CREATE TABLE `story` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `nick` varchar(20) NOT NULL,
  `body` varchar(512) DEFAULT NULL,
  `stat` tinyint DEFAULT '1' COMMENT 'stat=0 : del, stat=1 : pub, stat=2 : private, stat=3 : limit, stat=4 : ',
  `qt_good` int DEFAULT NULL,
  `qt_checked` int DEFAULT NULL COMMENT '조회수',
  `str_img` json NOT NULL,
  `at_update` datetime DEFAULT NULL,
  `at_create` datetime DEFAULT NULL,
  `str_imgbak` json DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `str_cmt` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `str_idx` int unsigned NOT NULL,
  `wuid` bigint NOT NULL,
  `nick` varchar(20) DEFAULT NULL,
  `thumb_url` varchar(256) DEFAULT NULL,
  `wgender` varchar(2) DEFAULT NULL,
  `wage` varchar(45) DEFAULT NULL,
  `warea` varchar(45) DEFAULT NULL,
  `body` varchar(256) DEFAULT NULL,
  `stat` tinyint DEFAULT NULL COMMENT 'stat=0:default, stat=1:private, stat=2::rerv, stat=3:rerv,stat=4:del',
  `at_create` datetime DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

*/

type StoryDB struct {
	conndb *sql.DB
	cfg    *conf.Config
	root   *Repositories

	quit     chan struct{}
	quitWait sync.WaitGroup
}

func NewStoryDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	r := &StoryDB{
		cfg:  cf,
		root: root,
		quit: make(chan struct{}),
	}

	var err error
	c := r.cfg.DB["sdb"]
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True",
		c["user"],
		c["pass"],
		c["host"],
		c["name"])

	r.conndb, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	r.conndb.SetMaxIdleConns(30)
	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

	go r.heartbeat()

	log.Info("load repository : Story db")
	return r, nil
}

func (p *StoryDB) Start() error {
	return nil
}

func (p *StoryDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (p *StoryDB) Close() error {
	return p.conndb.Close()
}

func (p *StoryDB) Ping() error {
	return p.conndb.Ping()
}

func (p *StoryDB) heartbeat() {
	p.quitWait.Add(1)
	defer p.quitWait.Done()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			for _, e := range p.root.elems {
				if repo, ok := e.Interface().(IRepository); ok {
					if err := repo.Ping(); err != nil {
						log.Error("mysql ping fail.", err)
					}
				} else {
					log.Error("element does not implement IRepository")
				}
			}
		}
	}
}

func (p *StoryDB) GetSList() (string, string, error) {
	row := p.conndb.QueryRow("SELECT privacy_url, terms_url FROM terms_info")
	var privacyUrl, termsUrl string
	err := row.Scan(&privacyUrl, &termsUrl)
	if err != nil {
		return "", "", err
	}
	return privacyUrl, termsUrl, nil
}

func (p *StoryDB) SetStory(simg *ptl.StoryImage) (int64, error) {
	query := `INSERT INTO story (uid, nick, body, stat, str_img, at_create, at_update) 
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := p.conndb.Exec(query, simg.Uid, simg.Nick, simg.Body, simg.Stat, simg.StrImg)
	if err != nil {
		return 0, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastID, nil
}

// stat = 0 : pub, stat = 1 : private, stat = 2 : limit, stat = 3 : resv, stat = 4 : del
func (p *StoryDB) GetStoryList(uid uint64) (*[]ptl.StoryListResp, error) {
	query := `SELECT idx, nick, str_img, at_create FROM story WHERE uid = ? AND stat IN (1, 0) ORDER BY at_create DESC`
	rows, err := p.conndb.Query(query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []ptl.StoryListResp
	for rows.Next() {
		var story ptl.StoryListResp
		err := rows.Scan(&story.Idx, &story.Nick, &story.StrImg, &story.AtCreate)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return &stories, nil
}

func (p *StoryDB) GetStory(strIdx int) (*ptl.StoryDetailResp, error) {
	query := `SELECT nick, body, str_img, at_create FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var story ptl.StoryDetailResp
	err := row.Scan(&story.Nick, &story.Body, &story.StrImg, &story.AtCreate)
	if err != nil {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, fmt.Errorf("story not found")
	}
	return &story, nil
}

func (p *StoryDB) UpdateStoryStat(strIdx int, stat int) (int64, error) {
	query := `UPDATE story SET stat = ?, at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, stat, strIdx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) UpdateStrBody(strIdx int, body string) (int64, error) {
	query := `UPDATE story SET body = ?, at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, body, strIdx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) SetStrComment(cmt *ptl.StrComment) (int64, error) {
	query := `INSERT INTO str_cmt (str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := p.conndb.Exec(
		query,
		cmt.StrIdx,
		cmt.Wuid,
		cmt.Nick,
		cmt.ThumbUrl,
		cmt.WGender,
		cmt.WAge,
		cmt.WArea,
		cmt.Body,
		cmt.Stat,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (p *StoryDB) GetStrCmtDetail(cmtIdx int, page int) (*[]ptl.StrComment, error) {
	const pageSize = 20
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	query := `SELECT idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update
	          FROM str_cmt
	          WHERE str_idx = ? AND stat IN (0,1)
	          ORDER BY at_create DESC
	          LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, cmtIdx, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []ptl.StrComment{}
	for rows.Next() {
		var comment ptl.StrComment
		var nick, thumbUrl, wgender, wage, warea, body sql.NullString
		err := rows.Scan(
			&comment.Idx,
			&comment.StrIdx,
			&comment.Wuid,
			&nick,
			&thumbUrl,
			&wgender,
			&wage,
			&warea,
			&body,
			&comment.Stat,
			&comment.AtCreate,
			&comment.AtUpdate,
		)
		if err != nil {
			return nil, err
		}
		// NULL string 처리
		comment.Nick = ""
		if nick.Valid {
			comment.Nick = nick.String
		}
		comment.ThumbUrl = ""
		if thumbUrl.Valid {
			comment.ThumbUrl = thumbUrl.String
		}
		comment.WGender = ""
		if wgender.Valid {
			comment.WGender = wgender.String
		}
		comment.WAge = ""
		if wage.Valid {
			comment.WAge = wage.String
		}
		comment.WArea = ""
		if warea.Valid {
			comment.WArea = warea.String
		}
		comment.Body = ""
		if body.Valid {
			comment.Body = body.String
		}
		comments = append(comments, comment)
	}
	return &comments, nil
}

func (p *StoryDB) GetStrCommentList(strIdx int64) (*[]ptl.StrComment, error) {
	query := `SELECT idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update
	          FROM str_cmt
	          WHERE str_idx = ? AND stat IN (0,1)
	          ORDER BY at_create DESC`
	// var comments []ptl.StrComment
	rows, err := p.conndb.Query(query, strIdx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []ptl.StrComment{}
	for rows.Next() {
		var (
			comment  ptl.StrComment
			nick     sql.NullString
			thumbUrl sql.NullString
			wgender  sql.NullString
			wage     sql.NullString
			warea    sql.NullString
			body     sql.NullString
		)
		err := rows.Scan(
			&comment.Idx,
			&comment.StrIdx,
			&comment.Wuid,
			&nick,
			&thumbUrl,
			&wgender,
			&wage,
			&warea,
			&body,
			&comment.Stat,
			&comment.AtCreate,
			&comment.AtUpdate,
		)
		if err != nil {
			return nil, err
		}
		comment.Nick = ""
		if nick.Valid {
			comment.Nick = nick.String
		}
		comment.ThumbUrl = ""
		if thumbUrl.Valid {
			comment.ThumbUrl = thumbUrl.String
		}
		comment.WGender = ""
		if wgender.Valid {
			comment.WGender = wgender.String
		}
		comment.WAge = ""
		if wage.Valid {
			comment.WAge = wage.String
		}
		comment.WArea = ""
		if warea.Valid {
			comment.WArea = warea.String
		}
		comment.Body = ""
		if body.Valid {
			comment.Body = body.String
		}
		comments = append(comments, comment)
	}
	return &comments, nil
}

func (p *StoryDB) GetStrPicList(strIdx int) (json.RawMessage, json.RawMessage, error) {
	query := `SELECT str_img, str_imgbak FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)

	fmt.Println("strIdx ", row)
	var strImg, strImgBak json.RawMessage
	err := row.Scan(&strImg, &strImgBak)
	fmt.Println("strImg", strImg)
	fmt.Println("strImgBak", strImgBak)
	if err != nil && strImg == nil {
		return nil, nil, err
	}

	// if strImg == nil {
	// 	return nil, nil, errors.New("str img is empty")
	// }

	return strImg, strImgBak, nil
}

func (p *StoryDB) DeleteStrPic(strIdx int, pic, picback json.RawMessage) (int64, error) {
	query := `UPDATE story SET str_img = ?, str_imgbak = ?, at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, pic, picback, strIdx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
	// return 0, nil
}

func (p *StoryDB) UpdateStrStatComment(cmtIdx int, stat int) (int64, error) {
	query := `UPDATE str_cmt SET stat = ?, at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, stat, cmtIdx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) UpdateStrBodyComment(cmtIdx int, body string) (int64, error) {
	query := `UPDATE str_cmt SET body = ?, at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, body, cmtIdx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
