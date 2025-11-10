package main

import (
	"fmt"
	"log"

	"kbtg-ai-workshop-nov/workshop-4/backend/internal/config"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/handlers"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// main is the entrypoint of the backend server. It loads configuration, initializes
// the database, registers HTTP routes and starts the Gin HTTP server.
func main() {
	port, dbPath := config.LoadConfig()

	// Initialize DB and make it available to handlers via store package.
	db, err := store.InitDB(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	store.SetDB(db)

	r := gin.Default()

	// Health check endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "hello world"})
	})

	// Register application routes
	handlers.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
