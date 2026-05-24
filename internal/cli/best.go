package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mfs1011/currency-cli/internal/api"
	"github.com/mfs1011/currency-cli/internal/cache"
	"github.com/mfs1011/currency-cli/internal/config"
	"github.com/spf13/cobra"
)

var bestCmd = &cobra.Command{
	Use:   "best AMOUNT FROM TO",
	Short: "Find best exchange rate",
	Long: `Find the bank offering the best exchange rate.

Examples:
  cx best 100 USD UZS
  cx best 1000000 UZS USD`,
	Args: cobra.ExactArgs(3),
	RunE: runBest,
}

func runBest(cmd *cobra.Command, args []string) error {
	amount, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", args[0])
	}

	from := strings.ToUpper(args[1])
	to := strings.ToUpper(args[2])

	if to != "UZS" && from != "UZS" {
		return fmt.Errorf("best command supports UZS pairs only (e.g. USD/UZS). Use 'cx convert' for cross-currency.")
	}

	c, _ := cache.New(config.Dir())

	queryCur := from
	if from == "UZS" {
		queryCur = to
	}
	toUZS := to == "UZS"

	onmapKey := "onmap:" + queryCur
	var onmapData api.OnmapResponse
	onmapOK := c != nil && c.Get(onmapKey, 5*time.Minute, &onmapData)
	if !onmapOK {
		result, fetchErr := api.FetchOnmap(queryCur)
		if fetchErr != nil {
			if c != nil {
				ok, _ := c.GetStale(onmapKey, &onmapData)
				onmapOK = ok
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

	type candidate struct {
		name   string
		result float64
		rate   float64
	}

	var best candidate
	best.result = -1

	evaluate := func(name, buyStr, sellStr string) {
		buy, _ := strconv.ParseFloat(strings.ReplaceAll(buyStr, " ", ""), 64)
		sell, _ := strconv.ParseFloat(strings.ReplaceAll(sellStr, " ", ""), 64)
		var result, rate float64
		if toUZS {
			// customer sells FCY to bank → bank pays at buying rate
			if buy > 0 {
				result = amount * buy
				rate = buy
			}
		} else {
			// customer buys FCY from bank → bank sells at selling rate
			if sell > 0 {
				result = amount / sell
				rate = sell
			}
		}
		if result > best.result {
			best = candidate{name: name, result: result, rate: rate}
		}
	}

	if onmapOK {
		if onmapData.CentralBank != nil {
			title := onmapData.CentralBank.FullTitle
			if title == "" {
				title = "Марказий банк"
			}
			evaluate(title, onmapData.CentralBank.Buying, onmapData.CentralBank.Selling)
		}
		for _, b := range onmapData.Banks {
			evaluate(b.FullTitle, b.Buying, b.Selling)
		}
	}

	if best.result < 0 {
		return fmt.Errorf("no data available for %s → %s", from, to)
	}

	green := color.New(color.Bold, color.FgGreen)
	dim := color.New(color.FgHiBlack)

	green.Printf("Best: %s\n", best.name)
	fmt.Printf("  %.2f %s → ", amount, from)
	green.Printf("%.2f %s\n", best.result, to)
	if toUZS {
		dim.Printf("  Rate: 1 %s = %.2f UZS (buying rate)\n", queryCur, best.rate)
	} else {
		dim.Printf("  Rate: 1 %s = %.2f UZS (selling rate)\n", queryCur, best.rate)
	}

	return nil
}
