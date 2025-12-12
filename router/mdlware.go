package router

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-Forwarded-For, Authorization, Accept, Origin, Cache-Control, X-Requested-With, OTP-Auth, X-Otp, X-Meta, X-Name, Siwe-Session")
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

// ValidateFileUpload checks if the uploaded files meet the requirements
func (p *Router) ValidateFileUpload(maxFiles int, size int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 파일이 존재하는지 먼저 확인
		form, err := c.MultipartForm()
		if err != nil {
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// form이나 files가 nil인지 확인
		if form == nil || form.File == nil {
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Get files from form
		files := form.File["files"]

		// Check if any files were uploaded
		if len(files) == 0 {
			p.ctl.RespError(c, "No Uploaded File", http.StatusBadRequest)
			return
		}

		// Check number of files
		if len(files) > maxFiles {
			p.ctl.RespError(c, "Too many files. Maximum allowed is "+strconv.Itoa(maxFiles), http.StatusBadRequest)
			return
		}

		// Check each file
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
		}

		validFiles := make([]*multipart.FileHeader, 0, len(files))

		for _, fileHeader := range files {
			// Check file size
			if fileHeader.Size > maxSize {
				p.ctl.RespError(c, "File "+fileHeader.Filename+" exceeds 10MB limit", http.StatusBadRequest)
				return
			}

			// Check file type
			contentType := fileHeader.Header.Get("Content-Type")
			if !allowedTypes[contentType] {
				p.ctl.RespError(c, "File "+fileHeader.Filename+" has invalid type. Only PDF, JPG, PNG, GIF and WEBP files are allowed", http.StatusBadRequest)
				return
			}

			validFiles = append(validFiles, fileHeader)
		}

		// Store validated files in context
		c.Set("uploadedFiles", validFiles)

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
