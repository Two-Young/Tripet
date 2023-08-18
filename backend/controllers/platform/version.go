package platform

import "github.com/gin-gonic/gin"

const VERSION = "23.0818.1509"

func Version(c *gin.Context) {
	c.JSON(200, VERSION)
}

func UseVersionRouter(g *gin.RouterGroup) {
	rg := g.Group("/version")
	rg.GET("", Version)
}
