package model

import "gorm.io/gorm"

/*
	{
	"data": {
		"EUR": 0.0286324743,
		"GBP": 0.0245400442,
		"JPY": 4.6819990076,
		"KRW": 41.676175464,
		"PLN": 0.1234590689,
		"RUB": 2.8670008947,
		"USD": 0.0309610521
		}
	}
*/

type CurrencyResponse struct {
	Data map[string]float64 `json:"data"`
}

type CurrencyError struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type Currency struct {
	gorm.Model
	EUR float64 `json:"eur"`
	GBP float64 `json:"gbp"`
	JPY float64 `json:"jpy"`
	KRW float64 `json:"krw"`
	PLN float64 `json:"pln"`
	RUB float64 `json:"rub"`
	USD float64 `json:"usd"`
}
