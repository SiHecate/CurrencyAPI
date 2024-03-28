package model

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
