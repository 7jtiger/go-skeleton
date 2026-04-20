package router

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"ms-gateway/common/logger"
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"
)

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*rateLimiterEntry)
	limitersMu sync.Mutex
)

func init() {
	go cleanupLimiters()
}

func cleanupLimiters() {
	for {
		time.Sleep(5 * time.Minute)
		limitersMu.Lock()
		for ip, entry := range limiters {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(limiters, ip)
			}
		}
		limitersMu.Unlock()
	}
}

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		limitersMu.Lock()
		entry, exists := limiters[ip]
		if !exists {
			entry = &rateLimiterEntry{
				limiter:  rate.NewLimiter(1, 5), // 1 request per second with a burst capacity of 5
				lastSeen: time.Now(),
			}
			limiters[ip] = entry
		}
		entry.lastSeen = time.Now()
		limitersMu.Unlock()

		if !entry.limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-Forwarded-For, Authorization, Accept, Origin, Cache-Control, X-Requested-With, OTP-Auth, X-Otp, X-Totp, X-Meta, X-Name, Siwe-Session")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Meta data를 들고 올경우 파싱해서 넘겨줌.
func (p *Router) GetReqXMeta() gin.HandlerFunc {
	return func(c *gin.Context) {
		encMetaHeader := c.GetHeader("x-meta")
		if encMetaHeader == "" {
			p.ctl.RespError(c, "Missing x-meta header", http.StatusBadRequest)
			return
		}

		metaBytes, err := utils.DecryptGCM(encMetaHeader, []byte(p.cfg.Server.BaseKey))
		if err != nil {
			p.ctl.RespError(c, "Failed to decrypt x-meta header", http.StatusBadRequest)
			return
		}

		c.Header("x-meta", string(metaBytes))

		c.Next()
	}
}

func (p *Router) CheckBlacklist() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		ip := c.ClientIP()
		bl, err := p.rdb.GetCache("blacklist_" + ip)

		if err != nil {
			if err == redis.Nil {
				return
			}
			p.ctl.RespError(c, "ServerError", http.StatusInternalServerError)
			return
		}

		if bl != "" {
			// 블랙리스트에 포함된 경우
			logger.Error("Black User Try to Access", c.ClientIP())
			p.ctl.RespError(c, "Blacklisted User", http.StatusForbidden)
			return
		}

		// bl이 빈 문자열인 경우 (캐시에 없거나 정상 사용자)
		c.Next()
	}
}

func ParsePagination(c *gin.Context, limit int) (*ptl.DefaultPaginationQuery, error) {
	var q ptl.DefaultPaginationQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		return nil, err
	}
	if q.Limit > limit {
		q.Limit = limit
	}
	return &q, nil
}

func (p *Router) Pagination(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		pagination, err := ParsePagination(c, limit)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Set("pagination", *pagination)
		c.Next()
	}
}

func (p *Router) JwtAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "No JWT Header")
			return
		}

		tokens := strings.Split(authHeader, " ")

		if len(tokens) != 2 {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid Bearer Type")
			return
		}

		acToken := tokens[1]
		acClaims, err := utils.VerifyJWTToken(acToken, p.cfg.Server.JWTSecret)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid JWT ", err.Error())
			return
		}

		if acClaims.ExpiresAt.Before(time.Now()) {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "JWT expired")
			return
		}
		// JWT 토큰 유효성 검증 (HSET에서 조회)
		userID, err := p.rdb.HGetJWTAccess(tokens[1])
		if err != nil {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid JWT")
			return
		}
		/*
			// [신규] 토큰 만료 임박 체크 (30분 이내)
			expiringSoon, err := utils.IsTokenExpiringSoon(
				tokens[1],
				p.cfg.Server.JWTSecret,
				120*time.Minute,
			)
			if err != nil {
				p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check JWT token expiration")
				return
			}

			if expiringSoon {
				// [신규] 새 Access Token 발급
				claims := utils.GetJWTClaims(strconv.FormatUint(userID.Uid, 10))
				newToken, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, claims, 24*time.Hour)
				if err != nil {
					p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create new JWT access token")
					return
				}

				err = p.rdb.DeleteJWTToken(tokens[1])
				if err != nil {
					logger.Error("Failed to delete old JWT access token", "error", err.Error())
					p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to delete old JWT access token")
					return
				}

				// [신규] Redis 업데이트
				err = p.rdb.HSetJWTAccess(newToken, userID)
				if err != nil {
					logger.Error("Failed to set new JWT access token", "error", err.Error())
					p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set new JWT access token")
					return
				}

				err = p.rdb.HRefreshJoinWTRoom(userID.Uid)
				if err != nil {
					logger.Error("Failed to refresh join WebRTC room", "error", err.Error())
					p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to refresh join WebRTC room")
					return
				}

				c.Header("X-New-Access-Token", newToken)
				c.Header("Authorization", "Bearer "+newToken)
			}
		*/
		c.Set("user", userID)
		c.Next()
	}
}

// Check for login for favorites
func (p *Router) OptionalJwt() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokens := strings.Split(authHeader, " ")
		if len(tokens) != 2 {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid Bearer Type")
			return
		}

		// JWT 토큰 유효성 검증 (HSET에서 조회)
		userID, err := p.rdb.HGetJWTAccess(tokens[1])
		if err != nil {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid JWT")
			return
		}

		c.Set("user", userID)
		c.Next()
	}
}

// detectContentTypeFromFile reads the first 512 bytes and detects content type.
// Uses both http.DetectContentType and magic bytes for reliable detection.
// Used when client sends application/octet-stream so we can allow real image/video types.
func detectContentTypeFromFile(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if n == 0 {
		return "", fmt.Errorf("file is empty")
	}
	buf = buf[:n]

	firstBytesLen := 16
	if n < 16 {
		firstBytesLen = n
	}
	firstBytesHex := hex.EncodeToString(buf[:firstBytesLen])

	// 먼저 매직 바이트로 직접 확인 (더 확실함)
	detected := detectByMagicBytes(buf, n)
	if detected != "" {
		logger.Info("detectContentTypeFromFile: Detected by magic bytes",
			"filename", fh.Filename,
			"content_type", detected,
			"first_bytes", firstBytesHex)
		return detected, nil
	}

	// 매직 바이트로 확인 안되면 http.DetectContentType 사용
	detected = http.DetectContentType(buf)
	logger.Info("detectContentTypeFromFile: Using http.DetectContentType",
		"filename", fh.Filename,
		"detected_raw", detected,
		"bytes_read", n,
		"first_bytes", firstBytesHex)

	if idx := strings.Index(detected, ";"); idx >= 0 {
		detected = strings.TrimSpace(detected[:idx])
	}

	logger.Info("detectContentTypeFromFile: Final content type", "filename", fh.Filename, "content_type", detected)
	return detected, nil
}

// detectByMagicBytes checks file magic bytes to determine content type
func detectByMagicBytes(buf []byte, n int) string {
	if n < 4 {
		return ""
	}

	// JPEG: FF D8 FF
	if n >= 3 && buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return "image/jpeg"
	}

	// PNG: 89 50 4E 47
	if n >= 4 && buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return "image/png"
	}

	// GIF87a: 47 49 46 38 37 61
	// GIF89a: 47 49 46 38 39 61
	if n >= 6 && buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46 && buf[3] == 0x38 {
		if (buf[4] == 0x37 || buf[4] == 0x39) && buf[5] == 0x61 {
			return "image/gif"
		}
	}

	// WEBP: RIFF...WEBP (RIFF는 4바이트, 다음 4바이트는 파일 크기, 그 다음 4바이트는 "WEBP")
	if n >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
		return "image/webp"
	}

	// MP4: ftyp... (보통 4바이트 offset 후에 ftyp가 나타남)
	if n >= 8 {
		if string(buf[4:8]) == "ftyp" {
			return "video/mp4"
		}
		if n >= 4 && string(buf[0:4]) == "ftyp" {
			return "video/mp4"
		}
	}

	return ""
}

// validateFileUploadImpl is the shared implementation for file upload validation.
// When encParam is true, it extracts sinfo from form.Value["data"][0] (encrypted parameter mode).
// When encParam is false, it sets sinfo to the full form.Value map.
func (p *Router) validateFileUploadImpl(maxFiles int, size int64, encParam bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			logger.Warn("ValidateFileUpload: Invalid content type", "content-type", contentType)
			p.ctl.RespError(c, "Invalid content type. Expected multipart/form-data", http.StatusBadRequest)
			return
		}

		form, err := c.MultipartForm()
		if err != nil {
			logger.Warn("ValidateFileUpload: Failed to parse multipart form", "error", err.Error())
			if strings.Contains(err.Error(), "request body too large") {
				p.ctl.RespError(c, "Request body too large. Maximum size exceeded", http.StatusRequestEntityTooLarge)
			} else {
				p.ctl.RespError(c, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			}
			return
		}

		if form == nil || form.File == nil {
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Get files from form
		files := form.File["files"]
		formKeys := make([]string, 0, len(form.File))
		for k := range form.File {
			formKeys = append(formKeys, k)
		}
		logger.Info("ValidateFileUpload: Files found", "count", len(files), "form_keys", formKeys)

		if len(files) == 0 {
			logger.Warn("ValidateFileUpload: No files in 'files' field")
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		if len(files) > maxFiles {
			p.ctl.RespError(c, "Too many files. Maximum allowed is "+strconv.Itoa(maxFiles), http.StatusBadRequest)
			return
		}

		maxSize := size << 20
		allowedTypes := map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
			"video/mp4":  true,
		}

		validFiles := make([]*multipart.FileHeader, 0, len(files))

		for _, fileHeader := range files {
			if fileHeader.Size > maxSize {
				p.ctl.RespError(c, fmt.Sprintf("File %s exceeds %dMB limit", fileHeader.Filename, size), http.StatusBadRequest)
				return
			}

			ct := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
			if idx := strings.Index(ct, ";"); idx >= 0 {
				ct = strings.TrimSpace(ct[:idx])
			}

			if ct == "" || ct == "application/octet-stream" {
				detected, err := detectContentTypeFromFile(fileHeader)
				if err != nil {
					logger.Warn("ValidateFileUpload: Failed to detect content type", "file", fileHeader.Filename, "error", err.Error())
					p.ctl.RespError(c, fmt.Sprintf("File %s: could not determine type", fileHeader.Filename), http.StatusBadRequest)
					return
				}
				ct = detected
			}
			if !allowedTypes[ct] {
				p.ctl.RespError(c, fmt.Sprintf("File %s has invalid type. Only JPG, PNG, GIF, WEBP (images) and MP4 (video) files are allowed", fileHeader.Filename), http.StatusBadRequest)
				return
			}

			validFiles = append(validFiles, fileHeader)
		}

		// Store validated files in context
		c.Set("uploadedFiles", validFiles)
		if encParam {
			c.Set("sinfo", form.Value["data"][0])
		} else {
			c.Set("sinfo", form.Value)
		}

		c.Next()
	}
}

// ValidateFileUpload checks if the uploaded files meet the requirements
func (p *Router) ValidateFileUpload(maxFiles int, size int64) gin.HandlerFunc {
	return p.validateFileUploadImpl(maxFiles, size, false)
}

// EncParamFileUpload validates file upload and extracts encrypted parameter from form data
func (p *Router) EncParamFileUpload(maxFiles int, size int64) gin.HandlerFunc {
	return p.validateFileUploadImpl(maxFiles, size, true)
}

func (p *Router) AesEncrypt() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 원본 ResponseWriter를 보관
		originalWriter := c.Writer

		// ResponseWriter를 래핑하여 응답을 가로챔
		writer := &ptl.BodyWriter{
			ResponseWriter: c.Writer,
			Body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 다음 핸들러 실행
		c.Next()

		// 응답 본문 가져오기
		body := writer.Body.Bytes()

		// 빈 응답이면 그대로 반환
		if len(body) == 0 {
			return
		}

		// AES 키 준비
		aesKey, err := hex.DecodeString(p.cfg.Server.BaseKey)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to encode AES key")
			return
		}

		// 응답 데이터 암호화
		encryptedData, err := utils.EncryptGCM(body, aesKey)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to encrypt response")
			return
		}

		// 암호화된 응답 생성
		encResp := ptl.AesDataForm{
			Data: encryptedData,
		}

		// 원본 writer로 암호화된 응답 전송
		originalWriter.Header().Set("Content-Type", "application/json")
		originalWriter.WriteHeader(writer.Status())
		json.NewEncoder(originalWriter).Encode(encResp)
	}
}

func (p *Router) AesDecrypt() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 먼저 암호화된 wrapper 구조체를 파싱
		var encReq ptl.AesDataForm
		if err := c.ShouldBindJSON(&encReq); err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request format")
			return
		}

		aesKey := []byte(p.cfg.Server.BaseKey)

		// 3. data 필드 복호화
		decryptedBytes, err := utils.DecryptGCM(encReq.Data, aesKey)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to decrypt data")
			return
		}

		// 4. 복호화된 JSON을 새로운 request body로 설정
		c.Request.Body = io.NopCloser(bytes.NewBuffer(decryptedBytes))

		c.Next()
	}
}

func (p *Router) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-XSS-Protection", "1; mode=block")
		// 클릭재킹 방지
		c.Header("X-Frame-Options", "DENY")
		// MIME 타입 스니핑 방지
		c.Header("X-Content-Type-Options", "nosniff")
		// HSTS 설정
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
