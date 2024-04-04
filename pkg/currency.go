package service

import (
	"Currency/database"
	"Currency/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type CurrencyService interface {
	Updater(ws *websocket.Conn) error
	CurrencySave()
	CurrencyConvertor()
	CurrencyHandler(c *fiber.Ctx) error
}

type currencyService struct{}

func NewCurrencyService() CurrencyService {
	return &currencyService{}
}

// Updater fonksiyonu döviz kurlarını güncellemek için kullanılır.
func (s *currencyService) Updater(ws *websocket.Conn) error {
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
			s.CurrencySave()
			ws.WriteJSON("Döviz kurları güncellendi")
			ws.WriteJSON(existingCurrency)
			s.CurrencyConvertor()
		} else {
			ws.WriteJSON(fmt.Sprintf("Döviz kurları güncellenmesine kalan süre: %f dakika", timeRemain))
			time.Sleep(5 * time.Second)
		}
	}
}

// CurrencySave fonksiyonu döviz kurlarını API'den çekerek database'e kaydeder.
func (s *currencyService) CurrencySave() {
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

// CurrencyConvertor fonksiyonu döviz kurlarını TL'ye çevirir.
func (s *currencyService) CurrencyConvertor() {
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

// CurrencyHandler fonksiyonu döviz kurlarını döndürür.
func (s *currencyService) CurrencyHandler(c *fiber.Ctx) error {
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
