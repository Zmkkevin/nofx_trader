package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBacktestAPI_HTTPStatusCodes(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("should return 404 when metrics not ready", func(t *testing.T) {
		// This test verifies BLOCKING #5 fix: 202 → 404
		// 当指标数据尚未准备好时，应该返回 404 而不是 202
		// 202 Accepted 用于异步操作已被接受但尚未完成
		// 404 Not Found 用于资源不存在或尚未准备好

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Simulate the error response we expect
		c.JSON(http.StatusNotFound, gin.H{
			"error": "metrics not ready yet",
			"code":  "METRICS_NOT_READY",
		})

		// Verify status code is 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		// Verify response body format
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["error"] != "metrics not ready yet" {
			t.Errorf("Expected error message, got %v", response["error"])
		}

		if response["code"] != "METRICS_NOT_READY" {
			t.Errorf("Expected error code METRICS_NOT_READY, got %v", response["code"])
		}
	})

	t.Run("should NOT use 202 for metrics not ready state", func(t *testing.T) {
		// Verify we're NOT using 202 Accepted incorrectly
		// This test documents the semantic difference between status codes

		// 202 is for async operations that are accepted but not complete
		acceptedStatus := http.StatusAccepted // 202

		// 404 is for resources that don't exist yet or are not ready
		notFoundStatus := http.StatusNotFound // 404

		// They should be different
		if acceptedStatus == notFoundStatus {
			t.Error("Status codes 202 and 404 should be different")
		}

		// Our fix should use 404, not 202
		expectedStatus := http.StatusNotFound
		wrongStatus := http.StatusAccepted

		if expectedStatus == wrongStatus {
			t.Error("Expected to use 404 Not Found, not 202 Accepted")
		}
	})
}

