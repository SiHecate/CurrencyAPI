package main

import (
	"Currency/database"
	"Currency/model"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"net/http"

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

	database.Connect()

	app := fiber.New()

	app.Use(logger.New())
	app.Use(recover.New())

	router(app)
	websocketHandler(app)
	log.Fatal(app.Listen(":8080"))
}

func router(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowMethods:     "GET, POST, PATCH, DELETE",
		AllowCredentials: false,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello from Currency API! This is only for my personal use.")
	})

	app.Use(TokenCheck)
	app.Get("/currency", CurrencyHandler)
	app.Get("/token", TokenCreate)
	app.Get("/tokens", TokenList)
}

// Websocket handler
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
func WSSaveCurrencyToDatabase(ws *websocket.Conn) error {
	for {
		currentTime := time.Now()
		if ws == nil {
			fmt.Println("WebSocket bağlantısı hala başlatılmamış.")
			return errors.New("WebSocket bağlantısı başlatılmadı")
		}

		var existingCurrency model.Currency
		if err := database.Conn.First(&existingCurrency).Error; err != nil {
			fmt.Println("Para birimi bulunamadı. Hata:", err)
			continue
		}

		lastUpdatedTime := existingCurrency.UpdatedAt
		timeRemain := 10 - currentTime.Sub(lastUpdatedTime).Minutes()

		if timeRemain <= 0 {
			SaveCurrencyToDatabase()
			ws.WriteJSON("Döviz kurları güncellendi")
			ws.WriteJSON(existingCurrency)
			CurrencyConvertor()
		} else {
			ws.WriteJSON("Döviz kurları güncellenmesine kalan süre: " + fmt.Sprintf("%f", timeRemain) + " dakika")
			time.Sleep(5 * time.Second)
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

	// API Tarafından gelen döviz kurları
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
	var existingCurrency model.Currency
	if err := database.Conn.First(&existingCurrency).Error; err != nil {
		if err := database.Conn.Create(&currency).Error; err != nil {
			log.Fatal(err)
		}
	} else {
		if err := database.Conn.Model(&existingCurrency).Updates(&currency).Error; err != nil {
			log.Fatal(err)
		}
	}
}

// Para birimlerinin TL'ye çevrilmesi fonksiyonu
func CurrencyConvertor() {
	var existingCurrency model.Currency
	if err := database.Conn.First(&existingCurrency).Error; err != nil {
		log.Fatal(err)
	}
	currencies := map[string]float64{
		"EUR": existingCurrency.EUR,
		"GBP": existingCurrency.GBP,
		"JPY": existingCurrency.JPY,
		"KRW": existingCurrency.KRW,
		"PLN": existingCurrency.PLN,
		"RUB": existingCurrency.RUB,
		"USD": existingCurrency.USD,
	}
	turkishCurrencies := make(map[string]float64)
	for currency, value := range currencies {
		turkishValue := 1 / value
		turkishCurrencies[currency] = turkishValue
		database.Conn.Model(&existingCurrency).Update(currency, turkishValue)
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

	type CurrencyResponse struct {
		USD float64 `json:"usd"`
		EUR float64 `json:"eur"`
		GBP float64 `json:"gbp"`
		PLN float64 `json:"pln"`
		RUB float64 `json:"rub"`
		JPY float64 `json:"jpy"`
		KRW float64 `json:"krw"`
	}

	response := CurrencyResponse{
		USD: currency.USD,
		EUR: currency.EUR,
		GBP: currency.GBP,
		PLN: currency.PLN,
		RUB: currency.RUB,
		JPY: currency.JPY,
		KRW: currency.KRW,
	}

	return c.JSON(response)
}

func TokenCreate(c *fiber.Ctx) error {
	token := TokenGenerator()
	return c.JSON(fiber.Map{
		"token": token,
	})
}

func TokenGenerator() string {
	newToken := make([]byte, 12)
	_, err := rand.Read(newToken)
	if err != nil {
		log.Fatalf("Rastgele token oluşturulurken bir hata oluştu: %v", err)
	}
	tokenHex := hex.EncodeToString(newToken)

	if err := database.Conn.Create(&model.Token{Token: tokenHex}).Error; err != nil {
		log.Fatalf("Token database'e kaydedilirken bir hata oluştu: %v", err)
	}

	fmt.Println("Token:", tokenHex)
	return tokenHex
}

func TokenList(c *fiber.Ctx) error {
	var tokens []model.Token
	if err := database.Conn.Find(&tokens).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Token not found",
		})
	}

	return c.JSON(tokens)
}

// Token middleware
func TokenCheck(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token not found",
		})
	}

	var dbToken model.Token
	if err := database.Conn.Where("token = ?", token).First(&dbToken).Error; err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	return c.Next()
}
