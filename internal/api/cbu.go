package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const cbuURL = "https://cbu.uz/uz/arkhiv-kursov-valyut/json/"

type CBURate struct {
	ID       int    `json:"id"`
	Code     string `json:"Code"`
	Ccy      string `json:"Ccy"`
	CcyNmRU  string `json:"CcyNm_RU"`
	CcyNmUZ  string `json:"CcyNm_UZ"`
	CcyNmEN  string `json:"CcyNm_EN"`
	Nominal  string `json:"Nominal"`
	Rate     string `json:"Rate"`
	Diff     string `json:"Diff"`
	Date     string `json:"Date"`
}

// RateFloat returns the rate as float64
func (r CBURate) RateFloat() float64 {
	v, _ := strconv.ParseFloat(r.Rate, 64)
	return v
}

// NominalInt returns the nominal as int
func (r CBURate) NominalInt() int {
	v, _ := strconv.Atoi(r.Nominal)
	if v == 0 {
		return 1
	}
	return v
}

// RatePerUnit returns rate per 1 unit of currency
func (r CBURate) RatePerUnit() float64 {
	return r.RateFloat() / float64(r.NominalInt())
}

func FetchCBU() ([]CBURate, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(cbuURL)
	if err != nil {
		return nil, fmt.Errorf("CBU request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CBU returned %d", resp.StatusCode)
	}

	var rates []CBURate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, fmt.Errorf("CBU parse error: %w", err)
	}

	return rates, nil
}

// FindCBURate returns the CBURate for a given currency code (case-insensitive)
func FindCBURate(rates []CBURate, currency string) *CBURate {
	upper := toUpper(currency)
	for i, r := range rates {
		if r.Ccy == upper {
			return &rates[i]
		}
	}
	return nil
}

func toUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		result[i] = c
	}
	return string(result)
}

// ConvertViaCBU converts amount from one currency to another using CBU rates
// Returns converted amount and error
func ConvertViaCBU(rates []CBURate, amount float64, from, to string) (float64, error) {
	fromUpper := toUpper(from)
	toUpper2 := toUpper(to)

	// UZS is the base currency for CBU
	var amountInUZS float64

	if fromUpper == "UZS" {
		amountInUZS = amount
	} else {
		fromRate := FindCBURate(rates, fromUpper)
		if fromRate == nil {
			return 0, fmt.Errorf("currency %s not found in CBU rates", from)
		}
		amountInUZS = amount * fromRate.RatePerUnit()
	}

	if toUpper2 == "UZS" {
		return amountInUZS, nil
	}

	toRate := FindCBURate(rates, toUpper2)
	if toRate == nil {
		return 0, fmt.Errorf("currency %s not found in CBU rates", to)
	}

	return amountInUZS / toRate.RatePerUnit(), nil
}

// UpdatedAt returns the date string from CBU response
func UpdatedAt(rates []CBURate) string {
	if len(rates) > 0 {
		return rates[0].Date
	}
	return ""
}

// ListCurrencies returns all available currency codes
func ListCurrencies(rates []CBURate) []string {
	codes := make([]string, 0, len(rates))
	for _, r := range rates {
		codes = append(codes, r.Ccy)
	}
	return codes
}
