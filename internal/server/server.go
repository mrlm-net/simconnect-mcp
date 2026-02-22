package server

import "github.com/gin-gonic/gin"

// New returns a configured Gin router.
func New() *gin.Engine {
	return gin.Default()
}
