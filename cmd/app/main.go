package main

import (
	"log"
	"os"
	"safelyyou/internal/adapters/repository/memory"

	"github.com/joho/godotenv"
)

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

}
