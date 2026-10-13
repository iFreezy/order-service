package rprocessor

import (
	"github.com/gin-gonic/gin"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
)

func v1RegOrderHandler(v1 *gin.RouterGroup, h rhandler.Order) {
	v1.POST("/order/create", h.Create)
	v1.GET("/order/:guid", h.GetByGUID)
	v1.PATCH("/order/:guid", h.Update)
	v1.DELETE("/order/:guid", h.Delete)
	v1.POST("/order/list", h.List)
}
