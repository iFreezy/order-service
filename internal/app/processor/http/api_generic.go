package rprocessor

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
)

func vGenericRegHealthCheck(r *gin.Engine, h rhandler.Health) {
	r.GET("/health", h.LastCheck)
}

func handleNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
}
