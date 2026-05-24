package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mfs1011/currency-cli/internal/api"
	"github.com/mfs1011/currency-cli/internal/cache"
	"github.com/mfs1011/currency-cli/internal/config"
	"github.com/mfs1011/currency-cli/internal/display"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert AMOUNT FROM TO",
	Short: "Convert amount across all bank rates",
	Long: `Convert currency and show results from all banks.

Examples:
  cx convert 100 USD UZS
  cx convert 1000000 UZS EUR
  cx convert 50 EUR RUB`,
	Args: cobra.ExactArgs(3),
	RunE: runConvert,
}

func stripSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func runConvert(cmd *cobra.Command, args []string) error {
	amount, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", args[0])
	}

	from := strings.ToUpper(args[1])
	to := strings.ToUpper(args[2])

	// Only UZS as base for onmap.uz
	if to != "UZS" && from != "UZS" {
		return convertCrossCurrency(amount, from, to)
	}

	c, _ := cache.New(config.Dir())

	var rows []display.ConvertRow
	stale := false
	var staleAt time.Time

	// Determine query currency for onmap (always the non-UZS one)
	queryCur := from
	if from == "UZS" {
		queryCur = to
	}
	toUZS := to == "UZS" // direction: are we converting TO uzs?

	onmapKey := "onmap:" + queryCur
	var onmapData api.OnmapResponse
	onmapOK := c != nil && c.Get(onmapKey, 5*time.Minute, &onmapData)
	if !onmapOK {
		result, fetchErr := api.FetchOnmap(queryCur)
		if fetchErr != nil {
			if c != nil {
				if ok, at := c.GetStale(onmapKey, &onmapData); ok {
					stale = true
					staleAt = at
					onmapOK = true
				}
			}
			if !onmapOK {
				fmt.Fprintf(os.Stderr, "onmap.uz error: %v\n", fetchErr)
			}
		} else {
			onmapData = *result
			onmapOK = true
			if c != nil {
				_ = c.Set(onmapKey, onmapData)
			}
		}
	}

	if onmapOK {
		calcRow := func(title, buyStr, sellStr string) display.ConvertRow {
			buy, _ := strconv.ParseFloat(stripSpaces(buyStr), 64)
			sell, _ := strconv.ParseFloat(stripSpaces(sellStr), 64)
			var buyResult, sellResult float64
			if toUZS {
				// selling FCY → UZS: bank's sell rate applies
				if buy > 0 {
					buyResult = amount * buy
				}
				if sell > 0 {
					sellResult = amount * sell
				}
			} else {
				// buying UZS → FCY: bank's buy rate applies
				if buy > 0 {
					buyResult = amount / buy
				}
				if sell > 0 {
					sellResult = amount / sell
				}
			}
			return display.ConvertRow{Source: title, BuyResult: buyResult, SellResult: sellResult}
		}

		if onmapData.CentralBank != nil && onmapData.CentralBank.Buying != "" {
			title := onmapData.CentralBank.FullTitle
			if title == "" {
				title = "Марказий банк (ЦБ)"
			}
			rows = append(rows, calcRow(title, onmapData.CentralBank.Buying, onmapData.CentralBank.Selling))
		}
		for _, b := range onmapData.Banks {
			if b.Buying == "" && b.Selling == "" {
				continue
			}
			rows = append(rows, calcRow(b.FullTitle, b.Buying, b.Selling))
		}
	}

	// CBU official rate
	cbuKey := "cbu:all"
	var cbuRates []api.CBURate
	cbuOK := c != nil && c.Get(cbuKey, 24*time.Hour, &cbuRates)
	if !cbuOK {
		rates, fetchErr := api.FetchCBU()
		if fetchErr != nil {
			if c != nil {
				if ok, at := c.GetStale(cbuKey, &cbuRates); ok {
					if !stale {
						stale = true
						staleAt = at
					}
					cbuOK = true
				}
			}
		} else {
			cbuRates = rates
			cbuOK = true
			if c != nil {
				_ = c.Set(cbuKey, cbuRates)
			}
		}
	}

	if cbuOK {
		result, convErr := api.ConvertViaCBU(cbuRates, amount, from, to)
		if convErr == nil {
			rows = append(rows, display.ConvertRow{
				Source:     "CBU (official)",
				BuyResult:  result,
				SellResult: result,
			})
		}
	}

	if len(rows) == 0 {
		return fmt.Errorf("no data available for %s → %s", from, to)
	}

	display.PrintConvertTable(os.Stdout, rows, amount, from, to, stale, staleAt)
	return nil
}

// convertCrossCurrency handles non-UZS pairs via CBU rates
func convertCrossCurrency(amount float64, from, to string) error {
	c, _ := cache.New(config.Dir())
	cbuKey := "cbu:all"
	var cbuRates []api.CBURate
	cbuOK := c != nil && c.Get(cbuKey, 24*time.Hour, &cbuRates)
	if !cbuOK {
		rates, err := api.FetchCBU()
		if err != nil {
			if c != nil {
				ok, _ := c.GetStale(cbuKey, &cbuRates)
				cbuOK = ok
			}
		} else {
			cbuRates = rates
			cbuOK = true
			if c != nil {
				_ = c.Set(cbuKey, cbuRates)
			}
		}
	}
	if !cbuOK {
		return fmt.Errorf("no data for cross-currency conversion %s → %s", from, to)
	}

	result, err := api.ConvertViaCBU(cbuRates, amount, from, to)
	if err != nil {
		return err
	}

	rows := []display.ConvertRow{{
		Source:     "CBU (via UZS)",
		BuyResult:  result,
		SellResult: result,
	}}
	display.PrintConvertTable(os.Stdout, rows, amount, from, to, false, time.Time{})
	return nil
}
