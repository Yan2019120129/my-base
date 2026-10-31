package app

import (
	"context"
	"errors"
	"io"
	"my-base/app/router"
	"my-base/app/tables"
	"my-base/app/tasks"
	"my-base/code"
	"my-base/configs"
	fileService "my-base/module/file"
	"my-base/pages"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/logger"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	"github.com/gin-gonic/gin"
)

func StartServer() {
	cfg := configs.GetConfig()
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	r := gin.Default()
	r.LoadHTMLGlob("html/*")

	template.AddComp(chartjs.NewChart())

	eng := engine.Default()
	var tusServer *fileService.TusServer

	if configs.GetTusUpload().Enabled {
		var err error
		tusServer, err = fileService.NewTusServer(configs.GetTusUpload())
		if err != nil {
			panic(err)
		}
	}

	if err := eng.AddConfig(configs.GetAdmin()).
		AddGenerators(tables.Generators).
		Use(r); err != nil {
		panic(err)
	}
	conn := eng.DefaultConnection()

	r.Use(func(c *gin.Context) {
		c.Set(code.ContextDBKey, conn)
	})

	r.Static(cfg.Admin.AssetRootPath, cfg.Admin.Store.Path)

	router.InitRouter(r, router.AdminAuthMiddleware)
	if tusServer != nil {
		router.RegisterTusUploadRouter(r, configs.GetTusUpload().Endpoint, tusServer)
		r.StaticFile("/assets/vendor/tus.min.js", "./website/public/vendor/tus-js-client/tus.min.js")
	}

	eng.HTML("GET", cfg.Admin.Prefix(), pages.GetDashBoard)

	taskCtx, stopTasks := context.WithCancel(context.Background())
	taskRunner, err := tasks.Start(taskCtx, conn)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Info("|********** http://localhost" + srv.Addr + cfg.Admin.Prefix() + "**********|")
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			logger.Info("listen:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	stopTasks()
	taskRunner.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}
	logger.Info("closing database connection")
	eng.DefaultConnection().Close()

	logger.Info("Server exiting")
}
