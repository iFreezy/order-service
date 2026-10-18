package rhandler

import "github.com/gin-gonic/gin"

type Health interface {
	LastCheck(c *gin.Context)
}

type Order interface {
	Create(c *gin.Context)
	GetByGUID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	List(c *gin.Context)
}
