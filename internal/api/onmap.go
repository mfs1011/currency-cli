package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const onmapBase = "https://onmap.uz/api"

type Bank struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	FullTitle string `json:"full_title"`
	Slug     string `json:"slug"`
	Buying   string `json:"buying"`
	Selling  string `json:"selling"`
	Status   int    `json:"status"`
}

type OnmapResponse struct {
	Banks       []Bank `json:"banks"`
	CentralBank *Bank  `json:"central_bank"`
	Date        string `json:"date"`
}

func FetchOnmap(currency string) (*OnmapResponse, error) {
	cur := strings.ToLower(currency)
	url := fmt.Sprintf("%s/banks?cur=%s&take=all&lang=ru", onmapBase, cur)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("onmap request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("onmap returned %d", resp.StatusCode)
	}

	var result OnmapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("onmap parse error: %w", err)
	}

	// inject CBU as a bank entry if present
	if result.CentralBank != nil && result.CentralBank.Title == "" {
		result.CentralBank.Title = "ЦБ"
		result.CentralBank.FullTitle = "Марказий банк"
	}

	return &result, nil
}

// FetchBestRates fetches best-rates summary for a currency
func FetchBestRates(currency string) (*OnmapResponse, error) {
	cur := strings.ToLower(currency)
	url := fmt.Sprintf("%s/best-rates?lang=ru&cur=%s", onmapBase, cur)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("onmap best-rates request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("onmap best-rates returned %d", resp.StatusCode)
	}

	var result OnmapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("onmap best-rates parse error: %w", err)
	}

	return &result, nil
}
