# Story Database (StoryDB) Documentation

## Overview
StoryDB handles story sharing functionality with image uploads, comments, and status management. Stories can have multiple images stored as JSON, support comments, and include soft delete with backup capabilities.

## Database Schema

### story Table
Main table for storing user stories with images.

**Columns:**
- `idx` (int unsigned, PK, AUTO_INCREMENT): Unique story identifier
- `uid` (bigint, NOT NULL): User ID who created the story
- `nick` (varchar 20, NOT NULL): Nickname of the story creator
- `body` (varchar 512, NULL): Story body text content
- `stat` (tinyint, DEFAULT 1): Story status
  - 0: Deleted
  - 1: Public
  - 2: Private
  - 3: Limited
  - 4: Reserved
- `qt_good` (int, NULL): Good/like count
- `qt_checked` (int, NULL): View count
- `str_img` (JSON, NOT NULL): Active story images as JSON object
- `str_imgbak` (JSON, NULL): Backup of deleted images
- `at_create` (datetime, NULL): Creation timestamp
- `at_update` (datetime, NULL): Last update timestamp

**Indexes:**
- PRIMARY KEY (`idx`)

**Example JSON Structure for str_img (canonical):**
```json
{
  "1": {
    "type": "img",
    "url": "https://imagedelivery.net/.../pic1/public"
  },
  "2": {
    "type": "vdo",
    "url": "https://customer-xxx.cloudflarestream.com/abc/manifest/video.m3u8",
    "thumb": "https://imagedelivery.net/.../thumb2/public"
  },
  "3": {
    "type": "img",
    "url": "https://imagedelivery.net/.../pic3/public"
  }
}
```

- 슬롯 키: `"1"` … `"5"` (업로드 순서, 1-based, **최대 5개**)
- `type`: `img` | `vdo`
- `url`: 본편 URL
- `thumb`: **vdo만** (이미지 슬롯에는 없음)
- 저장: `SaveStory` → `BuildStrImgForDB`로 위 형태로 INSERT (`ValidateStoryMedia`에서 1~5개 검증)
- 조회: list/detail `str_img`를 `StoryStrImg`로 파싱해 동일 형태 반환
- 삭제: 슬롯 제거 후 `1..N` 재인덱싱, 삭제분은 `str_imgbak`으로 이동

#### BuildStrImgForDB
- media 검증(개수 ≤5, 키 1..5) → 키 오름차순 `1..N` 재번호
- img: `{type,url}` / vdo: `{type,url,thumb}`
- 반환: `str_img` JSON + `type`(0=img, 1=vdo, 2=both)

### str_pcs Table
스토리 구매 기록 테이블.

**Columns:**
- `idx` (int, PK): 구매 레코드 ID
- `str_idx` (int, NOT NULL): 구매한 스토리 `story.idx`
- `tid` (bigint, NULL): 판매자 UID
- `pcs_uid` (bigint, NOT NULL): 구매자 UID
- `at_create` / `at_update` (date, NULL)
- `qt_payed` (bigint, NULL): 결제 금액

### str_cmt Table
Table for storing comments on stories.

**Columns:**
- `idx` (int unsigned, PK, AUTO_INCREMENT): Unique comment identifier
- `str_idx` (int unsigned, NOT NULL): Story index (foreign key reference)
- `wuid` (bigint, NOT NULL): User ID who wrote the comment
- `nick` (varchar 20, NULL): Nickname of comment writer
- `thumb_url` (varchar 256, NULL): Writer thumbnail URL
- `wgender` (varchar 2, NULL): Writer gender
- `wage` (varchar 45, NULL): Writer age text
- `warea` (varchar 45, NULL): Writer area
- `body` (varchar 256, NULL): Comment content
- `stat` (tinyint, NULL): Comment status
  - 0: Default/active
  - 1: Private
  - 2-3: Reserved
  - 4: Deleted
- `at_create` (datetime, DEFAULT CURRENT_TIMESTAMP): Creation timestamp
- `at_update` (datetime, DEFAULT CURRENT_TIMESTAMP): Last update timestamp

**Indexes:**
- PRIMARY KEY (`idx`)
- UNIQUE KEY (`idx`)

## Repository Functions

### Connection Management
```go
func NewStoryDB(cf *conf.Config, root *Repositories) (IRepository, error)
func (p *StoryDB) Start() error
func (p *StoryDB) Terminate()
func (p *StoryDB) Close() error
func (p *StoryDB) Ping() error
func (p *StoryDB) heartbeat() // Goroutine: pings every 2 minutes
```

### Story Functions

#### SetStory
```go
func (p *StoryDB) SetStory(simg *ptl.StoryImage) (int64, error)
```
Creates new story with uid, nickname, body, status, and image JSON.

**Parameters:**
- `simg.Uid` (uint64): User ID
- `simg.Nick` (string): Nickname
- `simg.Body` (string): Story text (max 512 chars)
- `simg.Stat` (int): Status (0-4)
- `simg.StrImg` ([]byte): JSON marshaled image URLs

**Returns:** Last insert ID

#### GetPrfPicWaitingList
```go
func (p *StoryDB) GetPrfPicWaitingList(page, limit int) ([]ptc.PrfPicWaitingItem, int, error)
```
Admin review list: unpivots `prf_info.sub_pic1`~`sub_pic5`, filters JSON `stat IN (1,2)`, returns paged items (`uid`, `nick`, `slot`, `url`, `stat`) and `total_count`. Limit capped at 100. Slot JSON format: `{"url":"...","stat":N}` (no idx).

#### SetPrfPicStat
```go
func (p *StoryDB) SetPrfPicStat(uid uint64, slot, newStat int) (int64, error)
```
Updates `sub_pic{slot}` JSON `$.stat` via `JSON_SET`. Only rows with current stat 1 or 2 are updated. Used to approve (stat=3).

#### UploadPrfPic / DeletePrfPic
- `UploadPrfPic`: slice 순서대로 `sub_pic1..N`에 `{"url","stat"}` 저장, 나머지 슬롯 NULL
- `DeletePrfPic(slot)`: 해당 컬럼 삭제 후 뒤 슬롯을 앞으로 당김 (JSON 재번호 불필요 — 순번은 컬럼)

#### GetStoryList
```go
func (p *StoryDB) GetStoryList(uid uint64) (*[]ptl.StoryListResp, error)
```
Retrieves user's public stories (stat IN (1, 0)).

**Returns:**
- Array of stories with: idx, nick, str_img (JSON), at_create
- Ordered by at_create DESC

#### GetCondStoryList
```go
func (p *StoryDB) GetCondStoryList(conds []string, orderQuery string, args []interface{}) (*map[int]string, error)
```
Retrieves condition-based story map (`idx -> first image url`) with dynamic filters.

**Query behavior:**
- Base query has no hardcoded `stat` filter — `conds` from controller drive WHERE (e.g. `stat = ?`)
- Extra filters are appended as `WHERE ... AND ...` when `conds` is not empty
- Sorting always uses `ORDER BY <orderQuery>`
- Pagination uses `LIMIT ? OFFSET ?` from `args`
- `area=0`(all) is omitted in controller so all regions are queried

**Notes:**
- `stories` map is initialized before scan loop
- `orderQuery` should be built from whitelist mapping (e.g. `protocol.GetOrderQuery`) to avoid invalid SQL
- `str_img`는 `ParseStoryMedia`/`GetFirstPicUrl`로 파싱. 최소 슬롯의 img→url, vdo→thumb 반환.
  파싱 실패 시 raw DB 값 fallback.

#### str_img 포맷
- 슬롯 키 `"1"`, `"2"`, ... (업로드 순서, 1-based)
- 값: `{"type":"img"|"vdo","url":"...","thumb":"..."}` (`thumb`은 vdo만)
- 입력(upload/create)에서만 생성. 조회 시 변환 없음

#### BuyStory
```go
func (p *StoryDB) BuyStory(pcsUid uint64, strIdx int) error
```
`story`에서 판매자 `tid` 조회 후 `str_pcs`에 INSERT (`pcs_uid`=구매자, `qt_payed=0`). 결제 로직은 컨트롤러 TODO.

#### GetPurchaseList
```go
func (p *StoryDB) GetPurchaseList(uid uint64, page int, limit int) (*[]ptc.StoryListResp, int, error)
```
`str_pcs.pcs_uid = uid`인 구매 스토리를 `story`와 INNER JOIN 후 `story.at_create DESC`로 페이징.

**Returns:** `StoryListResp` 슬라이스(대표 미디어 포함), total_count

#### GetStory
```go
func (p *StoryDB) GetStory(strIdx int) (*ptl.StoryDetailResp, error)
```
Retrieves single story detail by index. `str_img`는 DB 저장값 그대로 반환.

**Returns:** nick, body, str_img (unified media JSON), at_create

#### ParseStoryMedia / GetFirstPicUrl (`models/story_media.go`)
- `ParseStoryMedia`: 통합 media JSON → `map[string]StoryMediaItem` (그 외 포맷 거부)
- `GetFirstPicUrl`: 리스트용 대표 URL (vdo면 thumb)
- `MediaMType`: story.type 계산 (0=img, 1=vdo, 2=both)
- `ReIdxAfterDel`: 슬롯 삭제 후 남은 키를 1..N 재번호

#### GetStoryViewCount
```go
func (p *StoryDB) GetStoryViewCount(strIdx int) (int, error)
```
`SELECT qt_view FROM story WHERE idx = ?` — 현재 조회수 반환.

#### GetStoryLikeCount
```go
func (p *StoryDB) GetStoryLikeCount(strIdx int) (int, error)
```
`SELECT qt_good FROM story WHERE idx = ?` — 현재 좋아요 수 반환.

#### UpdateStoryView
```go
func (p *StoryDB) UpdateStoryView(strIdx int) error
```
조회수(`qt_view`)를 +1 합니다.

- SQL: `SET qt_view = LAST_INSERT_ID(COALESCE(qt_view, 0) + 1)` — 원자적 +1
- 대상 행 없으면 `"story not found"`
- API(`UpdateStoryViewCnt`) 응답에는 증가 후 값을 포함하지 않음

#### UpdateStoryStat
```go
func (p *StoryDB) UpdateStoryStat(strIdx int, stat int) (int64, error)
```
`UPDATE story SET stat=?, at_update=NOW() WHERE idx=?` — 상태 갱신.

**Returns:** Rows affected count

#### UpdateStrBody
```go
func (p *StoryDB) UpdateStrBody(strIdx int, body string) (int64, error)
```
`UPDATE story SET body=?, at_update=NOW() WHERE idx=?` — 본문 갱신.

**Returns:** Rows affected count

#### GetStrPicList
```go
func (p *StoryDB) GetStrPicList(strIdx int) (json.RawMessage, json.RawMessage, error)
```
Retrieves str_img and str_imgbak JSON for a story.

**Returns:** 
- str_img: Active images JSON
- str_imgbak: Backup images JSON

#### DeleteStrPic
```go
func (p *StoryDB) DeleteStrPic(strIdx int, pic, picback json.RawMessage) (int64, error)
```
컨트롤러가 재구성한 `str_img`/`str_imgbak` JSON을 DB에 저장합니다.

**컨트롤러 처리:**
1. `GetStrPicList`로 현재 JSON 조회
2. `pic_idx` 슬롯을 bak으로 복사
3. `ReIdxAfterDel`로 남은 슬롯 1..N 재번호
4. `DeleteStrPic`으로 UPDATE

**Returns:** Rows affected count

### Comment Functions

#### SetStrComment
```go
func (p *StoryDB) SetStrComment(cmt *ptl.StrComment) (int64, error)
```
Creates new comment on a story.

**Parameters:**
- `cmt.StrIdx` (int): Story index
- `cmt.Wuid` (uint64): Writer user ID
- `cmt.Nick` (string): Writer nickname
- `cmt.ThumbUrl` (string): Writer thumbnail
- `cmt.WGender` (string): Writer gender
- `cmt.WAge` (string): Writer age
- `cmt.WArea` (string): Writer area
- `cmt.Body` (string): Comment text (max 256 chars)
- `cmt.Stat` (int): Comment status

**Returns:** Last insert ID

#### GetStrCmtDetail
```go
func (p *StoryDB) GetStrCmtDetail(cmtIdx int, page int) (*[]ptl.StrComment, int, error)
```
Retrieves paginated active comments (stat IN (0, 1)) for a story.

**Returns:**
- Array of comments with: idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update
- Total active comment count for the story (same filter: stat IN (0, 1))
- Ordered by at_create DESC

#### GetStrCommentList
```go
func (p *StoryDB) GetStrCommentList(strIdx int64) (*[]ptl.StrComment, error)
```
Alternative function to get comment list (same as GetStrCmtDetail).

#### UpdateStrStatComment
```go
func (p *StoryDB) UpdateStrStatComment(cmtIdx int, stat int) (int64, error)
```
Updates comment status.

**Returns:** Rows affected count

#### UpdateStrBodyComment
```go
func (p *StoryDB) UpdateStrBodyComment(cmtIdx int, body string) (int64, error)
```
Updates comment body content.

**Returns:** Rows affected count

## Usage Examples

### Creating a Story
```go
storyImg := &ptl.StoryImage{
    Uid: 12345,
    Nick: "testnick",
    Body: "This is my story",
    Stat: 1, // Public
    StrImg: jsonImageData, // {"1": "url1", "2": "url2"}
}

lastID, err := storyDB.SetStory(storyImg)
```

### Getting User's Stories
```go
stories, err := storyDB.GetStoryList(userId)
// Returns: [{idx: 1, nick: "user", str_img: {...}, at_create: "2026-02-02"}, ...]
```

### Deleting a Picture
```go
// 1. Get current images
strImg, strImgBak, err := storyDB.GetStrPicList(storyIdx)

// 2. Unmarshal JSON
mImg := make(map[string]string)
mImgBak := make(map[string]string)
json.Unmarshal(strImg, &mImg)
json.Unmarshal(strImgBak, &mImgBak)

// 3. Move image to backup
picIdx := "2" // Delete picture with index 2
mImgBak[picIdx] = mImg[picIdx]
delete(mImg, picIdx)

// 4. Marshal and update
mImgBytes, _ := json.Marshal(mImg)
mImgBakBytes, _ := json.Marshal(mImgBak)
affected, err := storyDB.DeleteStrPic(storyIdx, mImgBytes, mImgBakBytes)
```

### Adding a Comment
```go
comment := &ptl.StrComment{
    StrIdx: storyIdx,
    Wuid: userId,
    Nick: "commenter",
    Body: "Great story!",
    Stat: 0, // Active
}

commentID, err := storyDB.SetStrComment(comment)
```

## Key Features

### JSON Image Storage
- **Flexible Structure**: Numeric indexes allow easy addition/deletion
- **Multiple Images**: Up to 5 images per story supported
- **URL Storage**: Direct Cloudflare delivery URLs

### Backup System
- **Soft Delete**: Images moved to str_imgbak instead of permanent deletion
- **Recovery**: Deleted images can be restored from backup
- **Audit Trail**: Maintains history of deleted content

### Status Management
- **Story Status**: Public (1), Private (2), Limited (3), Deleted (0)
- **Comment Status**: Active (0), Private (1), Deleted (4)
- **Filtered Queries**: Automatically excludes deleted content

### Performance
- **Connection Pooling**: 50 max connections, 25 idle, 30-minute lifetime, 5-minute idle timeout
- **Heartbeat**: 2-minute health checks
- **Indexed Queries**: Fast lookups by idx and uid
- **Automatic Timestamps**: at_create and at_update managed by DB

## Integration

### Controller Integration
- **StoryController** uses StoryDB for all story operations
- File uploads: 본편 **utils.UpTotalCldFlrThumb()** (Images/Stream), 썸네일 **utils.UploadCldFlr()** before DB storage
- Form data parsing in middleware provides uid, body, nick, stat

### Cloudflare Integration
- Images → Cloudflare Images API (`variants[0]`)
- Video → Cloudflare Stream API (`playback.hls`)
- Returns delivery URLs stored in str_img JSON
- Configuration: cfg.Server.CfId, cfg.Server.CfToken

### Protocol Types
```go
type StoryImage struct {
    Uid    uint64
    Nick   string
    Body   string
    Stat   int
    StrImg []byte // JSON marshaled
}

type StoryListResp struct {
    Idx      int
    Nick     string
    StrImg   json.RawMessage
    AtCreate time.Time
}

type StrComment struct {
    Idx      int
    StrIdx   int
    Wuid     uint64
    Nick     string
    Body     string
    Stat     int
    AtCreate time.Time
}
```

## Security Considerations

### Input Validation
- **Body Length**: 512 chars for stories, 256 for comments
- **Required Fields**: uid, nick, stat must be present
- **Status Values**: Validated against allowed range (0-4)

### SQL Injection Prevention
- **Prepared Statements**: All queries use parameterized statements
- **Type Safety**: Strong typing for all parameters

### Data Privacy
- **Status-based Access**: Only public/active content returned by default
- **Soft Delete**: Sensitive content preserved for audit

## Monitoring and Maintenance

### Health Checks
- Automatic ping every 2 minutes via heartbeat()
- Connection pool monitoring
- Error logging for failed queries

### Database Maintenance
- Regular backup of story and comment tables
- Monitor str_img and str_imgbak JSON sizes
- Clean up old deleted stories (stat=0) periodically

## 2026-04 Update (Social Extension)

### Added DB templates in code comments (`models/story_db.go`)
- `story_like`: like/unlike ledger (`stat`) with unique `(story_idx, uid)`
- `user_follow`: follow/unfollow ledger (`stat`) with unique `(follower_uid, followee_uid)`
- `user_block`: block/unblock ledger (`stat`) with unique `(blocker_uid, blocked_uid)`
- `str_cutout`: story feed cutout ledger (`stat`) with unique `(uid, tid)` and profile snapshot columns

### Added repository methods
- `ToggleStoryLikeTx()`: transactional like toggle + `story.qt_good` sync
- `SetFollow(follower,followee)`: `user_follow` INSERT ON DUPLICATE KEY UPDATE `stat=1`
- `SetUnfollow(follower,followee)`: `user_follow.stat=0` 업데이트
- `GetFollowerList(uid,page,limit)`: `followee_uid=uid AND stat=1`, `FollowUserItem` + total_count
- `GetFollowingList(uid,page,limit)`: `follower_uid=uid AND stat=1`, `FollowUserItem` + total_count
- `SetBlock()`, `SetUnblock()`, `GetBlockList()`, `IsBlockedPair()`
- `GetStoryOwnerUID()` for ownership lookup in block checks
- `SetCutoutUser()`: upsert 방식으로 cutout 활성화 및 대상 프로필 스냅샷 갱신
- `UnsetCutoutUser()`: cutout 비활성화(`stat=0`)
- `GetCutoutCount()`, `GetCutoutList()`: 활성 cutout 개수/페이지 목록 조회
- `GetActiveCutoutTIDs()`: 조건부 스토리 조회 시 제외 대상 UID 목록 조회

## Future Enhancements
- [ ] Implement view tracking (`qt_checked`)
- [ ] Add story expiration feature (24-hour stories)
- [ ] Support story mentions and hashtags
- [ ] Implement story analytics

---
*Created: 2026-02-02*
*Last Updated: 2026-04-21*
*Database: MySQL 8.0+*
*Character Set: utf8mb4*
