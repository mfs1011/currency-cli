package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mfs1011/currency-cli/internal/api"
	"github.com/mfs1011/currency-cli/internal/cache"
	"github.com/mfs1011/currency-cli/internal/config"
	"github.com/mfs1011/currency-cli/internal/display"
	"github.com/spf13/cobra"
)

var ratesCmd = &cobra.Command{
	Use:   "rates [CURRENCY]",
	Short: "Show exchange rates from all sources",
	Long: `Show current exchange rates from onmap.uz banks + CBU.

Examples:
  cx rates         # USD rates (default)
  cx rates EUR
  cx rates RUB`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRates,
}

var ratesSortFlag string

func init() {
	ratesCmd.Flags().StringVarP(&ratesSortFlag, "sort", "s", "buy", "sort by: buy or sell")
}

func runRates(cmd *cobra.Command, args []string) error {
	currency := "USD"
	if len(args) > 0 {
		currency = strings.ToUpper(args[0])
	}

	c, err := cache.New(config.Dir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cache unavailable: %v\n", err)
	}

	var rows []display.RateRow
	stale := false
	var staleAt time.Time

	// Try onmap.uz
	onmapKey := "onmap:" + currency
	var onmapData api.OnmapResponse

	onmapOK := c != nil && c.Get(onmapKey, 5*time.Minute, &onmapData)
	if !onmapOK {
		result, fetchErr := api.FetchOnmap(currency)
		if fetchErr != nil {
			// Try stale cache
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
		// Central bank first
		if onmapData.CentralBank != nil && onmapData.CentralBank.Buying != "" {
			title := onmapData.CentralBank.FullTitle
			if title == "" {
				title = onmapData.CentralBank.Title
			}
			rows = append(rows, display.RateRow{
				Source:  title,
				Buy:     onmapData.CentralBank.Buying,
				Sell:    onmapData.CentralBank.Selling,
				IsOnmap: true,
			})
		}
		for _, b := range onmapData.Banks {
			if b.Buying == "" && b.Selling == "" {
				continue
			}
			rows = append(rows, display.RateRow{
				Source:  b.FullTitle,
				Buy:     b.Buying,
				Sell:    b.Selling,
				IsOnmap: true,
			})
		}
	}

	// CBU rate
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
			if !cbuOK {
				fmt.Fprintf(os.Stderr, "cbu.uz error: %v\n", fetchErr)
			}
		} else {
			cbuRates = rates
			cbuOK = true
			if c != nil {
				_ = c.Set(cbuKey, cbuRates)
			}
		}
	}

	if cbuOK && currency != "UZS" {
		rate := api.FindCBURate(cbuRates, currency)
		if rate != nil {
			// Check if CBU already in rows (via onmap central_bank)
			hasCBU := false
			for _, r := range rows {
				if strings.Contains(r.Source, "банк") || strings.Contains(r.Source, "ЦБ") {
					hasCBU = true
					break
				}
			}
			_ = hasCBU
			// Add CBU as a separate authoritative row
			rows = append(rows, display.RateRow{
				Source:  fmt.Sprintf("CBU (official, %s)", rate.Date),
				Buy:     rate.Rate,
				Sell:    rate.Rate,
				IsOnmap: false,
			})
		}
	}

	if len(rows) == 0 {
		return fmt.Errorf("no data available for %s", currency)
	}

	display.PrintRatesTable(rows, currency, "UZS", stale, staleAt)
	return nil
}
