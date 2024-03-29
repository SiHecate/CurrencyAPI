package main

import (
	"Currency/database"
	"Currency/model"
	"encoding/json"
	"io"
	"log"
	"time"

	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	database.Connect()

	app := fiber.New()

	app.Use(logger.New())
	app.Use(recover.New())

	router(app)
	websocketHandler(app)

	log.Fatal(app.Listen(":8080"))
}

func router(app *fiber.App) {
	// Route definitions
	app.Get("/currency", CurrencyHandler)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
}

func trying(ws *websocket.Conn) {
	ws.WriteJSON("Döviz kurları güncelleniyor...")
}

func websocketHandler(app *fiber.App) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		if c.Locals("allowed") == true {
			trying(c)
			WSSaveCurrencyToDatabase(c)
		}
	}))
}

/*
	Algoritma konsepti:
	JS Tarafından Golang API'ya istek atarken istekte istenilen döviz cinsini belirtmek gerekmektedir.
	Örneğin EUR döviz cinsi için istek atıldığında, API tarafından EUR döviz kuru döndürülecektir.

	Kullanılan API aylık olarak 5000 istek hakkı sunmaktadır. Bu nedenle aylık 5000 istek hakkı aşılmamalıdır. Bunun içinde 10 dakikada bir gelen döviz kurlarını database'e kaydedip kullanıcıya döviz kurlarını database üzerinden sunabiliriz.

	Websocket yardımı ile 10 dakika bir döviz kurlarını güncelleyebiliriz.
	Bu da ortalama olarak günde 144 istek yapmamıza olanak sağlar. Ayda 4320 istek yapmış oluruz. Bu sayede aylık 5000 istek hakkımızı aşmamış oluruz.

	Websocket'i database ile birlikte kullanarak, database'e kaydedeceğiz herhangi bir dışarıdan isteğe açık olmamalı ve sadece websocket üzerinden database'e kayıt yapılmalıdır.
*/

func WSSaveCurrencyToDatabase(ws *websocket.Conn) {

	ws.WriteJSON("Döviz kurları güncelleniyor2...")

	for {
		currentTime := time.Now()
		if err := database.Conn.FirstOrCreate(&model.Currency{}).Error; err != nil {
			lastUpdatedTime := model.Currency{}.UpdatedAt
			timeRemain := 10 - currentTime.Sub(lastUpdatedTime).Minutes()
			ws.WriteJSON(timeRemain)
			if timeRemain >= 10 {
				ws.WriteJSON("Döviz kurları güncelleniyor...")
				SaveCurrencyToDatabase()
			} else {
				time.Sleep(time.Duration(timeRemain) * time.Minute)
			}
		}
	}
}

// Para birimlerinin database'e kaydedilmesi fonksiyonu
func SaveCurrencyToDatabase() {
	url := "https://api.freecurrencyapi.com/v1/latest?apikey=fca_live_u0sYK4DYWBJTH2FqyODV5rGrvhFcxGnKgSymXi5a&currencies=USD%2CEUR%2CGBP%2CPLN%2CRUB%2CJPY%2CKRW&base_currency=TRY"

	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var currencyResp model.CurrencyResponse
	if err := json.Unmarshal(data, &currencyResp); err != nil {
		log.Fatal(err)
	}

	var currencyError model.CurrencyError
	if err := json.Unmarshal(data, &currencyError); err != nil {
		log.Fatal(err)
	}

	if currencyError.ErrorCode != 0 {
		log.Fatal(currencyError)
	}

	currency := model.Currency{
		EUR: currencyResp.Data["EUR"],
		GBP: currencyResp.Data["GBP"],
		JPY: currencyResp.Data["JPY"],
		KRW: currencyResp.Data["KRW"],
		PLN: currencyResp.Data["PLN"],
		RUB: currencyResp.Data["RUB"],
		USD: currencyResp.Data["USD"],
	}

	// Para birimlerinin database içerisinde olup olmadığına göre ilk kaydın yapılması eğer varsa güncellenmesi
	if err := database.Conn.First(&model.Currency{}).Error; err != nil {
		if err := database.Conn.Create(&currency).Error; err != nil {
			log.Fatal(err)
		}
	} else {
		if err := database.Conn.Model(&model.Currency{}).Updates(&currency).Error; err != nil {
			log.Fatal(err)
		}
	}
}

// Döviz kurlarının database içerisinden getirilmesi fonksiyonu
func CurrencyHandler(c *fiber.Ctx) error {
	currency := model.Currency{}
	if err := database.Conn.First(&currency).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Currency not found",
		})
	}

	return c.JSON(currency)
}
