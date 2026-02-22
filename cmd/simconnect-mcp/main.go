package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	mode := os.Getenv("MCP_MODE")
	if mode == "" {
		mode = "docs"
	}

	log.Printf("Starting SimConnect MCP in mode: %s", mode)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "mode": mode})
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
