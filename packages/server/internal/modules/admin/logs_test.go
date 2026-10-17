package admin

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/logx"
)

func TestLogsReturnsBoundedFormattedEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := logx.NewBuffer(4)
	logs.Append(logx.Entry{Level: "INFO", Message: "first", Line: "[INFO] first"})
	logs.Append(logx.Entry{Level: "WARN", Message: "second", Line: "[WARN] second"})
	h := &Handler{deps: app.Deps{Logs: logs}}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/admin/logs?limit=1&level=warn", nil)
	h.logs(c)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Entries []struct {
				Level string `json:"level"`
				Line  string `json:"line"`
			} `json:"entries"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(recorder.Body.Bytes())).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 200 || len(envelope.Data.Entries) != 1 || envelope.Data.Entries[0].Level != "warn" || envelope.Data.Entries[0].Line != "[WARN] second" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestLogsRejectsUnknownLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{deps: app.Deps{Logs: logx.NewBuffer(2)}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/admin/logs?level=trace", nil)
	h.logs(c)
	if recorder.Code != 400 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
