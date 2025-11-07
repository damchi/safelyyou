package main

import (
	"log"
	"os"
	_ "safelyyou/docs"
	"safelyyou/internal/adapters/http"
	"safelyyou/internal/adapters/repository/memory"
	"safelyyou/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Package main Fleet Management Simple Metrics Server.
//
// @title Fleet Management Simple Metrics Server
// @version 1.0
// @description Simple and correct implementation of the Fleet Management Metrics Coding Assessment
//
// @BasePath /api/v1
func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}
	csvPath := os.Getenv("DEVICE_CSV")

	deviceRepo := memory.NewDeviceRepository()

	if err := deviceRepo.LoadFromCSV(csvPath); err != nil {
		log.Fatalf("failed to load devices from %s: %v", csvPath, err)
	}

	log.Printf("devices loaded from %s: %d", csvPath, deviceRepo.Count())

	deviceSvc := services.NewDeviceService(deviceRepo)

	r := gin.Default()
	http.RegisterRoutes(r, deviceSvc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
