package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

type PriceForecastResponse struct {
	Success int               `json:"success"`
	Data    PriceForecastData `json:"data"`
}

type PriceForecastData struct {
	GraphData   [][]float64 `json:"graphData"`
	High        string      `json:"high"`
	Mean        string      `json:"mean"`
	Low         string      `json:"low"`
	DisplayLock string      `json:"displayLock"`
}

type PriceForecast struct {
	Symbol        string
	CurrentPrice  float64
	OneYearTarget float64
	UpsidePercent float64
}

func fetchPriceForecast(symbol string) (PriceForecast, error) {
	scId := symbol
	url := fmt.Sprintf("https://api.moneycontrol.com/mcapi/v1/stock/estimates/price-forecast?scId=%s&ex=N&deviceType=W", scId)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PriceForecast{}, err
	}
	req.Header.Set("auth-token", "BzjBBOiKtB0savNfOXSZN96ffcSZtX69uJRhu4blKWhAudp9RwGfxPAZhNmU8kG37qpKenM0YSIhz6ctWhluIw")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.4 Safari/605.1.15")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.moneycontrol.com")
	req.Header.Set("Referer", "https://www.moneycontrol.com/")

	resp, err := client.Do(req)
	if err != nil {
		return PriceForecast{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PriceForecast{}, fmt.Errorf("status %d for %s", resp.StatusCode, symbol)
	}

	var pfResp PriceForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&pfResp); err != nil {
		return PriceForecast{}, err
	}

	// Extract current price from the last graphData point
	graph := pfResp.Data.GraphData
	var currentPrice float64
	if len(graph) > 0 && len(graph[len(graph)-1]) > 1 {
		currentPrice = graph[len(graph)-1][1]
	}

	// Parse target price (mean) and calculate upside
	mean, _ := strconv.ParseFloat(pfResp.Data.Mean, 64)
	var upside float64
	if currentPrice > 0 {
		upside = ((mean - currentPrice) / currentPrice) * 100
	}

	return PriceForecast{
		Symbol:        symbol,
		CurrentPrice:  currentPrice,
		OneYearTarget: mean,
		UpsidePercent: upside,
	}, nil
}

func printAllPriceForecasts() {
	fileName := fmt.Sprintf("PriceForecasts_%s", time.Now().Format("02-01-2006"))
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer f.Close()
	w := tabwriter.NewWriter(f, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Stock Symbol\tCurrent Price (₹)\t1-Year Target Price (₹)\t1-Year Upside (%)")

	type forecastRow struct {
		Symbol        string
		CurrentPrice  float64
		OneYearTarget float64
		UpsidePercent float64
	}

	var forecasts []forecastRow

	for _, v := range instruments {
		if len(v) < 2 {
			continue
		}
		forecast, err := fetchPriceForecast(v[1])
		if err != nil {
			continue
		}
		forecasts = append(forecasts, forecastRow{
			Symbol:        v[0],
			CurrentPrice:  forecast.CurrentPrice,
			OneYearTarget: forecast.OneYearTarget,
			UpsidePercent: forecast.UpsidePercent,
		})
		time.Sleep(1 * time.Second)
	}

	// Sort in descending order of UpsidePercent
	sort.Slice(forecasts, func(i, j int) bool {
		return forecasts[i].UpsidePercent > forecasts[j].UpsidePercent
	})

	for _, forecast := range forecasts {
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f%%\n",
			forecast.Symbol, forecast.CurrentPrice, forecast.OneYearTarget, forecast.UpsidePercent)
	}
	w.Flush()
}
