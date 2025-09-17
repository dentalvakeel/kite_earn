package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Response struct {
	NoBlockDeals        bool                `json:"noBlockDeals"`
	BulkBlockDeals      []BulkBlockDeal     `json:"bulkBlockDeals"`
	MarketDeptOrderBook MarketDeptOrderBook `json:"marketDeptOrderBook"`
	SecurityWiseDP      SecurityWiseDP      `json:"securityWiseDP"`
}

type BulkBlockDeal struct {
	Name string `json:"name"`
}

type MarketDeptOrderBook struct {
	TotalBuyQuantity  int         `json:"totalBuyQuantity"`
	TotalSellQuantity int         `json:"totalSellQuantity"`
	Open              float64     `json:"open"`
	Bid               []Order     `json:"bid"`
	Ask               []Order     `json:"ask"`
	TradeInfo         TradeInfo   `json:"tradeInfo"`
	ValueAtRisk       ValueAtRisk `json:"valueAtRisk"`
}

type Order struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type TradeInfo struct {
	TotalTradedVolume  json.Number `json:"totalTradedVolume"`
	TotalTradedValue   float64     `json:"totalTradedValue"`
	TotalMarketCap     float64     `json:"totalMarketCap"`
	Ffmc               float64     `json:"ffmc"`
	ImpactCost         float64     `json:"impactCost"`
	CMDailyVolatility  string      `json:"cmDailyVolatility"`
	CMAnnualVolatility string      `json:"cmAnnualVolatility"`
	MarketLot          string      `json:"marketLot"`
	ActiveSeries       string      `json:"activeSeries"`
}

type ValueAtRisk struct {
	SecurityVar       float64 `json:"securityVar"`
	IndexVar          float64 `json:"indexVar"`
	VarMargin         float64 `json:"varMargin"`
	ExtremeLossMargin float64 `json:"extremeLossMargin"`
	AdhocMargin       float64 `json:"adhocMargin"`
	ApplicableMargin  float64 `json:"applicableMargin"`
}

type SecurityWiseDP struct {
	QuantityTraded           int     `json:"quantityTraded"`
	DeliveryQuantity         int     `json:"deliveryQuantity"`
	DeliveryToTradedQuantity float64 `json:"deliveryToTradedQuantity"`
	SeriesRemarks            *string `json:"seriesRemarks"` // Pointer to handle null
	SecWiseDelPosDate        string  `json:"secWiseDelPosDate"`
}

var DeliveryQuantityTrend = make(map[string][]float64)

type Trend string

const (
	UpTrend   Trend = "↑"
	DownTrend Trend = "↓"
	FlatTrend Trend = "↔"
)

var DeliveryTrend = make(map[string]Trend)
var DeliveryValue = make(map[string]float64)

var cache = make(map[string]struct {
	value     float64
	timestamp time.Time
})

var count = make(map[string]int)
var globalRateLimitHit time.Time

// Function to make an HTTP call to the NSE historical data API
func fetchDeliveryToTradedQuantity(symbol string) float64 {

	if !globalRateLimitHit.IsZero() && time.Since(globalRateLimitHit) < 5*time.Minute {
		if cached, found := cache[symbol]; found {
			fmt.Println("Using cached value due to global rate limit for symbol:", symbol)
			return cached.value
		}
		return 0
	}
	// Check if the symbol is in the cache and if the cache is still valid
	if cached, found := cache[symbol]; found {
		if time.Since(cached.timestamp) < 25*time.Minute {
			return cached.value
		}
		// Clear the cache if it's older than 5 minutes
		delete(cache, symbol)
	}
	count[symbol]++
	url := fmt.Sprintf("https://www.nseindia.com/api/quote-equity?symbol=%s&section=trade_info", symbol)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return 0
	}
	req.Header.Add("Cookie", os.Getenv("Cookie"))
	req.Header.Add("User-Agent", "PostmanRuntime/7.43.0")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	// req.Header.Add("Postman-Token", "8ee56bdc-1204-46d1-a552-579dc75723c3")
	req.Header.Add("Host", "www.nseindia.com")
	req.Header.Add("Connection", "keep-alive")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		//
		return 0
	}
	defer res.Body.Close()

	var body []byte
	body, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		str := string(body)
		if includes := "Resource not found"; str == includes {
			fmt.Println("Resource not found for symbol:", symbol)
			return 0
		}
		getNSECookie()
		if includes := "Access Denied"; str == includes {
			fmt.Println("Rate limit exceeded. Please try after some time for symbol:", symbol)
			globalRateLimitHit = time.Now()
			if cached, found := cache[symbol]; found {
				return cached.value
			}
			return 0
		}
		fmt.Println("Error unmarshalling response:", err)
		return 0
	}

	// Cache the response
	cache[symbol] = struct {
		value     float64
		timestamp time.Time
	}{
		value:     response.SecurityWiseDP.DeliveryToTradedQuantity,
		timestamp: time.Now(),
	}

	fmt.Printf("Response Body:%s %f\n", symbol, response.SecurityWiseDP.DeliveryToTradedQuantity)
	return response.SecurityWiseDP.DeliveryToTradedQuantity
}

var every5MinChannel = time.Tick(5 * time.Minute)

func initDeliveryTrend() {
	go func() {
		for range every5MinChannel {
			for _, k := range instruments {
				time.Sleep(time.Second * 30)
				val := fetchDeliveryToTradedQuantity(k[0])
				DeliveryValue[k[0]] = val
				DeliveryQuantityTrend[k[0]] = append(DeliveryQuantityTrend[k[0]], val)
				if len(DeliveryQuantityTrend[k[0]]) > 20 {
					DeliveryQuantityTrend[k[0]] = DeliveryQuantityTrend[k[0]][:20]
				}
				DeliveryTrend[k[0]] = determineTrend(DeliveryQuantityTrend[k[0]])
			}
		}
	}()
}

func determineTrend(values []float64) Trend {
	if len(values) < 2 {
		return FlatTrend // Not enough data to determine trend
	}

	upCount := 0
	downCount := 0

	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1] {
			upCount++
		} else if values[i] < values[i-1] {
			downCount++
		}
	}

	if upCount > downCount {
		return UpTrend
	} else if downCount > upCount {
		return DownTrend
	} else {
		return FlatTrend
	}
}
