package main

import (
	"Currency/database"
	"Currency/model"
	"encoding/json"
	"io"
	"log"

	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	app.Get("/currency", CurrencyApiHandler)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":8080"))
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

func CurrencyApiHandler(c *fiber.Ctx) error {
	url := "https://api.freecurrencyapi.com/v1/latest?apikey=fca_live_u0sYK4DYWBJTH2FqyODV5rGrvhFcxGnKgSymXi5a&currencies=USD%2CEUR%2CGBP%2CPLN%2CRUB%2CJPY%2CKRW&base_currency=TRY"

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var currencyResp model.CurrencyResponse
	if err := json.Unmarshal(data, &currencyResp); err != nil {
		return err
	}

	var currencyError model.CurrencyError
	if err := json.Unmarshal(data, &currencyError); err != nil {
		return err
	}

	if currencyError.ErrorCode != 0 {
		return c.JSON(currencyError)
	}

	return c.JSON(currencyResp.Data["EUR"])
}
