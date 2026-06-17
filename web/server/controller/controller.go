package controller

import (
	"net/http"
	"time"

	"livein-web/conf"
	"livein-web/model"
	"livein-web/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Controller struct {
	cfg   *conf.Config
	store *store.FileStore
}

func New(cfg *conf.Config, s *store.FileStore) *Controller {
	return &Controller{cfg: cfg, store: s}
}

func (c *Controller) GetContent(ctx *gin.Context) {
	content, err := c.store.GetContent()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "콘텐츠를 불러올 수 없습니다",
		})
		return
	}

	ctx.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data:    content,
	})
}

func (c *Controller) UpdateContent(ctx *gin.Context) {
	var content model.Content
	if err := ctx.ShouldBindJSON(&content); err != nil {
		ctx.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "잘못된 요청입니다",
		})
		return
	}

	if err := c.store.UpdateContent(&content); err != nil {
		ctx.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "콘텐츠 저장에 실패했습니다",
		})
		return
	}

	ctx.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "저장되었습니다",
		Data:    content,
	})
}

func (c *Controller) SubmitContact(ctx *gin.Context) {
	var req model.ContactRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "이름, 이메일, 문의 내용을 확인해주세요",
		})
		return
	}

	inquiry := model.Inquiry{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		Message:   req.Message,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := c.store.AddInquiry(inquiry); err != nil {
		ctx.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "문의 접수에 실패했습니다",
		})
		return
	}

	ctx.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "문의가 접수되었습니다",
	})
}

func (c *Controller) ListInquiries(ctx *gin.Context) {
	inquiries, err := c.store.ListInquiries()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "문의 목록을 불러올 수 없습니다",
		})
		return
	}

	ctx.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data:    inquiries,
	})
}

func (c *Controller) Login(ctx *gin.Context) {
	var req model.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "아이디와 비밀번호를 입력해주세요",
		})
		return
	}

	if req.Username != c.cfg.Admin.Username {
		ctx.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    401,
			Message: "인증에 실패했습니다",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(c.cfg.Admin.Password), []byte(req.Password)); err != nil {
		// 평문 비밀번호 fallback (개발용 config)
		if req.Password != c.cfg.Admin.Password {
			ctx.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    401,
				Message: "인증에 실패했습니다",
			})
			return
		}
	}

	expire := time.Duration(c.cfg.Admin.JWTExpireMin) * time.Minute
	claims := jwt.MapClaims{
		"sub": req.Username,
		"exp": time.Now().Add(expire).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(c.cfg.Admin.JWTSecret))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    500,
			Message: "토큰 발급에 실패했습니다",
		})
		return
	}

	ctx.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: model.LoginResponse{Token: tokenStr},
	})
}

func (c *Controller) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
