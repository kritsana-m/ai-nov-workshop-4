package main

import (
	"fmt"
	"log"

	mod "kbtg-ai-workshop-nov/workshop-4/backend/internal/config"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/handlers"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	port, dbPath := mod.LoadConfig()

	// init DB
	db, err := store.InitDB(dbPath)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	store.SetDB(db)

	r := gin.Default()

	// health
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "hello world"})
	})

	// register handlers
	handlers.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
