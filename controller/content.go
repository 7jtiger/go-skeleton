package controller

import (
	"github.com/gin-gonic/gin"

	"ms-gateway/conf"
	"ms-gateway/models"
)

type ContentController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories
}

func NewContentController(ctl *Controller, rep *models.Repositories) (*ContentController, error) {
	r := &ContentController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	return r, nil
}

// swagger:route GET /content/v01/test content test
// @Summary Content test endpoint
// @Description Test endpoint for content controller
// @Tags content
// @Accept json
// @Produce json
// @Success 200 {object} protocol.OkResp
// @Router /content/v01/test [get]
func (p *ContentController) GetTestContent(c *gin.Context) {
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Content controller is working"})
}

type ChatRoomListController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories
	rdb *models.RedisDB
}

func NewChatRoomListController(ctl *Controller, rep *models.Repositories) (*ChatRoomListController, error) {
	r := &ChatRoomListController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

// swagger:route GET /chatroom/v01/list get chatroom list
// @Summary Get user's chatroom list
// @Description Get user's chatroom list sorted by last activity and priority
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(20)
// @Success 200 {object} protocol.RespDataHeader{data=[]models.ChatRoomListItem}
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/list [get]
func (p *ChatRoomListController) GetChatRoomList(c *gin.Context) {
	/*
		 	// JWT에서 사용자 ID 추출
			userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			// 페이지네이션 파라미터
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

			if limit > 100 {
				limit = 100 // 최대 100개로 제한
			}

			// 채팅룸 리스트 조회
			rooms, err := p.rdb.GetUserChatRoomList(userID.(string), offset, limit)
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to get chatroom list"), http.StatusInternalServerError, err)
				return
			}

			p.ctl.SendDataResponse(c, http.StatusOK, rooms)
	*/
}

// swagger:route POST /chatroom/v01/join join chatroom
// @Summary Join a chatroom
// @Description Add a chatroom to user's list (when login or first access)
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param data body object true "{roomId: string, roomName: string}"
// @Success 200 {object} protocol.OkResp
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/join [post]
func (p *ChatRoomListController) JoinChatRoom(c *gin.Context) {
	/*
		 	userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			var req struct {
				RoomID   string `json:"roomId" binding:"required"`
				RoomName string `json:"roomName" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "Invalid request"), http.StatusBadRequest, err)
				return
			}

			// 채팅룸 리스트에 추가
			err := p.rdb.AddUserToChatRoomList(userID.(string), req.RoomID, req.RoomName)
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to join chatroom"), http.StatusInternalServerError, err)
				return
			}

			log.Info(fmt.Sprintf("User %s joined chatroom %s", userID.(string), req.RoomID))
			p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully joined chatroom"})
	*/
}

// swagger:route POST /chatroom/v01/event handle chatroom event
// @Summary Handle chatroom event
// @Description Move chatroom to top based on event type (message, mention, urgent, read)
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param data body object true "{roomId: string, eventType: string, lastMessage: string}"
// @Success 200 {object} protocol.OkResp
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/event [post]
func (p *ChatRoomListController) HandleChatRoomEvent(c *gin.Context) {
	/*
		 	userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			var req struct {
				RoomID      string `json:"roomId" binding:"required"`
				EventType   string `json:"eventType" binding:"required"`
				LastMessage string `json:"lastMessage"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "Invalid request"), http.StatusBadRequest, err)
				return
			}

			// 유효한 이벤트 타입 확인
			validEvents := map[string]bool{
				"message": true,
				"mention": true,
				"urgent":  true,
				"read":    true,
			}

			if !validEvents[req.EventType] {
				p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid event type")
				return
			}

			// 채팅룸을 최상단으로 이동
			err := p.rdb.MoveRoomToTop(userID.(string), req.RoomID, req.EventType, req.LastMessage)
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to handle event"), http.StatusInternalServerError, err)
				return
			}

			log.Info(fmt.Sprintf("User %s - Room %s event: %s", userID.(string), req.RoomID, req.EventType))
			p.ctl.SimpleRespOK(c, gin.H{"msg": "Event handled successfully"})
	*/
}

// swagger:route DELETE /chatroom/v01/leave/:roomId leave chatroom
// @Summary Leave a chatroom
// @Description Remove chatroom from user's list
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param roomId path string true "Room ID"
// @Success 200 {object} protocol.OkResp
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/leave/{roomId} [delete]
func (p *ChatRoomListController) LeaveChatRoom(c *gin.Context) {
	/*
		 	userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			roomID := c.Param("roomId")
			if roomID == "" {
				p.ctl.SimpleError(c, http.StatusBadRequest, "Room ID is required")
				return
			}

			// 채팅룸 리스트에서 제거
			err := p.rdb.RemoveUserFromChatRoomList(userID.(string), roomID)
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to leave chatroom"), http.StatusInternalServerError, err)
				return
			}

			log.Info(fmt.Sprintf("User %s left chatroom %s", userID.(string), roomID))
			p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully left chatroom"})
	*/
}

// swagger:route GET /chatroom/v01/stats get chatroom stats
// @Summary Get chatroom statistics
// @Description Get total chatroom count and unread message count
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} protocol.RespDataHeader{data=object}
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/stats [get]
func (p *ChatRoomListController) GetChatRoomStats(c *gin.Context) {
	/*
		 	userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			// 총 채팅룸 수 조회
			totalRooms, err := p.rdb.GetChatRoomListCount(userID.(string))
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to get chatroom count"), http.StatusInternalServerError, err)
				return
			}

			// 총 읽지 않은 메시지 수 조회
			totalUnread, err := p.rdb.GetUnreadTotalCount(userID.(string))
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to get unread count"), http.StatusInternalServerError, err)
				return
			}

			stats := gin.H{
				"totalRooms":  totalRooms,
				"totalUnread": totalUnread,
				"hasUnread":   totalUnread > 0,
			}

			p.ctl.SendDataResponse(c, http.StatusOK, stats)
	*/
}

// swagger:route POST /chatroom/v01/read mark as read
// @Summary Mark chatroom as read
// @Description Update unread count to 0 for specific chatroom
// @Tags chatroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param data body object true "{roomId: string}"
// @Success 200 {object} protocol.OkResp
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Router /chatroom/v01/read [post]
func (p *ChatRoomListController) MarkAsRead(c *gin.Context) {
	/*
		 	userID, exists := c.Get("user")
			if !exists {
				p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
				return
			}

			var req struct {
				RoomID string `json:"roomId" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "Invalid request"), http.StatusBadRequest, err)
				return
			}

			// 읽음 처리 (읽지 않은 메시지 수를 0으로 설정)
			err := p.rdb.UpdateRoomUnreadCount(userID.(string), req.RoomID, 0)
			if err != nil {
				p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "Failed to mark as read"), http.StatusInternalServerError, err)
				return
			}

			// 읽음 이벤트 처리 (우선순위도 초기화)
			err = p.rdb.MoveRoomToTop(userID.(string), req.RoomID, "read", "")
			if err != nil {
				log.Warn("Failed to update room priority after read:", err)
			}

			p.ctl.SimpleRespOK(c, gin.H{"msg": "Marked as read"})
	*/
}
