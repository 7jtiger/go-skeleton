package models

import (
	"encoding/json"
	"errors"
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
  `type` tinyint DEFAULT '1' COMMENT '0=img, 1=vdo, 2=txt, 3=mix, 4=rsv',
  `stat` tinyint DEFAULT '1' COMMENT 'stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)',
  `qt_like` int DEFAULT '0',
  `qt_view` int DEFAULT '0' COMMENT 'view count',
  `qt_checked` int DEFAULT '0' COMMENT '조회수',
  `price` int DEFAULT '0',
  `str_img` json NOT NULL,
  `at_update` datetime DEFAULT NULL,
  `at_create` datetime DEFAULT NULL,
  `str_imgbak` json DEFAULT NULL,
  `cntt_rvsn` tinyint unsigned NOT NULL DEFAULT 1 COMMENT '콘텐츠 변경 버전: 본문·상태·미디어 수정 시 증가, 같은 초 변경의 캐시 검증용. 0=미확인, 1~254=버전, 255=포화 시 캐시 사용 안 함',
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
  `idx` int NOT NULL AUTO_INCREMENT,
  `str_idx` int NOT NULL,
  `tid` bigint DEFAULT NULL COMMENT '판매자',
  `pcs_uid` bigint NOT NULL COMMENT '구매자 uid',
  `at_create` date DEFAULT NULL,
  `at_update` date DEFAULT NULL,
  `qt_payed` bigint DEFAULT NULL,
  `psc_stat` tinyint NOT NULL DEFAULT '0' COMMENT '구매 상태: 0=대기, 1=완료(열람 권한), 2=회수. 현재 BuyStory는 결제 없이 완료 처리',
  PRIMARY KEY (`idx`),
  KEY `idx_str_pcs_story_buyer` (`str_idx`,`pcs_uid`,`idx`)
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
			return nil, sql.ErrNoRows
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

// GetStoryFirstPicUrls fills up to four readable media entries, scanning past
// text-only and locked stories without exposing their original media.
func (p *StoryDB) GetStoryFirstPicUrls(viewer, owner uint64) (*map[int]string, *map[int]string, error) {
	images, thumbs := map[int]string{}, map[int]string{}
	for page := 1; len(images) < 4; page++ {
		list, err := p.GetDefStoryList(viewer, owner, page, 100)
		if err != nil {
			return nil, nil, err
		}
		for _, story := range *list {
			if story.IsLocked == 1 || story.CntUrl == "" {
				continue
			}
			images[story.Idx] = story.CntUrl
			if story.CntThumb != "" {
				thumbs[story.Idx] = story.CntThumb
			}
			if len(images) == 4 {
				break
			}
		}
		if len(*list) < 100 {
			break
		}
	}

	return &images, &thumbs, nil
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
	if len(media) == 0 && strings.TrimSpace(simg.Body) == "" {
		return 0, fmt.Errorf("sbody is required for text stories")
	}
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

const storyAuthorColumns = `COALESCE(pr.nick,''),COALESCE(pr.main_pic,''),COALESCE(pr.gender,0),COALESCE(pr.area,0),COALESCE(DATE_FORMAT(pr.birth,'%Y-%m-%d'),''),COALESCE(pr.age,'')`
const storyListColumns = `SELECT s.idx,s.uid,` + storyAuthorColumns + `,s.stat,s.body,s.qt_like,s.qt_view,s.price,s.str_img,s.at_create`
const storyProfileJoin = ` LEFT JOIN prf_info pr ON pr.uid=s.uid `

func (p *StoryDB) GetDefStoryList(viewer, owner uint64, page, limit int) (*[]ptc.StoryListResp, error) {
	if page < 1 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid pagination")
	}
	rows, err := p.conndb.Query(storyListColumns+` FROM story s `+storyProfileJoin+` WHERE s.uid=? AND (s.stat IN (0,1,2) OR (s.uid=? AND s.uid<>0 AND s.stat IN (3,5)))
 ORDER BY s.at_create DESC,s.idx DESC LIMIT ? OFFSET ?`, owner, viewer, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	stories, err := p.readStoryList(rows, viewer)
	if err != nil {
		return nil, err
	}
	list := make([]ptc.StoryListResp, 0, len(stories))
	for _, story := range stories {
		item := ptc.StoryListResp{Idx: story.Idx, Uid: story.Uid, Nick: story.Nick, MainPic: story.MainPic, Gender: story.Gender, Area: story.Area, Age: story.Age, Stat: story.Stat, AtCreate: story.AtCreate, CntCount: story.ContentTotal, IsLocked: story.IsLocked, GoodByMe: story.GoodByMe}
		if story.IsLocked == 0 && len(story.StrImg) > 0 {
			raw, err := json.Marshal(story.StrImg)
			if err != nil {
				return nil, err
			}
			first, _, err := GetFirstPicUrl(string(raw))
			if err != nil {
				return nil, err
			}
			item.CntUrl, item.CntType, item.CntThumb = first.URL, first.Type, first.Thumb
		}
		list = append(list, item)
	}
	return &list, nil
}

func (p *StoryDB) storyAccess(story *ptc.StoryDetailResp, viewerUID uint64) (bool, error) {
	owner := viewerUID != 0 && viewerUID == story.Uid
	if !storyVisible(story, viewerUID) {
		return false, sql.ErrNoRows
	}

	// Response flags describe actual relationships, independent of visibility.
	story.IsPaid, story.IsFollowed = 0, 0
	story.GoodByMe = 0
	if viewerUID != 0 {
		followed, err := p.IsFollowed(viewerUID, story.Uid)
		if err != nil {
			return false, err
		}
		paid, err := p.IsPcsPaid(story.Idx, viewerUID)
		if err != nil {
			return false, err
		}
		story.IsFollowed, story.IsPaid = followed, paid
		if err := p.conndb.QueryRow("SELECT EXISTS(SELECT 1 FROM story_like WHERE story_idx=? AND uid=? AND stat=1)", story.Idx, viewerUID).Scan(&story.GoodByMe); err != nil {
			return false, err
		}
	}

	// Ownership grants access without implying a follow or purchase relationship.
	locked := false
	if !owner {
		switch story.Stat {
		case ptc.StoryStatFollower:
			locked = story.IsFollowed == 0
		case ptc.StoryStatPaid:
			locked = story.IsPaid == 0
		}
	}

	story.IsLocked = 0
	if locked {
		story.IsLocked = 1
	}
	return locked, nil
}

// ApplyStoryAccess personalizes a request-local copy; never write it back to cache.
func (p *StoryDB) ApplyStoryAccess(story *ptc.StoryDetailResp, viewerUID uint64) error {
	locked, err := p.storyAccess(story, viewerUID)
	if err != nil {
		return err
	}
	if locked {
		story.Body = ""
		story.CommentList = nil
		story.CmtCount = 0
	}
	return nil
}

func (p *StoryDB) fillStoryDetail(story *ptc.StoryDetailResp, raw json.RawMessage, viewerUID uint64) error {
	// Reject hidden stories before loading comments or media.
	locked, err := p.storyAccess(story, viewerUID)
	if err != nil {
		return err
	}
	if err := p.fillStoryContent(story, raw); err != nil {
		return err
	}
	if locked {
		story.Body = ""
		story.CommentList, story.CmtCount = nil, 0
	}
	return nil
}

func (p *StoryDB) fillStoryContent(story *ptc.StoryDetailResp, raw json.RawMessage) error {
	var media ptc.StoryStrImg
	if err := json.Unmarshal(raw, &media); err != nil {
		return fmt.Errorf("invalid media JSON: %w", err)
	}

	cmtlist, tcount, err := p.GetStrComment2Low(int64(story.Idx))
	if err != nil {
		return err
	}
	story.ContentTotal = len(media)
	story.StrImg = media
	story.CmtCount = tcount
	if tcount > 0 {
		story.CommentList = *cmtlist
	}

	flwCount, err := p.GetFollowerCount(story.Uid)
	if err != nil {
		// return err
		story.FlwCount = 0
	} else {
		story.FlwCount = flwCount
	}
	return nil
}

// GetCondStoryList builds a filtered story feed.
// stat filter comes from conds (controller): 0=PUB, 1=FLW, 2=PAY, 3=PRV, 6=ALL(query only); 4/5 excluded from feed
func (p *StoryDB) GetCondStoryList(uid uint64, conds []string, orderQuery string, args []interface{}) (*[]ptc.StoryDetailResp, int, error) {
	// The caller appends LIMIT/OFFSET after the filter arguments.
	if len(args) < 2 {
		return nil, 0, fmt.Errorf("missing story pagination arguments")
	}
	filterArgs := append([]interface{}{}, args[:len(args)-2]...)
	conds = append(append([]string{}, conds...), "(stat IN (0,1,2) OR (stat=3 AND uid=? AND uid<>0))")
	filterArgs = append(filterArgs, uid)
	args = append(append([]interface{}{}, filterArgs...), args[len(args)-2:]...)
	cntQuery := `SELECT COUNT(1)
					FROM story`
	if len(conds) > 0 {
		// area(0-12), gender(0,1), type(0-4), stat(0-3)
		cntQuery += " WHERE " + strings.Join(conds, " AND ")
	}

	var totalCount int
	if err := p.conndb.QueryRow(cntQuery, filterArgs...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	base := `SELECT * FROM story`
	if len(conds) > 0 {
		base += " WHERE " + strings.Join(conds, " AND ")
	}
	query := storyListColumns + " FROM (" + base + ") s " + storyProfileJoin + " ORDER BY " + strings.NewReplacer("story.idx", "s.idx", "at_create", "s.at_create", "idx DESC", "s.idx DESC", "idx ASC", "s.idx ASC").Replace(orderQuery) + " LIMIT ? OFFSET ?"
	// fmt.Println("query", query, " args ", args)
	rows, err := p.conndb.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	stories, err := p.readStoryList(rows, uid)
	if err != nil {
		return nil, 0, err
	}
	return &stories, totalCount, nil
}

// Purchase errors preserve validation and revoked entitlements.
var ErrStoryPurchaseInvalid = errors.New("story is not available for purchase")
var ErrStoryPurchaseNotPaid = errors.New("only paid stories (stat=2) are available for purchase")
var ErrStoryPurchaseRevoked = errors.New("story purchase has been revoked")

// BuyStory completes purchases without charging under the current no-payment policy.
// Existing pending requests are completed on retry; revoked requests remain revoked.
// Locking the story row serializes requests for the same story across instances.
func (p *StoryDB) BuyStory(pcsUid uint64, strIdx int) (int, error) {
	if pcsUid == 0 || strIdx <= 0 {
		return 0, ErrStoryPurchaseInvalid
	}
	tx, err := p.conndb.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var owner uint64
	var stat int
	if err = tx.QueryRow(`SELECT uid, stat FROM story WHERE idx = ? FOR UPDATE`, strIdx).Scan(&owner, &stat); err != nil {
		return 0, err
	}
	if owner == pcsUid {
		return 0, ErrStoryPurchaseInvalid
	}
	if stat != ptc.StoryStatPaid {
		return 0, ErrStoryPurchaseNotPaid
	}

	var status int
	err = tx.QueryRow(`SELECT psc_stat FROM str_pcs
		WHERE str_idx = ? AND pcs_uid = ? ORDER BY idx DESC LIMIT 1 FOR UPDATE`, strIdx, pcsUid).Scan(&status)
	switch {
	case err == nil:
		if status != 0 && status != 1 {
			return 0, ErrStoryPurchaseRevoked
		}
		if status == 0 {
			_, err = tx.Exec(`UPDATE str_pcs SET psc_stat = 1, at_update = NOW()
				WHERE str_idx = ? AND pcs_uid = ? AND psc_stat = 0 ORDER BY idx DESC LIMIT 1`, strIdx, pcsUid)
			if err != nil {
				return 0, err
			}
		}
		return 1, tx.Commit()
	case !errors.Is(err, sql.ErrNoRows):
		return 0, err
	}

	_, err = tx.Exec(`INSERT INTO str_pcs
		(str_idx, tid, pcs_uid, at_create, at_update, qt_payed, psc_stat)
		VALUES (?, ?, ?, NOW(), NOW(), 0, 1)`, strIdx, owner, pcsUid)
	if err != nil {
		return 0, err
	}
	return 1, tx.Commit()
}

// IsPcsPaid accepts only server-confirmed, non-revoked purchases, regardless of amount.
func (p *StoryDB) IsPcsPaid(idx int, uid uint64) (int, error) {
	var count int
	err := p.conndb.QueryRow(`SELECT COUNT(1) FROM str_pcs WHERE str_idx = ? AND pcs_uid = ? AND psc_stat = 1`, idx, uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 1, nil
	}
	return 0, nil
}

// GetPurchaseList returns stories the user purchased (str_pcs.pcs_uid), newest first.
// pcs_uid = my uid, 구매한 사람
func (p *StoryDB) GetPurchaseList(uid uint64, page int, limit int) (*[]ptc.StoryDetailResp, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var totalCount int
	cntQuery := `
		SELECT COUNT(1)
		FROM str_pcs p
		INNER JOIN story s ON s.idx = p.str_idx
		WHERE p.pcs_uid = ? AND p.psc_stat = 1 AND (s.stat IN (0,1,2) OR (s.uid = p.pcs_uid AND s.uid <> 0 AND s.stat IN (3,5)))
	`
	if err := p.conndb.QueryRow(cntQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := storyListColumns + `
		FROM str_pcs p
		INNER JOIN story s ON s.idx = p.str_idx
		` + storyProfileJoin + ` WHERE p.pcs_uid = ? AND p.psc_stat = 1 AND (s.stat IN (0,1,2) OR (s.uid = p.pcs_uid AND s.uid <> 0 AND s.stat IN (3,5)))
		ORDER BY s.at_create DESC, s.idx DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	stories, err := p.readStoryList(rows, uid)
	if err != nil {
		return nil, 0, err
	}
	return &stories, totalCount, nil
}

// StoryCacheEntry contains only viewer-independent data. ContentRevision detects
// same-second edits; 255 saturates and bypasses cache instead of wrapping.
// CachedAt bounds auxiliary-data staleness.
type StoryCacheEntry struct {
	ContentRevision uint8               `json:"cntt_rvsn"`
	Story           ptc.StoryDetailResp `json:"story"`
	UpdatedAt       sql.NullTime        `json:"updated_at"`
	CachedAt        time.Time           `json:"cached_at"`
}

func (e StoryCacheEntry) Matches(current StoryCacheEntry) bool {
	return e.ContentRevision != 0 && e.ContentRevision < 255 && e.ContentRevision == current.ContentRevision &&
		!e.CachedAt.IsZero() && time.Since(e.CachedAt) < 3*time.Minute &&
		e.Story.Idx == current.Story.Idx && e.Story.Uid == current.Story.Uid &&
		e.Story.Stat == current.Story.Stat && e.UpdatedAt.Valid == current.UpdatedAt.Valid &&
		e.UpdatedAt.Time.Equal(current.UpdatedAt.Time)
}

func (p *StoryDB) loadStoryCacheEntries(query string, args ...interface{}) ([]StoryCacheEntry, error) {
	rows, err := p.conndb.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]StoryCacheEntry, 0)
	for rows.Next() {
		var entry StoryCacheEntry
		var raw json.RawMessage
		story := &entry.Story
		var mainPic, body sql.NullString
		var price sql.NullInt64
		if err := rows.Scan(&story.Idx, &story.Uid, &story.Nick, &story.Stat, &mainPic, &body, &story.QtLike, &story.QtView, &price, &raw, &story.AtCreate, &entry.UpdatedAt, &entry.ContentRevision); err != nil {
			return nil, err
		}
		story.MainPic, story.Body, story.Price = mainPic.String, body.String, int(price.Int64)
		if err := json.Unmarshal(raw, &story.StrImg); err != nil {
			return nil, err
		}
		story.ContentTotal = len(story.StrImg)
		entry.CachedAt = time.Now()
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}

const storyCacheColumns = "SELECT idx, uid, nick, stat, main_pic, body, qt_like, qt_view, price, str_img, at_create, at_update, cntt_rvsn FROM story"

func (p *StoryDB) GetStory1000() ([]StoryCacheEntry, int, error) {
	entries, err := p.loadStoryCacheEntries(storyCacheColumns + " WHERE stat IN (0,1,2,3,5) ORDER BY at_create DESC, idx DESC LIMIT 1000")
	return entries, len(entries), err
}

func (p *StoryDB) GetStoryCacheEntry(idx int) (StoryCacheEntry, error) {
	entries, err := p.loadStoryCacheEntries(storyCacheColumns+" WHERE idx = ?", idx)
	if err != nil {
		return StoryCacheEntry{}, err
	}
	if len(entries) == 0 {
		return StoryCacheEntry{}, sql.ErrNoRows
	}
	return entries[0], nil
}

// GetStoryPage filters by owner and current viewer visibility before pagination, without loading body/media.
func (p *StoryDB) GetStoryPage(viewerUID, ownerUID uint64, page, limit int) ([]StoryCacheEntry, int, error) {
	const where = " WHERE s.uid = ? AND (s.stat IN (0,1,2) OR (s.uid = ? AND s.uid <> 0 AND s.stat IN (3,5)))"
	var total int
	if err := p.conndb.QueryRow("SELECT COUNT(1) FROM story s"+where, ownerUID, viewerUID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := p.conndb.Query("SELECT s.idx,s.uid,s.stat,s.qt_like,s.qt_view,s.price,s.at_update,s.cntt_rvsn,"+storyAuthorColumns+" FROM story s "+storyProfileJoin+where+" ORDER BY s.at_create DESC,s.idx DESC LIMIT ? OFFSET ?", ownerUID, viewerUID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	entries := make([]StoryCacheEntry, 0)
	for rows.Next() {
		var e StoryCacheEntry
		var price sql.NullInt64
		var birth, profileAge string
		var areaCode int
		if err := rows.Scan(&e.Story.Idx, &e.Story.Uid, &e.Story.Stat, &e.Story.QtLike, &e.Story.QtView, &price, &e.UpdatedAt, &e.ContentRevision, &e.Story.Nick, &e.Story.MainPic, &e.Story.Gender, &areaCode, &birth, &profileAge); err != nil {
			return nil, 0, err
		}
		e.Story.Price = int(price.Int64)
		e.Story.Area = ptc.GetAreaName(areaCode)
		e.Story.Age = calcStoryAge(birth, profileAge)
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// GetFlwStoryList returns stories of users that uid is following (user_follow.stat=1), newest first.
func (p *StoryDB) GetFlwStoryList(uid uint64, page int, limit int) (*[]ptc.StoryDetailResp, int, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var totalCount int
	cntQuery := `
		SELECT COUNT(1)
		FROM user_follow f
		INNER JOIN story s ON s.uid = f.followee_uid
		WHERE f.uid = ? AND f.stat = 1 AND (s.stat IN (0,1,2) OR (s.uid = f.uid AND s.uid <> 0 AND s.stat IN (3,5)))
	`
	if err := p.conndb.QueryRow(cntQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := storyListColumns + `
		FROM user_follow f
		INNER JOIN story s ON s.uid = f.followee_uid
		` + storyProfileJoin + ` WHERE f.uid = ? AND f.stat = 1 AND (s.stat IN (0,1,2) OR (s.uid = f.uid AND s.uid <> 0 AND s.stat IN (3,5)))
		ORDER BY s.at_create DESC, s.idx DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	stories, err := p.readStoryList(rows, uid)
	if err != nil {
		return nil, 0, err
	}
	return &stories, totalCount, nil
}

func (p *StoryDB) GetStoryList(viewerUID uint64, nPage int, nLimit int) (*[]ptc.StoryDetailResp, int, error) {
	cntQuery := `SELECT COUNT(1) FROM story WHERE (stat IN (0,1,2) OR (uid = ? AND uid <> 0 AND stat IN (3,5)))`
	var totalCount int
	if err := p.conndb.QueryRow(cntQuery, viewerUID).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := storyListColumns + ` FROM story s ` + storyProfileJoin + `
 WHERE (s.stat IN (0,1,2) OR (s.uid = ? AND s.uid <> 0 AND s.stat IN (3,5)))
 ORDER BY s.at_create DESC,s.idx DESC LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, viewerUID, nLimit, (nPage-1)*nLimit)
	if err != nil {
		return nil, 0, err
	}
	stories, err := p.readStoryList(rows, viewerUID)
	if err != nil {
		return nil, 0, err
	}
	return &stories, totalCount, nil
}

func (p *StoryDB) GetStoryDetail(uid uint64, strIdx int) (*ptc.StoryDetailResp, error) {
	query := storyListColumns + ` FROM story s ` + storyProfileJoin + ` WHERE s.idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var (
		story   ptc.StoryDetailResp
		raw     json.RawMessage
		mainPic sql.NullString
		body    sql.NullString
		price   sql.NullInt64
	)
	var birth, profileAge string
	var areaCode int
	err := row.Scan(&story.Idx, &story.Uid, &story.Nick, &mainPic, &story.Gender, &areaCode, &birth, &profileAge, &story.Stat, &body, &story.QtLike, &story.QtView, &price, &raw, &story.AtCreate)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	story.Price = int(price.Int64)
	story.Area = ptc.GetAreaName(areaCode)
	story.Age = calcStoryAge(birth, profileAge)
	if mainPic.Valid {
		story.MainPic = mainPic.String
	}
	if body.Valid {
		story.Body = body.String
	}

	if err := p.fillStoryDetail(&story, raw, uid); err != nil {
		return nil, err
	}

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
	query := `SELECT qt_like FROM story WHERE idx = ?`
	row := p.conndb.QueryRow(query, strIdx)
	var qtGood int
	err := row.Scan(&qtGood)
	if err != nil {
		return 0, err
	}
	return qtGood, nil
}

// UpdateStoryView counts at most once per viewer/story in a rolling 24-hour
// window. The ledger and counter commit together, including on client retries.
// counted=false means the previous counted view is still inside the window.
func (p *StoryDB) UpdateStoryView(uid uint64, strIdx int) (counted bool, err error) {
	if uid == 0 || strIdx <= 0 {
		return false, fmt.Errorf("invalid viewer or story ID")
	}
	tx, err := p.conndb.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// The unique key serializes concurrent requests for this viewer/story.
	// Do not use RowsAffected here: MySQL clientFoundRows changes its semantics.
	_, err = tx.Exec(`INSERT INTO story_view_history (story_idx, uid, last_counted_at)
		VALUES (?, ?, NULL) ON DUPLICATE KEY UPDATE uid = uid`, strIdx, uid)
	if err != nil {
		return false, err
	}

	result, err := tx.Exec(`UPDATE story_view_history SET last_counted_at = UTC_TIMESTAMP(6)
		WHERE story_idx = ? AND uid = ?
		AND (last_counted_at IS NULL OR last_counted_at <= UTC_TIMESTAMP(6) - INTERVAL 24 HOUR)`, strIdx, uid)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}

	// Views do not change content; leave at_update unchanged to retain raw cache hits.
	result, err = tx.Exec(`UPDATE story SET qt_view = COALESCE(qt_view, 0) + 1 WHERE idx = ?`, strIdx)
	if err != nil {
		return false, err
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		// Even if the commit outcome is unknown, retrying checks the same ledger.
		return false, err
	}
	return true, nil
}

// storyMutationResult distinguishes a missing/foreign story from an owned no-op
// (for example, identical content after the revision counter has saturated).
func (p *StoryDB) storyMutationResult(result sql.Result, idx int, actor uint64) (int64, error) {
	affected, err := result.RowsAffected()
	if err != nil || affected != 0 {
		return affected, err
	}
	var found int
	err = p.conndb.QueryRow("SELECT 1 FROM story WHERE idx=? AND uid=? AND uid<>0", idx, actor).Scan(&found)
	return 0, err
}

func (p *StoryDB) UpdateStoryStat(strIdx int, actorUID uint64, stat int) (int64, error) {
	query := `UPDATE story SET stat = ?, cntt_rvsn = LEAST(cntt_rvsn + 1, 255), at_update = NOW() WHERE idx = ? AND uid = ? AND uid <> 0`
	result, err := p.conndb.Exec(query, stat, strIdx, actorUID)
	if err != nil {
		return 0, err
	}
	return p.storyMutationResult(result, strIdx, actorUID)
}

func (p *StoryDB) UpdatePriceStat(strIdx int, actorUID uint64, stat, price int) (int64, error) {
	if !ptc.IsValidStoryStat(stat) || price < 0 || price > 2147483647 {
		return 0, fmt.Errorf("invalid story status or price")
	}
	query := `UPDATE story SET stat = ?, price = ?, cntt_rvsn = LEAST(cntt_rvsn + 1, 255), at_update = NOW() WHERE idx = ? AND uid = ? AND uid <> 0`
	result, err := p.conndb.Exec(query, stat, price, strIdx, actorUID)
	if err != nil {
		return 0, err
	}
	return p.storyMutationResult(result, strIdx, actorUID)
}

func (p *StoryDB) UpdateStrBody(strIdx int, actorUID uint64, body string) (int64, error) {
	query := `UPDATE story SET body = ?, cntt_rvsn = LEAST(cntt_rvsn + 1, 255), at_update = NOW() WHERE idx = ? AND uid = ? AND uid <> 0`
	result, err := p.conndb.Exec(query, body, strIdx, actorUID)
	if err != nil {
		return 0, err
	}
	return p.storyMutationResult(result, strIdx, actorUID)
}

func (p *StoryDB) SetStrComment(cmt *ptc.StrComment) (int64, error) {
	if cmt.Wuid == 0 {
		return 0, ErrStoryLocked
	}
	if _, err := p.RequireStoryRead(cmt.Wuid, cmt.StrIdx); err != nil {
		return 0, err
	}
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
func (p *StoryDB) GetStrCmtDetail(viewer uint64, cmtIdx, page, limit int) (*[]ptc.StrComment, int, error) {
	if _, err := p.RequireStoryRead(viewer, cmtIdx); err != nil {
		return nil, 0, err
	}
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
	          ORDER BY at_create DESC, idx DESC
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

func (p *StoryDB) GetStrPicList(strIdx int, actorUID uint64) (json.RawMessage, json.RawMessage, error) {
	query := `SELECT str_img, str_imgbak FROM story WHERE idx = ? AND uid = ? AND uid <> 0`
	row := p.conndb.QueryRow(query, strIdx, actorUID)

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

func (p *StoryDB) DeleteStrPic(strIdx int, actorUID uint64, pic, picback json.RawMessage) (int64, error) {
	var media ptc.StoryStrImg
	if err := json.Unmarshal(pic, &media); err != nil {
		return 0, err
	}
	query := `UPDATE story SET str_img = ?, str_imgbak = ?, type = ?, cntt_rvsn = LEAST(cntt_rvsn + 1, 255), at_update = NOW() WHERE idx = ? AND uid = ? AND uid <> 0`
	result, err := p.conndb.Exec(query, pic, picback, MediaMType(media), strIdx, actorUID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
	// return 0, nil
}

func (p *StoryDB) UpdateStrStatComment(cmtIdx int, actor uint64, admin bool, stat int) (int64, error) {
	parent, err := p.GetCommentStoryID(cmtIdx, actor, admin)
	if err != nil {
		return 0, err
	}
	if _, err = p.RequireStoryRead(actor, parent); err != nil {
		return 0, err
	}
	query := `UPDATE str_cmt SET stat = ?, at_update = NOW() WHERE idx = ? AND (wuid = ? OR ?) AND ? <> 0`
	result, err := p.conndb.Exec(query, stat, cmtIdx, actor, admin, actor)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) UpdateStrBodyComment(cmtIdx int, actor uint64, admin bool, body string) (int64, error) {
	parent, err := p.GetCommentStoryID(cmtIdx, actor, admin)
	if err != nil {
		return 0, err
	}
	if _, err = p.RequireStoryRead(actor, parent); err != nil {
		return 0, err
	}
	query := `UPDATE str_cmt SET body = ?, at_update = NOW() WHERE idx = ? AND (wuid = ? OR ?) AND ? <> 0`
	result, err := p.conndb.Exec(query, body, cmtIdx, actor, admin, actor)
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
			if _, err = tx.Exec(`UPDATE story SET qt_like = COALESCE(qt_like, 0) + 1, at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
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
			if _, err = tx.Exec(`UPDATE story SET qt_like = GREATEST(COALESCE(qt_like, 0) - 1, 0), at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
				return false, 0, err
			}
			stat = 0
		} else {
			if _, err = tx.Exec(`UPDATE story_like SET stat = 1, at_update = NOW() WHERE story_idx = ? AND uid = ?`, storyIdx, uid); err != nil {
				return false, 0, err
			}
			if _, err = tx.Exec(`UPDATE story SET qt_like = COALESCE(qt_like, 0) + 1, at_update = NOW() WHERE idx = ?`, storyIdx); err != nil {
				return false, 0, err
			}
			stat = 1
		}
	}

	var likeCount int
	if err = tx.QueryRow(`SELECT COALESCE(qt_like, 0) FROM story WHERE idx = ?`, storyIdx).Scan(&likeCount); err != nil {
		return false, 0, err
	}

	if err = tx.Commit(); err != nil {
		return false, 0, err
	}
	rollback = false
	return stat == 1, likeCount, nil
}

func (p *StoryDB) IsFollowed(viewerUID, authorUID uint64) (int, error) {
	query := `SELECT COUNT(1) FROM user_follow WHERE uid = ? AND followee_uid = ? AND stat = 1`
	var count int
	err := p.conndb.QueryRow(query, viewerUID, authorUID).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 1, nil
	}
	return 0, nil
}

func (p *StoryDB) SetFollow(followerUid, followeeUid uint64) (int64, error) {
	query := `
		INSERT INTO user_follow (uid, followee_uid, stat, at_create, at_update)
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
	query := `UPDATE user_follow SET stat = 0, at_update = NOW() WHERE uid = ? AND followee_uid = ? AND stat = 1`
	result, err := p.conndb.Exec(query, followerUid, followeeUid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *StoryDB) GetFollowerCount(uid uint64) (int, error) {
	query := `SELECT COUNT(1) FROM user_follow WHERE followee_uid = ? AND stat = 1`
	var count int
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

// accountUserInfoTable qualifies the account table because this connection uses sdb.
func (p *StoryDB) accountUserInfoTable() (string, error) {
	if p.cfg == nil {
		return "", fmt.Errorf("account database configuration is missing")
	}
	name, ok := p.cfg.DB["adb"]["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("account database name is missing")
	}
	return "`" + strings.ReplaceAll(name, "`", "``") + "`.`user_info`", nil
}

func (p *StoryDB) GetFollowerList(uid uint64, page int, limit int) (*[]ptc.FollowUserItem, int, error) {
	accountTable, err := p.accountUserInfoTable()
	if err != nil {
		return nil, 0, err
	}

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
		SELECT uf.uid, COALESCE(u.nick, ''), COALESCE(u.thmb_pic, ''), uf.at_update
		FROM user_follow uf
		LEFT JOIN ` + accountTable + ` u ON u.uid = uf.uid
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
	accountTable, err := p.accountUserInfoTable()
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(1)
		FROM user_follow uf
		WHERE uf.uid = ? AND uf.stat = 1
	`
	var totalCount int
	if err := p.conndb.QueryRow(countQuery, uid).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT uf.followee_uid, COALESCE(u.nick, ''), COALESCE(u.thmb_pic, ''), uf.at_update
		FROM user_follow uf
		LEFT JOIN ` + accountTable + ` u ON u.uid = uf.followee_uid
		WHERE uf.uid = ? AND uf.stat = 1
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
