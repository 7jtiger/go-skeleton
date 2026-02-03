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

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"ms-gateway/common/logger"
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"
)

var limiters = make(map[string]*rate.Limiter)

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if _, exists := limiters[ip]; !exists {
			limiters[ip] = rate.NewLimiter(1, 5) // 1 request per second with a burst capacity of 5
		}
		limiter := limiters[ip]

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// TODO: 추후 확인 필요
/*
func CORS(cfg *conf.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set-Cookie 사용 시 Origin을 특정을 해줘야 CORS 에러가 발생하지 않음. FE 테스트 시 Test 도메인과 Local 도메인을 둘 다 사용하기 때문에 두 도메인을 모두 허용
		if cfg.Server.Mode == "test" || cfg.Server.Mode == "local" {
			origin := c.GetHeader("Origin")
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

*/

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
		metaHeader := c.GetHeader("x-meta")
		if metaHeader == "" {
			p.ctl.RespError(c, "Missing x-meta header", http.StatusBadRequest)
			return
		}

		parts := strings.Split(metaHeader, "/")
		if len(parts) != 8 {
			p.ctl.RespError(c, "Invalid x-meta header format", http.StatusBadRequest)
			return
		}

		// meta := ptc.MetaHeader{
		// 	UID:             parts[0],
		// 	SID:		     parts[1],
		// 	DID:		     parts[2],
		// 	NICK: 		     parts[3],
		// 	GENDER: 	     parts[4],
		// 	AGE: 		     parts[5],
		// 	AREA: 		     parts[6],
		// 	EMAIL: 		     parts[7],
		// 	MAIN_PIC: 		 parts[8],
		// 	THMB_PIC: 		 parts[9],
		// 	SP_INTRO:        parts[10],
		// }

		// MetaHeader를 context에 저장하여 핸들러에서 사용할 수 있도록 함
		// c.Set("metaHeader", meta)
		c.Header("x-meta", metaHeader)

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

		// JWT 토큰 유효성 검증 (HSET에서 조회)
		userID, err := p.rdb.HGetJWTAccess(tokens[1])
		logger.Info("userID", userID)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid JWT")
			return
		}

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
	// MP4는 여러 변형이 있어서 더 복잡하지만, 일반적으로 ftyp로 시작
	if n >= 8 {
		// ftyp는 보통 4바이트 offset 후에 나타남
		if string(buf[4:8]) == "ftyp" {
			return "video/mp4"
		}
		// 또는 처음부터 ftyp로 시작할 수도 있음
		if n >= 4 && string(buf[0:4]) == "ftyp" {
			return "video/mp4"
		}
	}

	return ""
}

// ValidateFileUpload checks if the uploaded files meet the requirements
func (p *Router) ValidateFileUpload(maxFiles int, size int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content-Type 확인
		/*
			meta := c.GetHeader("x-meta")
			if meta == "" {
				p.ctl.RespError(c, "Missing x-meta header", http.StatusBadRequest)
				return
			}
			// fmt.Println(meta) */

		contentType := c.GetHeader("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			logger.Warn("ValidateFileUpload: Invalid content type", "content-type", contentType)
			p.ctl.RespError(c, "Invalid content type. Expected multipart/form-data", http.StatusBadRequest)
			return
		}

		// 파일이 존재하는지 먼저 확인
		// MultipartForm()은 MaxMultipartMemory 설정에 따라 메모리나 디스크에 저장됨
		form, err := c.MultipartForm()
		if err != nil {
			logger.Warn("ValidateFileUpload: Failed to parse multipart form", "error", err.Error())
			// 더 자세한 에러 정보 제공
			if strings.Contains(err.Error(), "request body too large") {
				p.ctl.RespError(c, "Request body too large. Maximum size exceeded", http.StatusRequestEntityTooLarge)
			} else {
				p.ctl.RespError(c, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			}
			return
		}

		// form이나 files가 nil인지 확인
		if form == nil || form.File == nil {
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Get files from form
		files := form.File["files"]
		// 디버깅을 위한 form 키 목록
		formKeys := make([]string, 0, len(form.File))
		for k := range form.File {
			formKeys = append(formKeys, k)
		}
		logger.Info("ValidateFileUpload: Files found", "count", len(files), "form_keys", formKeys)

		// Check if any files were uploaded
		if len(files) == 0 {
			logger.Warn("ValidateFileUpload: No files in 'files' field")
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Check number of files
		if len(files) > maxFiles {
			p.ctl.RespError(c, "Too many files. Maximum allowed is "+strconv.Itoa(maxFiles), http.StatusBadRequest)
			return
		}

		// Check each file
		//5MB = 5 << 20   // 5 MB
		//100MB = 100 << 20  // 100 MB
		//20MB = 20 << 20  // 20 MB
		//2MB = 2 << 20  // 2 MB
		//50MB = 50 << 20  // 50 MB
		maxSize := size << 20 // 10MB per file
		// 추후 확장된다면 protocol에 추가하고 라우터에 맞는
		// 컨텐츠 타입들만 선언된 타입을 인자로 받도록 수정 가능
		allowedTypes := map[string]bool{
			// "application/pdf": true,
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
			"video/mp4":  true,
		}

		validFiles := make([]*multipart.FileHeader, 0, len(files))

		for _, fileHeader := range files {
			// Check file size
			if fileHeader.Size > maxSize {
				// maxSizeMB := size
				p.ctl.RespError(c, fmt.Sprintf("File %s exceeds %dMB limit", fileHeader.Filename, size), http.StatusBadRequest)
				return
			}

			// Check file type (헤더 + application/octet-stream일 때 실제 내용으로 재판별)
			contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
			if idx := strings.Index(contentType, ";"); idx >= 0 {
				contentType = strings.TrimSpace(contentType[:idx])
			}

			// originalContentType := contentType
			if contentType == "" || contentType == "application/octet-stream" {
				detected, err := detectContentTypeFromFile(fileHeader)
				if err != nil {
					logger.Warn("ValidateFileUpload: Failed to detect content type", "file", fileHeader.Filename, "error", err.Error())
					p.ctl.RespError(c, fmt.Sprintf("File %s: could not determine type", fileHeader.Filename), http.StatusBadRequest)
					return
				}
				contentType = detected
			}
			if !allowedTypes[contentType] {
				p.ctl.RespError(c, fmt.Sprintf("File %s has invalid type. Only JPG, PNG, GIF, WEBP (images) and MP4 (video) files are allowed", fileHeader.Filename), http.StatusBadRequest)
				return
			}

			validFiles = append(validFiles, fileHeader)
		}

		// Store validated files in context
		c.Set("uploadedFiles", validFiles)
		c.Set("sinfo", form.Value)

		c.Next()
	}
}

func (p *Router) EncParamFileUpload(maxFiles int, size int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
			meta := c.GetHeader("x-meta")
			if meta == "" {
				p.ctl.RespError(c, "Missing x-meta header", http.StatusBadRequest)
				return
			}
			// fmt.Println(meta) */

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

		// Check if any files were uploaded
		if len(files) == 0 {
			logger.Warn("ValidateFileUpload: No files in 'files' field")
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Check number of files
		if len(files) > maxFiles {
			p.ctl.RespError(c, "Too many files. Maximum allowed is "+strconv.Itoa(maxFiles), http.StatusBadRequest)
			return
		}

		// Check each file
		//5MB = 5 << 20   // 5 MB
		//100MB = 100 << 20  // 100 MB
		//20MB = 20 << 20  // 20 MB
		//2MB = 2 << 20  // 2 MB
		//50MB = 50 << 20  // 50 MB
		maxSize := size << 20 // 10MB per file
		// 추후 확장된다면 protocol에 추가하고 라우터에 맞는
		// 컨텐츠 타입들만 선언된 타입을 인자로 받도록 수정 가능
		allowedTypes := map[string]bool{
			// "application/pdf": true,
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

			// Check file type (헤더 + application/octet-stream일 때 실제 내용으로 재판별)
			contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
			if idx := strings.Index(contentType, ";"); idx >= 0 {
				contentType = strings.TrimSpace(contentType[:idx])
			}

			// originalContentType := contentType
			if contentType == "" || contentType == "application/octet-stream" {
				detected, err := detectContentTypeFromFile(fileHeader)
				if err != nil {
					logger.Warn("ValidateFileUpload: Failed to detect content type", "file", fileHeader.Filename, "error", err.Error())
					p.ctl.RespError(c, fmt.Sprintf("File %s: could not determine type", fileHeader.Filename), http.StatusBadRequest)
					return
				}
				contentType = detected
			}
			if !allowedTypes[contentType] {
				p.ctl.RespError(c, fmt.Sprintf("File %s has invalid type. Only JPG, PNG, GIF, WEBP (images) and MP4 (video) files are allowed", fileHeader.Filename), http.StatusBadRequest)
				return
			}

			validFiles = append(validFiles, fileHeader)
		}

		// Store validated files in context
		c.Set("uploadedFiles", validFiles)
		c.Set("sinfo", form.Value["data"][0])

		c.Next()
	}
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

		// 2. AES 키 준비
		// var aesKey = []byte(p.cfg.Server.BaseKey)
		// aesKey, err := hex.DecodeString(p.cfg.Server.BaseKey)
		// if err != nil {
		// 	p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to decode AES key")
		// 	return
		// }

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
