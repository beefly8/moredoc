package serve

import (
	"moredoc/biz"
	"moredoc/conf"
	"moredoc/middleware/auth"
	"moredoc/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterGinRouter 注册gin路由
func RegisterGinRouter(app *gin.Engine, dbModel *model.DBModel, logger *zap.Logger, auth *auth.Auth, cfg *conf.Config) (err error) {
	attachmentAPIService := biz.NewAttachmentAPIService(dbModel, logger, &cfg.S3Store)

	//开启自动清除本地文件
	go attachmentAPIService.AutoCleanAttachment()

	app.GET("/favicon.ico", attachmentAPIService.Favicon)
	app.GET("/static/images/logo.png", attachmentAPIService.Logo)
	app.GET("/sitemap.xml", func(ctx *gin.Context) {
		ctx.File("./sitemap/sitemap.xml")
	})

	app.GET("/view/page/:hash/:page", attachmentAPIService.ViewDocumentPages)
	app.GET("/view/cover/:hash", attachmentAPIService.ViewDocumentCover)
	app.GET("/download/:jwt", attachmentAPIService.DownloadDocument)

	checkPermissionGroup := app.Group("/api/v1/upload")
	checkPermissionGroup.Use(auth.AuthGin())
	{
		checkPermissionGroup.POST("avatar", attachmentAPIService.UploadAvatar)
		checkPermissionGroup.POST("config", attachmentAPIService.UploadConfig)
		checkPermissionGroup.POST("banner", attachmentAPIService.UploadBanner)
		checkPermissionGroup.POST("document", attachmentAPIService.UploadDocument)
		checkPermissionGroup.POST("category", attachmentAPIService.UploadCategory)
		checkPermissionGroup.POST("article", attachmentAPIService.UploadArticle)
	}

	return
}
