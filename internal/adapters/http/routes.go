package http

import (
	"safelyyou/internal/core/ports"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(r *gin.Engine, deviceSvc ports.DeviceService) {

	h := NewHandler(deviceSvc)

	api := r.Group("/api/v1")
	{
		devicesGroup := api.Group("/devices")
		{
			devicesGroup.POST("/:device_id/heartbeat", h.PostHeartbeat)
			devicesGroup.POST("/:device_id/stats", h.PostStats)
			devicesGroup.GET("/:device_id/stats", h.GetStats)
		}
	}
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
