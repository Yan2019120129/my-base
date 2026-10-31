package apis

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Home 提供前台网站页面入口。
type Home struct{}

// Index 渲染网站首页。
func (Home) Index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "home.tmpl", gin.H{
		"Year": time.Now().Year(),
	})
}
