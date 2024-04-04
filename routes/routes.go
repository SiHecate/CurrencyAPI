package routes

import (
	"Currency/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

func apiRoutes(app *fiber.App) {
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

func websocketHandler(app *fiber.App) {
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
