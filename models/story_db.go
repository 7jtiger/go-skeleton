package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	ptc "ms-gateway/protocol"
)

/*
CREATE TABLE `prf_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `nick` varchar(20) DEFAULT NULL,
  `gender` tinyint(1) NOT NULL COMMENT 'woman = 0, man = 1',
  `birth` date DEFAULT '1900-12-30',
  `age` varchar(3) DEFAULT '0' COMMENT '+3 ~ -3',
  `area` tinyint DEFAULT NULL,
  `intro` varchar(128) DEFAULT NULL,
  `main_pic` varchar(128) DEFAULT NULL,
  `sub_pic1` json DEFAULT NULL,
  `sub_pic2` json DEFAULT NULL,
  `sub_pic3` json DEFAULT NULL,
  `sub_pic4` json DEFAULT NULL,
  `sub_pic5` json DEFAULT NULL,
  `at_update` date DEFAULT NULL,
  `at_create` date DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `story` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `nick` varchar(20) NOT NULL,
  `birth` varchar(12) DEFAULT NULL COMMENT '00-11-22',
  `area` tinyint DEFAULT NULL,
  `gender` int DEFAULT '1' COMMENT '0=woman, 1=man',
  `body` varchar(512) DEFAULT NULL,
  `type` tinyint DEFAULT '1' COMMENT '0=only img, 1=only video, 2=img+video',
  `stat` tinyint DEFAULT '1' COMMENT 'stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)',
  `qt_good` int DEFAULT '0',
  `qt_view` int DEFAULT '0' COMMENT 'view count',
  `qt_checked` int DEFAULT '0' COMMENT '조회수',
  `price` int DEFAULT '0',
  `str_img` json NOT NULL,
  `at_update` datetime DEFAULT NULL,
  `at_create` datetime DEFAULT NULL,
  `str_imgbak` json DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


CREATE TABLE `str_cmt` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `str_idx` int unsigned NOT NULL,
  `wuid` bigint NOT NULL,
  `nick` varchar(20) DEFAULT NULL,
  `thumb_url` varchar(256) DEFAULT NULL,
  `wgender` varchar(2) DEFAULT NULL,
  `wage` date DEFAULT NULL COMMENT '1900-12-30',
  `warea` varchar(45) DEFAULT NULL,
  `body` varchar(256) DEFAULT NULL,
  `stat` tinyint DEFAULT NULL COMMENT 'stat=0:default, stat=1:private, stat=2::rerv, stat=3:rerv,stat=4:del',
  `at_create` datetime DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `story_like` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `story_idx` int unsigned NOT NULL,
  `uid` bigint NOT NULL,
  `stat` tinyint NOT NULL DEFAULT '1' COMMENT '1:like, 0:unlike',
  `at_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_story_like_story_idx_uid` (`story_idx`,`uid`),
  KEY `idx_story_stat` (`story_idx`,`stat`,`at_update`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `str_pcs` (
  `idx` int NOT NULL,
  `str_idx` int NOT NULL,
  `tid` bigint DEFAULT NULL COMMENT '판매자',
  `pcs_uid` bigint NOT NULL COMMENT '구매자 uid',
  `at_create` date DEFAULT NULL,
  `at_update` date DEFAULT NULL,
  `qt_payed` bigint DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='스토리 구매 목록 리스트'

CREATE TABLE `user_follow` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `followee_uid` bigint NOT NULL,
  `stat` tinyint NOT NULL DEFAULT '1' COMMENT '1:follow, 0:unfollow',
  `at_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `uk_follow_pair` (`uid`,`followee_uid`),
  KEY `idx_uid_stat` (`uid`,`stat`,`at_update`),
  KEY `idx_followee_stat` (`followee_uid`,`stat`,`at_update`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci


CREATE TABLE `str_cutout` (
  `idx` bigint NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `tid` bigint NOT NULL,
  `stat` tinyint NOT NULL DEFAULT '1' COMMENT 'active:1, cancel:0',
  `tnick` varchar(20) DEFAULT NULL,
  `tgen` tinyint DEFAULT NULL,
  `tbirth` date DEFAULT NULL,
  `tsp_intro` varchar(100) DEFAULT NULL,
  `tarea` tinyint DEFAULT NULL,
  `tthmb_pic` varchar(145) DEFAULT NULL,
  `at_crt` datetime NOT NULL,
  `at_upd` datetime NOT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `uk_cutout_pair` (`uid`,`tid`),
  KEY `idx_cutout_uid_stat` (`uid`,`stat`,`at_upd` DESC),
  CONSTRAINT `chk_cutout_not_self` CHECK ((`uid` <> `tid`))
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

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

	// 풀 과다 방지: DB 4개 합산이 MySQL max_connections를 넘지 않도록 제한
	r.conndb.SetMaxIdleConns(25)
	r.conndb.SetMaxOpenConns(50)
	r.conndb.SetConnMaxLifetime(30 * time.Minute)
	r.conndb.SetConnMaxIdleTime(5 * time.Minute)

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

// --- story profile----------------------------------------------------------------------
func (p *StoryDB) SaveInitPrfInfo(req ptc.RegistReq) error {
	area := ptc.GetAreaCode(req.Area)
	query := `INSERT INTO prf_info (uid, nick, gender, birth, area, intro, at_update, at_create)
	VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`

	_, err := p.conndb.Exec(query, req.Uid, req.Nick, req.Gender, req.Birth, area, req.SPIntro)
	if err != nil {
		return err
	}

	return nil
}

// /prf/info -> GetPrfInfo ->
// 프로필 정보 조회
func (p *StoryDB) GetPrfInfo(uid uint64) (*ptc.PrfInfo, error) {
	query := `SELECT uid, nick, gender, birth, age, area, intro, main_pic FROM prf_info WHERE uid = ?`
	row := p.conndb.QueryRow(query, uid)
	var prf ptc.PrfInfo
	var mainPic, age sql.NullString
	err := row.Scan(&prf.Uid, &prf.Nick, &prf.Gender, &prf.Birth, &age, &prf.Area, &prf.Intro, &mainPic)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, err
	}
	if mainPic.Valid {
		prf.MainPic = mainPic.String
	} else {
		prf.MainPic = ""
	}
	if age.Valid {
		prf.Age = age.String
	} else {
		prf.Age = "0"
	}

	return &prf, nil
}

func (p *StoryDB) GetPrfPicList(uid uint64) (ptc.PrfPicMap, error) {
	query := `SELECT sub_pic1, sub_pic2, sub_pic3, sub_pic4, sub_pic5 FROM prf_info WHERE uid = ?`
	row := p.conndb.QueryRow(query, uid)

	var subPic1, subPic2, subPic3, subPic4, subPic5 sql.NullString
	err := row.Scan(&subPic1, &subPic2, &subPic3, &subPic4, &subPic5)
	if err != nil {
		return nil, err
	}

	out := make(ptc.PrfPicMap, 5)
	slots := []sql.NullString{subPic1, subPic2, subPic3, subPic4, subPic5}
	for i, slot := range slots {
		if !slot.Valid || strings.TrimSpace(slot.String) == "" || slot.String == "null" {
			continue
		}
		var pic ptc.PrfPicInfo
		if err := json.Unmarshal([]byte(slot.String), &pic); err != nil {
			log.Warn("GetPrfPicList skip invalid json", " uid ", uid, " slot ", i+1, " err ", err)
			continue
		}
		if strings.TrimSpace(pic.Url) == "" {
			continue
		}
		out[strconv.Itoa(i+1)] = pic
	}
	return out, nil
}

// prfPicWaitingBaseSQL unpivots sub_pic1~5 and keeps rows whose JSON stat is 1 or 2.
const prfPicWaitingBaseSQL = `
SELECT uid, nick, slot, pic_json FROM (
	SELECT uid, COALESCE(nick, '') AS nick, 1 AS slot, sub_pic1 AS pic_json
	FROM prf_info WHERE sub_pic1 IS NOT NULL AND JSON_TYPE(sub_pic1) <> 'NULL'
	UNION ALL
	SELECT uid, COALESCE(nick, '') AS nick, 2 AS slot, sub_pic2 AS pic_json
	FROM prf_info WHERE sub_pic2 IS NOT NULL AND JSON_TYPE(sub_pic2) <> 'NULL'
	UNION ALL
	SELECT uid, COALESCE(nick, '') AS nick, 3 AS slot, sub_pic3 AS pic_json
	FROM prf_info WHERE sub_pic3 IS NOT NULL AND JSON_TYPE(sub_pic3) <> 'NULL'
	UNION ALL
	SELECT uid, COALESCE(nick, '') AS nick, 4 AS slot, sub_pic4 AS pic_json
	FROM prf_info WHERE sub_pic4 IS NOT NULL AND JSON_TYPE(sub_pic4) <> 'NULL'
	UNION ALL
	SELECT uid, COALESCE(nick, '') AS nick, 5 AS slot, sub_pic5 AS pic_json
	FROM prf_info WHERE sub_pic5 IS NOT NULL AND JSON_TYPE(sub_pic5) <> 'NULL'
) AS pics
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(pic_json, '$.stat')) AS UNSIGNED) IN (1, 2)
`

// GetPrfPicWaitingList returns profile pics with stat=1|2 across all users (admin review), paged.
func (p *StoryDB) GetPrfPicWaitingList(page, limit int) ([]ptc.PrfPicWaitingItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int
	countQ := "SELECT COUNT(*) FROM (" + prfPicWaitingBaseSQL + ") AS waiting"
	if err := p.conndb.QueryRow(countQ).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQ := prfPicWaitingBaseSQL + " ORDER BY uid ASC, slot ASC LIMIT ? OFFSET ?"
	rows, err := p.conndb.Query(listQ, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]ptc.PrfPicWaitingItem, 0, limit)
	for rows.Next() {
		var (
			uid     uint64
			nick    string
			slot    int
			picJSON string
		)
		if err := rows.Scan(&uid, &nick, &slot, &picJSON); err != nil {
			return nil, 0, err
		}

		var pic ptc.PrfPicInfo
		if err := json.Unmarshal([]byte(picJSON), &pic); err != nil {
			log.Warn("GetPrfPicWaitingList skip invalid json", " uid ", uid, " slot ", slot, " err ", err)
			continue
		}
		list = append(list, ptc.PrfPicWaitingItem{
			Uid:  uid,
			Nick: nick,
			Slot: slot,
			Url:  pic.Url,
			Stat: pic.Stat,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// SetPrfPicStat updates sub_pic{slot} JSON $.stat for a user.
// Only slots currently in waiting state (stat 1 or 2) are updated. newStat is typically 3 (approved).
func (p *StoryDB) SetPrfPicStat(uid uint64, slot, newStat int) (int64, error) {
	if uid == 0 {
		return 0, fmt.Errorf("uid is required")
	}
	if slot < 1 || slot > 5 {
		return 0, fmt.Errorf("slot must be 1-5")
	}
	if newStat < 1 || newStat > 4 {
		return 0, fmt.Errorf("stat must be 1-4")
	}

	col := "sub_pic" + strconv.Itoa(slot)
	// col is whitelist-derived from validated slot (1~5) only
	query := fmt.Sprintf(`
UPDATE prf_info
SET %s = JSON_SET(%s, '$.stat', CAST(? AS UNSIGNED)), at_update = NOW()
WHERE uid = ?
  AND %s IS NOT NULL
  AND JSON_TYPE(%s) <> 'NULL'
  AND CAST(JSON_UNQUOTE(JSON_EXTRACT(%s, '$.stat')) AS UNSIGNED) IN (1, 2)
`, col, col, col, col, col)

	result, err := p.conndb.Exec(query, newStat, uid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) GetStoryFirstPicUrls(uid uint64) (*map[int]string, *map[int]string, error) {
	query := `SELECT idx, str_img FROM story WHERE uid = ? ORDER BY at_create DESC LIMIT 4`
	rows, err := p.conndb.Query(query, uid)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	strImgs := make(map[int]string)
	strthmbls := make(map[int]string)
	// var stories []ptc.StoryListResp
	for rows.Next() {
		i := 1
		var idx int
		var strImg string
		err := rows.Scan(&idx, &strImg)
		if err != nil {
			return nil, nil, err
		}
		//strImg := {"1": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public", "2": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0125d18e-aa56-45a0-98ac-853ecd3c1600/public"}

		// firstURL, tcnts, err := GetFirstPicUrl(strImg)
		// if err != nil {
		// 	return nil, nil, err
		// }

		i++
	}

	return &strImgs, &strthmbls, nil
}

func (p *StoryDB) UpdatePrfInfo(req ptc.PrfInfo) error {
	sets := make([]string, 0, 6)
	args := make([]interface{}, 0, 6)

	if req.Nick != "" {
		sets = append(sets, "nick = ?")
		args = append(args, req.Nick)
	}
	if req.Gender != "" {
		sets = append(sets, "gender = ?")
		args = append(args, req.Gender)
	}
	if req.Age != "" {
		sets = append(sets, "age = ?")
		args = append(args, req.Age)
	}
	if req.Area != "" {
		sets = append(sets, "area = ?")
		args = append(args, ptc.GetAreaCode(req.Area))
	}
	if req.Intro != "" {
		sets = append(sets, "intro = ?")
		args = append(args, req.Intro)
	}
	if len(sets) == 0 {
		return nil // 또는 fmt.Errorf("no fields to update")
	}
	sets = append(sets, "at_update = NOW()")
	args = append(args, req.Uid)
	query := "UPDATE prf_info SET " + strings.Join(sets, ", ") + " WHERE uid = ?"
	_, err := p.conndb.Exec(query, args...)
	return err
}

func (p *StoryDB) DeletePrfPic(uid uint64, slot int) error {
	// sub_pic{slot} 삭제 후 뒤 슬롯을 앞으로 당긴다 (JSON에 idx 없음 — 컬럼 위치가 순번).
	// 예: slot=2 → sub_pic2=sub_pic3, sub_pic3=sub_pic4, sub_pic4=sub_pic5, sub_pic5=NULL
	if slot < 1 || slot > 5 {
		return fmt.Errorf("slot must be between 1 and 5")
	}

	setCols := make([]string, 0, 6)
	for i := slot; i < 5; i++ {
		colNext := fmt.Sprintf("sub_pic%d", i+1)
		colCur := fmt.Sprintf("sub_pic%d", i)
		setCols = append(setCols, colCur+" = "+colNext)
	}
	setCols = append(setCols, "sub_pic5 = NULL")
	setCols = append(setCols, "at_update = NOW()")

	query := "UPDATE prf_info SET " + strings.Join(setCols, ", ") + " WHERE uid = ?"
	_, err := p.conndb.Exec(query, uid)
	return err
}

// /prf/set/mainpic/:url -> SetMainPic ->
func (p *StoryDB) SetMainPic(url string, uid uint64) error {
	query := `UPDATE prf_info SET main_pic = IFNULL(?, ''), at_update = NOW() WHERE uid = ?`
	_, err := p.conndb.Exec(query, url, uid)
	return err
}

func (p *StoryDB) UploadPrfPic(picInfos *[]ptc.PrfPicInfo, uid uint64) error {
	// 순차 저장: slice[0]→sub_pic1 … slice[n-1]→sub_pic{n}, 나머지는 NULL.
	// 각 컬럼 JSON: {"url":"...","stat":N} (idx 없음)
	if picInfos == nil {
		return nil
	}
	if len(*picInfos) > 5 {
		return fmt.Errorf("profile pictures allow at most 5 items")
	}

	sets := make([]string, 0, 6)
	args := make([]interface{}, 0, 6)
	for i := 0; i < 5; i++ {
		col := "sub_pic" + strconv.Itoa(i+1)
		if i < len(*picInfos) && strings.TrimSpace((*picInfos)[i].Url) != "" {
			raw, err := json.Marshal((*picInfos)[i])
			if err != nil {
				return err
			}
			sets = append(sets, col+" = ?")
			args = append(args, raw)
			continue
		}
		sets = append(sets, col+" = NULL")
	}

	sets = append(sets, "at_update = NOW()")
	args = append(args, uid)
	query := "UPDATE prf_info SET " + strings.Join(sets, ", ") + " WHERE uid = ?"
	_, err := p.conndb.Exec(query, args...)
	return err
}

// --- story profile ----------------------------------------------------------------------
func (p *StoryDB) GetPFMainPic(uid uint64) (string, error) {
	query := "SELECT main_pic FROM prf_info WHERE uid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, uid)

	var mainPic string
	err := row.Scan(&mainPic)
	if err != nil {
		// 허용: NULL(main_pic)도 OK → ""로 반환
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return mainPic, nil
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

func (p *StoryDB) SaveStory(simg *ptc.StoryImage, media ptc.StoryStrImg) (int64, error) {
	strImg, mtype, err := BuildStrImgForDB(media)
	if err != nil {
		return 0, err
	}
	simg.StrImg = strImg
	simg.MType = mtype

	query := `INSERT INTO story (uid, nick, birth, area, gender, main_pic, body, type, stat, price, str_img, at_create, at_update) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := p.conndb.Exec(query, simg.Uid, simg.Nick, simg.Birth,
		simg.Area, simg.Gender, simg.MainPic, simg.Body, simg.MType,
		simg.Stat, simg.Price, simg.StrImg)
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
func (p *StoryDB) GetDefStoryList(uid uint64, page int, limit int) (*[]ptc.StoryListResp, error) {
	query := `SELECT idx, uid, nick, stat, str_img, at_create FROM story WHERE stat IN (1, 2, 3) ORDER BY at_create DESC LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// var stories []ptc.StoryListResp
	var stories = make([]ptc.StoryListResp, 0)
	story := ptc.StoryListResp{}
	for rows.Next() {
		var raw json.RawMessage
		// var fstCnts ptc.StoryMediaItem
		err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &story.Stat, &raw, &story.AtCreate)
		if err != nil {
			return nil, err
		}

		// fstCnts, err = LoadStoryStrImg(raw)
		fstItem, nMaxCount, err := GetFirstPicUrl(string(raw))
		if err != nil {
			return nil, fmt.Errorf("story %d str_img: %w", story.Idx, err)
		}

		tmp := ptc.StoryListResp{
			Idx:      story.Idx,
			Uid:      story.Uid,
			Nick:     story.Nick,
			Stat:     story.Stat,
			CntCount: nMaxCount,
			CntUrl:   fstItem.URL,
			CntType:  fstItem.Type,
			CntThumb: fstItem.Thumb,
			AtCreate: story.AtCreate,
		}

		stories = append(stories, tmp)
	}

	return &stories, nil
}

// GetCondStoryList builds a filtered story feed.
// stat filter comes from conds (controller): 0=PUB, 1=FLW, 2=PAY, 3=PRV, 4=DEL, 5=RSV
func (p *StoryDB) GetCondStoryList(conds []string, orderQuery string, args []interface{}) (*[]ptc.StoryListResp, error) {
	base := `SELECT idx, uid, nick, stat, str_img, at_create FROM story`
	if len(conds) > 0 {
		base += " WHERE " + strings.Join(conds, " AND ")
	}
	query := base + " ORDER BY " + orderQuery + " LIMIT ? OFFSET ?"
	// fmt.Println("query", query, " args ", args)
	rows, err := p.conndb.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories = make([]ptc.StoryListResp, 0)
	story := ptc.StoryListResp{}
	for rows.Next() {
		var raw json.RawMessage
		// var fstCnts ptc.StoryMediaItem
		err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &story.Stat, &raw, &story.AtCreate)
		if err != nil {
			return nil, err
		}

		// fstCnts, err = LoadStoryStrImg(raw)
		fstItem, nMaxCount, err := GetFirstPicUrl(string(raw))
		if err != nil {
			return nil, fmt.Errorf("story %d str_img: %w", story.Idx, err)
		}

		tmp := ptc.StoryListResp{
			Idx:      story.Idx,
			Uid:      story.Uid,
			Nick:     story.Nick,
			Stat:     story.Stat,
			CntCount: nMaxCount,
			CntUrl:   fstItem.URL,
			CntType:  fstItem.Type,
			CntThumb: fstItem.Thumb,
			AtCreate: story.AtCreate,
		}

		stories = append(stories, tmp)
	}

	return &stories, nil
}

// BuyStory records a purchase in str_pcs (pcs_uid = buyer, tid = story owner).
func (p *StoryDB) BuyStory(pcsUid uint64, strIdx int) error {
	var tid uint64
	err := p.conndb.QueryRow(`SELECT uid FROM story WHERE idx = ?`, strIdx).Scan(&tid)
	if err == sql.ErrNoRows {
		return fmt.Errorf("story not found")
	}
	if err != nil {
		return err
	}

	query := `
		INSERT INTO str_pcs (idx, str_idx, tid, pcs_uid, at_create, at_update, qt_payed)
		SELECT COALESCE(MAX(idx), 0) + 1, ?, ?, ?, NOW(), NOW(), 0
		FROM str_pcs
	`
	_, err = p.conndb.Exec(query, strIdx, tid, pcsUid)
	return err
}

func (p *StoryDB) isPaid(idx int, tid uint64) (int, error) {
	query := `SELECT COUNT(1) FROM str_pcs WHERE str_idx = ? AND tid = ?`
	var count int
	err := p.conndb.QueryRow(query, idx, tid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetPurchaseList returns stories the user purchased (str_pcs.pcs_uid), newest first.
// pcs_uid = my uid, 구매한 사람
func (p *StoryDB) GetPurchaseList(uid uint64, page int, limit int) (*[]ptc.StoryListResp, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var totalCount int
	cntQuery := `
		SELECT COUNT(1)
		FROM str_pcs p
		INNER JOIN story s ON s.idx = p.str_idx
		WHERE p.pcs_uid = ?
	`
	if err := p.conndb.QueryRow(cntQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT s.idx, s.uid, s.nick, s.stat, s.str_img, s.at_create
		FROM str_pcs p
		INNER JOIN story s ON s.idx = p.str_idx
		WHERE p.pcs_uid = ?
		ORDER BY s.at_create DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	stories := make([]ptc.StoryListResp, 0, limit)
	for rows.Next() {
		var (
			story ptc.StoryListResp
			raw   json.RawMessage
		)
		if err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &story.Stat, &raw, &story.AtCreate); err != nil {
			return nil, 0, err
		}

		fstItem, nMaxCount, err := GetFirstPicUrl(string(raw))
		if err != nil {
			return nil, 0, fmt.Errorf("story %d str_img: %w", story.Idx, err)
		}

		stories = append(stories, ptc.StoryListResp{
			Idx:      story.Idx,
			Uid:      story.Uid,
			Nick:     story.Nick,
			Stat:     story.Stat,
			CntCount: nMaxCount,
			CntUrl:   fstItem.URL,
			CntType:  fstItem.Type,
			CntThumb: fstItem.Thumb,
			AtCreate: story.AtCreate,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return &stories, totalCount, nil
}

func (p *StoryDB) GetDetailList(tid uint64, nPage int, nLimit int) (*[]ptc.StoryDetailResp, int, error) {
	cntQuery := `SELECT COUNT(1) FROM story WHERE stat IN (0, 1, 2, 3)`
	var totalCount int
	if err := p.conndb.QueryRow(cntQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// query := `SELECT idx, uid, nick, stat, str_img, at_create FROM story WHERE idx = ? AND uid = ? ORDER BY at_create DESC LIMIT ? OFFSET ?`
	// query := `SELECT idx, uid, nick, body, stat, qt_good, qt_view, price, str_img, at_create FROM story WHERE uid = ? AND idx <= ? ORDER BY idx DESC LIMIT ? OFFSET ?`
	query := `SELECT idx, uid, nick, stat, main_pic, body, qt_good, qt_view, price, str_img, at_create FROM story WHERE stat IN (0,1,2,3) ORDER BY at_create DESC LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, nLimit, (nPage-1)*nLimit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	stlist := make([]ptc.StoryDetailResp, 0)
	for rows.Next() {
		var raw json.RawMessage
		story := ptc.StoryDetailResp{}
		err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &story.Stat, &story.MainPic, &story.Body, &story.QtGood, &story.QtView, &story.Price, &raw, &story.AtCreate)
		if err != nil {
			return nil, 0, err
		}

		// story.StrImg, err = LoadStoryStrImg(raw)
		var media ptc.StoryStrImg
		if err := json.Unmarshal([]byte(raw), &media); err != nil {
			return nil, 0, fmt.Errorf("invalid media JSON: %w", err)
		}

		cmtlist, tcount, err := p.GetStrComment2Low(int64(story.Idx))
		if err != nil {
			return nil, 0, err
		}

		story.ContentTotal = len(media)
		story.StrImg = media
		if tcount > 0 {
			story.CmtCount = tcount
			story.CommentList = *cmtlist
		} else {
			story.CmtCount = 0
			story.CommentList = nil
		}

		paid, err := p.isPaid(story.Idx, tid)
		if err != nil {
			return nil, 0, err
		}
		story.IsPaid = paid // 0=not paid, 1=paid

		Followed, err := p.IsFollowed(tid, story.Uid)
		if err != nil {
			return nil, 0, err
		}
		story.IsFollowed = Followed // 0=not followed, 1=followed

		stlist = append(stlist, story)
	}
	return &stlist, totalCount, nil
}

func (p *StoryDB) GetStory(strIdx int) (*ptc.StoryDetailResp, error) {
	query := `SELECT nick, body, str_img, at_create FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var (
		story ptc.StoryDetailResp
		raw   json.RawMessage
	)
	err := row.Scan(&story.Nick, &story.Body, &raw, &story.AtCreate)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("story not found")
	}
	if err != nil {
		return nil, err
	}
	var media ptc.StoryStrImg
	if err := json.Unmarshal([]byte(raw), &media); err != nil {
		return nil, fmt.Errorf("invalid media JSON: %w", err)
	}
	story.StrImg = media
	return &story, nil
}

func (p *StoryDB) GetStoryViewCount(strIdx int) (int, error) {
	query := `SELECT qt_view FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var qtView int
	err := row.Scan(&qtView)
	if err != nil {
		return 0, err
	}
	return qtView, nil
}

func (p *StoryDB) GetStoryLikeCount(strIdx int) (int, error) {
	query := `SELECT qt_good FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var qtGood int
	err := row.Scan(&qtGood)
	if err != nil {
		return 0, err
	}
	return qtGood, nil
}

// UpdateStoryView increments story.qt_view by 1 and returns the new value.
// Uses LAST_INSERT_ID(expr) so one UPDATE covers both increment and return.
func (p *StoryDB) UpdateStoryView(strIdx int) error {
	query := `UPDATE story SET qt_view = LAST_INSERT_ID(COALESCE(qt_view, 0) + 1), at_update = NOW() WHERE idx = ?`
	result, err := p.conndb.Exec(query, strIdx)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("story not found")
	}

	return nil
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

func (p *StoryDB) SetStrComment(cmt *ptc.StrComment) (int64, error) {
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

// func (p *StoryDB) GetStrCmtDetail(cmtIdx int, page int) (*[]ptc.StrComment, int, error) {
func (p *StoryDB) GetStrCmtDetail(cmtIdx, page, limit int) (*[]ptc.StrComment, int, error) {
	if limit == 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	countQuery := `SELECT COUNT(1) FROM str_cmt WHERE str_idx = ? AND stat IN (0,1)`
	var totalCount int
	if err := p.conndb.QueryRow(countQuery, cmtIdx).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `SELECT idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update
	          FROM str_cmt
	          WHERE str_idx = ? AND stat IN (0,1)
	          ORDER BY at_create DESC
	          LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, cmtIdx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// comments := []ptc.StrComment{}
	comments := make([]ptc.StrComment, 0)
	for rows.Next() {
		var comment ptc.StrComment
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
			return nil, 0, err
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
	return &comments, totalCount, nil
}

func (p *StoryDB) GetStrComment2Low(strIdx int64) (*[]ptc.StrComment, int, error) {
	var totalCount int
	countQuery := `SELECT COUNT(1) FROM str_cmt WHERE str_idx = ? AND stat IN (0,1)`
	if err := p.conndb.QueryRow(countQuery, strIdx).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `SELECT idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update
	          FROM str_cmt
	          WHERE str_idx = ? AND stat IN (0,1)
	          ORDER BY at_create DESC LIMIT 2`
	// var comments []ptl.StrComment
	rows, err := p.conndb.Query(query, strIdx)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// comments := []ptc.StrComment{}
	comments := make([]ptc.StrComment, 0)
	for rows.Next() {
		var comment ptc.StrComment
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
			return nil, 0, err
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

	return &comments, totalCount, nil
}

func (p *StoryDB) GetStrPicList(strIdx int) (json.RawMessage, json.RawMessage, error) {
	query := `SELECT str_img, str_imgbak FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)

	// fmt.Println("strIdx ", row)
	var strImg, strImgBak json.RawMessage
	err := row.Scan(&strImg, &strImgBak)
	// fmt.Println("strImg", strImg)
	// fmt.Println("strImgBak", strImgBak)
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

func (p *StoryDB) ToggleStoryLikeTx(storyIdx int, uid uint64) (bool, int, error) {
	tx, err := p.conndb.Begin()
	if err != nil {
		return false, 0, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	var stat int
	row := tx.QueryRow(`SELECT stat FROM story_like WHERE story_idx = ? AND uid = ? FOR UPDATE`, storyIdx, uid)
	err = row.Scan(&stat)
	if err != nil {
		if err == sql.ErrNoRows {
			if _, err = tx.Exec(`INSERT INTO story_like (story_idx, uid, stat, at_create, at_update) VALUES (?, ?, 1, NOW(), NOW())`, storyIdx, uid); err != nil {
				return false, 0, err
			}
			if _, err = tx.Exec(`UPDATE story SET qt_good = COALESCE(qt_good, 0) + 1, at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
				return false, 0, err
			}
			stat = 1
		} else {
			return false, 0, err
		}
	} else {
		if stat == 1 {
			if _, err = tx.Exec(`UPDATE story_like SET stat = 0, at_update = NOW() WHERE story_idx = ? AND uid = ?`, storyIdx, uid); err != nil {
				return false, 0, err
			}
			if _, err = tx.Exec(`UPDATE story SET qt_good = GREATEST(COALESCE(qt_good, 0) - 1, 0), at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
				return false, 0, err
			}
			stat = 0
		} else {
			if _, err = tx.Exec(`UPDATE story_like SET stat = 1, at_update = NOW() WHERE story_idx = ? AND uid = ?`, storyIdx, uid); err != nil {
				return false, 0, err
			}
			if _, err = tx.Exec(`UPDATE story SET qt_good = COALESCE(qt_good, 0) + 1, at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
				return false, 0, err
			}
			stat = 1
		}
	}

	var likeCount int
	if err = tx.QueryRow(`SELECT COALESCE(qt_good, 0) FROM story WHERE idx = ?`, storyIdx).Scan(&likeCount); err != nil {
		return false, 0, err
	}

	if err = tx.Commit(); err != nil {
		return false, 0, err
	}
	rollback = false
	return stat == 1, likeCount, nil
}

func (p *StoryDB) IsFollowed(tid, uid uint64) (int, error) {
	query := `SELECT COUNT(1) FROM user_follow WHERE uid = ? AND followee_uid = ?`
	var count int
	err := p.conndb.QueryRow(query, uid, tid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *StoryDB) SetFollow(followerUid, followeeUid uint64) (int64, error) {
	query := `
		INSERT INTO user_follow (follower_uid, followee_uid, stat, at_create, at_update)
		VALUES (?, ?, 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE stat = 1, at_update = NOW()
	`
	result, err := p.conndb.Exec(query, followerUid, followeeUid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) SetUnfollow(followerUid, followeeUid uint64) (int64, error) {
	query := `UPDATE user_follow SET stat = 0, at_update = NOW() WHERE follower_uid = ? AND followee_uid = ?`
	result, err := p.conndb.Exec(query, followerUid, followeeUid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) GetFollowerCount(uid uint64) (int64, error) {
	query := `SELECT COUNT(1) FROM user_follow WHERE followee_uid = ? AND stat = 1`
	var count int64
	err := p.conndb.QueryRow(query, uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *StoryDB) GetFollowingCount(uid uint64) (int64, error) {
	query := `SELECT COUNT(1) FROM user_follow WHERE uid = ? AND stat = 1`
	var count int64
	err := p.conndb.QueryRow(query, uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *StoryDB) GetFollowerList(uid uint64, page int, limit int) (*[]ptc.FollowUserItem, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(1)
		FROM user_follow uf
		WHERE uf.followee_uid = ? AND uf.stat = 1
	`
	var totalCount int
	if err := p.conndb.QueryRow(countQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT uf.follower_uid, COALESCE(u.nick, ''), COALESCE(u.thumb_pic, ''), uf.at_update
		FROM user_follow uf
		LEFT JOIN user_info u ON u.uid = uf.follower_uid
		WHERE uf.followee_uid = ? AND uf.stat = 1
		ORDER BY uf.at_update DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	list := make([]ptc.FollowUserItem, 0, limit)
	for rows.Next() {
		var item ptc.FollowUserItem
		if err = rows.Scan(&item.Uid, &item.Nick, &item.ThumbPic, &item.AtUpdate); err != nil {
			return nil, totalCount, err
		}
		list = append(list, item)
	}
	if err = rows.Err(); err != nil {
		return nil, totalCount, err
	}
	return &list, totalCount, nil
}

func (p *StoryDB) GetFollowingList(uid uint64, page, limit int) (*[]ptc.FollowUserItem, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(1)
		FROM user_follow uf
		WHERE uf.follower_uid = ? AND uf.stat = 1
	`
	var totalCount int
	if err := p.conndb.QueryRow(countQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT uf.followee_uid, COALESCE(u.nick, ''), COALESCE(u.thumb_pic, ''), uf.at_update
		FROM user_follow uf
		LEFT JOIN user_info u ON u.uid = uf.followee_uid
		WHERE uf.follower_uid = ? AND uf.stat = 1
		ORDER BY uf.at_update DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	list := make([]ptc.FollowUserItem, 0, limit)
	for rows.Next() {
		var item ptc.FollowUserItem
		if err = rows.Scan(&item.Uid, &item.Nick, &item.ThumbPic, &item.AtUpdate); err != nil {
			return nil, totalCount, err
		}
		list = append(list, item)
	}

	if err = rows.Err(); err != nil {
		return nil, totalCount, err
	}

	return &list, totalCount, nil
}

func (p *StoryDB) GetStoryOwnerUID(storyIdx int) (uint64, error) {
	var uid uint64
	err := p.conndb.QueryRow(`SELECT uid FROM story WHERE idx = ?`, storyIdx).Scan(&uid)
	if err != nil {
		return 0, err
	}
	return uid, nil
}

//-------------------- user cut out ---------------------------------

func (p *StoryDB) SetCutoutUser(uid uint64, cutout *ptc.CutoutUserItem) (int64, error) {
	tbirth := utils.Time2StrDay(cutout.Tbirth)
	query := `
		INSERT INTO str_cutout
			(uid, tid, stat, tnick, tgen, tbirth, tsp_intro, tarea, tthmb_pic, at_crt, at_upd)
		VALUES
			(?, ?, 1, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			stat = 1,
			tnick = VALUES(tnick),
			tgen = VALUES(tgen),
			tbirth = VALUES(tbirth),
			tsp_intro = VALUES(tsp_intro),
			tarea = VALUES(tarea),
			tthmb_pic = VALUES(tthmb_pic),
			at_upd = NOW()
	`
	result, err := p.conndb.Exec(
		query,
		uid,
		cutout.Tid,
		cutout.Tnick,
		cutout.Tgen,
		tbirth,
		cutout.TspIntro,
		cutout.Tarea,
		cutout.TthmbPic,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) UnsetCutoutUser(uid, tid uint64) (int64, error) {
	query := `UPDATE str_cutout SET stat = 0, at_upd = NOW() WHERE uid = ? AND tid = ?`
	result, err := p.conndb.Exec(query, uid, tid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) GetCutoutCount(uid uint64) (int64, error) {
	query := `SELECT COUNT(1) FROM str_cutout WHERE uid = ? AND stat = 1`
	var count int64
	err := p.conndb.QueryRow(query, uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *StoryDB) GetActiveCutoutTIDs(uid uint64) ([]uint64, error) {
	rows, err := p.conndb.Query(`SELECT tid FROM str_cutout WHERE uid = ? AND stat = 1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tids := make([]uint64, 0)
	for rows.Next() {
		var tid uint64
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		tids = append(tids, tid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tids, nil
}

func (p *StoryDB) GetCutoutList(uid uint64, page, limit int) (*[]ptc.CutoutUserItem, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var totalCount int
	countQuery := `SELECT COUNT(1) FROM str_cutout WHERE uid = ? AND stat = 1`
	if err := p.conndb.QueryRow(countQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT idx, tid, stat, tnick, tgen, tbirth, tsp_intro, tarea, tthmb_pic, at_crt, at_upd
		FROM str_cutout
		WHERE uid = ? AND stat = 1
		ORDER BY at_upd DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]ptc.CutoutUserItem, 0, limit)
	for rows.Next() {
		var item ptc.CutoutUserItem
		if err = rows.Scan(
			&item.Idx,
			&item.Tid,
			&item.Stat,
			&item.Tnick,
			&item.Tgen,
			&item.Tbirth,
			&item.TspIntro,
			&item.Tarea,
			&item.TthmbPic,
			&item.AtCreate,
			&item.AtUpdate,
		); err != nil {
			return nil, 0, err
		}
		item.Age = utils.CalcBirth2Age(item.Tbirth)
		list = append(list, item)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return &list, totalCount, nil
}

//-------------------- user cut out ---------------------------------
