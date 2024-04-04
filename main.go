package main

import (
	"Currency/database"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	database.Database()

	app := fiber.New()

	app.Use(logger.New())
	app.Use(recover.New())

	websocketHandler(app)
	log.Fatal(app.Listen(":8080"))
}

func router(app *fiber.App) {
}

// Websocket handler
func websocketHandler(app *fiber.App) {
}
