package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ptl "ms-gateway/protocol"
)

func TestStoryPriceValidation(t *testing.T) {
	for _, tc := range []struct {
		body   string
		uid    uint64
		status int
	}{
		{`{"idx":"5","stat":"2","price":100}`, 0, http.StatusUnauthorized},
		{`{"idx":"5","stat":"2"}`, 10, http.StatusBadRequest},
		{`{"idx":"5","stat":"2","price":null}`, 10, http.StatusBadRequest},
		{`{"idx":"0","stat":"2","price":100}`, 10, http.StatusBadRequest},
		{`{"idx":"5","stat":"6","price":100}`, 10, http.StatusBadRequest},
		{`{"idx":"5","stat":"2","price":-1}`, 10, http.StatusBadRequest},
		{`{"idx":"5","stat":"2","price":1.5}`, 10, http.StatusBadRequest},
		{`{"idx":"5","stat":"2","price":2147483648}`, 10, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/story/v01/updprice", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", &ptl.UserInfoResp{Uid: tc.uid})
		ctl := &StoryController{ctl: &Controller{}}
		ctl.UpdStoryPriceInfo(c)
		if w.Code != tc.status {
			t.Fatalf("%s: HTTP %d: %s", tc.body, w.Code, w.Body.String())
		}
	}
}
