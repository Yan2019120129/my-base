package router

import (
	"my-base/app/apis"

	"github.com/gin-gonic/gin"
)

func init() {
	routerNoCheckRole = append(routerNoCheckRole, homeRouter)
}

// homeRouter 注册前台网站首页路由。
func homeRouter(v1 *gin.RouterGroup) {
	api := apis.Home{}
	v1.GET("/", api.Index)
}
