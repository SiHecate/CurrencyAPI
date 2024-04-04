package main

import (
	"Currency/database"
	"Currency/pkg"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	database.Database()

	app := fiber.New()

	// API endpointlerini tanımla
	ApiRoutes(app)
	WebsocketHandler(app)

	// Middleware'ler
	app.Use(logger.New())
	app.Use(recover.New())

	log.Fatal(app.Listen(":8080"))
}

// ApiRoutes fonksiyonu API endpointlerini tanımlar
func ApiRoutes(app *fiber.App) {
	tokenService := pkg.NewTokenService()
	currencyService := pkg.NewCurrencyService()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowMethods:     "GET, POST, PATCH, DELETE",
		AllowCredentials: false,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello from Currency API! This is only for my personal use.")
	})

	app.Use(func(c *fiber.Ctx) error {
		return tokenService.Check(c)
	})

	app.Get("/token/create", tokenService.Generate)
	app.Get("/token/list", tokenService.List)
	app.Get("/currency", currencyService.CurrencyHandler)
}

// WebsocketHandler fonksiyonu websocket endpointini tanımlar
func WebsocketHandler(app *fiber.App) {
	currencyService := pkg.NewCurrencyService()

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		if c.Locals("allowed") == true {
			if err := currencyService.Updater(c); err != nil {
				c.Close()
			}
		}
	}))
}
