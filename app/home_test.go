package app

import (
	"net/http"
	"strings"
	"testing"

	appRouter "my-base/app/router"

	"github.com/gin-gonic/gin"
)

// TestHomePageIsPubliclyAccessible 校验首页无需登录即可访问并完成模板渲染。
func TestHomePageIsPubliclyAccessible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.LoadHTMLGlob("../html/*")
	appRouter.InitRouter(router)

	recorder := performRequest(router, http.MethodGet, "/", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected home page status %d, got %d with body %q", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected html content type, got %q", contentType)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"020200",
		"把复杂留给系统",
		"/assets/home/home.css",
		"/assets/home/home.js",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected home page to contain %q", expected)
		}
	}
}
