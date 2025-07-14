package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	fireCastSecret := os.Getenv("FIRECAST_SECRET")
	azuraCastApiKey := os.Getenv("AZURACAST_API_KEY")
	azuraCastDomain := os.Getenv("AZURACAST_DOMAIN")

	if fireCastSecret == "" || azuraCastApiKey == "" || azuraCastDomain == "" {
		fmt.Println("Environment variables FIRECAST_SECRET, AZURACAST_API_KEY, and AZURACAST_DOMAIN must be set")
		return
	}



}
