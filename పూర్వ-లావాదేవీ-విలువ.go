package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"
)

type History struct {
	Status string `json:"status"`
	Data   struct {
		Candles [][]any `json:"candles"`
	} `json:"data"`
}

var top10Volumes = map[string][]uint32{}
var top40Volumes = map[string][]uint32{}
var top10VolumesDates = map[string][]string{}
var incrementForLastThreeDays = map[string]string{}
var incrementForLastThreeWeeks = map[string]string{}
var dmaValues = map[string]float64{}

func getHistory(instrument uint32) {
	path := fmt.Sprintf("https://kite.zerodha.com/oms/instruments/historical/%d/day", instrument)
	baseURL, err := url.Parse(path) // Replace with your base URL
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}
	params := url.Values{}
	params.Add("user_id", "YA0828")
	params.Add("oi", "1")
	today := time.Now().Format("2006-01-02")
	lastOneYear := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	params.Add("from", lastOneYear)
	params.Add("to", today)
	baseURL.RawQuery = params.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add headers
	token := fmt.Sprintf("enctoken %s", os.Getenv("enctoken"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token) // Replace with your authorization token, if needed

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body) // response body is []byte
	// fmt.Println(string(body))        // Print the response body for debugging
	var hist History
	if err := json.Unmarshal(body, &hist); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	// write the historical data in to a csv file with name of file as stock name
	file, err := os.Create(fmt.Sprintf("backtest/historical_data_%s.csv", instruments[instrument][0]))
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	writer.Write([]string{"Date", "Open", "High", "Low", "Close", "Volume"})
	for _, candle := range hist.Data.Candles {
		writer.Write([]string{
			string(candle[0].(string)),
			fmt.Sprintf("%f", candle[1].(float64)),
			fmt.Sprintf("%f", candle[2].(float64)),
			fmt.Sprintf("%f", candle[3].(float64)),
			fmt.Sprintf("%f", candle[4].(float64)),
			fmt.Sprintf("%d", uint32(candle[5].(float64))),
		})
	}
	var volumes []uint32 // Slice to store volumes
	var volumeDatemap = map[uint32]string{}
	for _, candle := range hist.Data.Candles {
		volume := uint32(candle[5].(float64))
		volumes = append(volumes, volume)
		volumeDatemap[volume] = string(candle[0].(string))
	}

	// Sort volumes in descending order
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i] > volumes[j]
	})
	if len(volumes) < 10 {
		return
	}

	var dates []string
	for _, k := range volumes[:20] {
		v, ok := volumeDatemap[k]
		if ok {
			t, err := time.Parse("2006-01-02T15:04:05-0700", v)
			if err != nil {
				panic(err)
			}
			// fmt.Println(t.Format("2006-01-02"), v)
			dates = append(dates, t.Format("2006-01-02"))
		}
	}
	top10Volumes[instruments[instrument][0]] = volumes[:20]
	top40Volumes[instruments[instrument][0]] = volumes[:40]
	top10VolumesDates[instruments[instrument][0]] = dates

	CalculateAndStoreDMA(instrument, hist)
	increasingThreeDays(instrument, hist)
	// fmt.Println(volumes[:20])
}

func getWeeklyHistory(instrument uint32) {
	path := fmt.Sprintf("https://kite.zerodha.com/oms/instruments/historical/%d/week", instrument)
	baseURL, err := url.Parse(path) // Replace with your base URL
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}
	params := url.Values{}
	params.Add("user_id", "YA0828")
	params.Add("oi", "1")
	today := time.Now().Format("2006-01-02")
	lastOneYear := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	params.Add("from", lastOneYear)
	params.Add("to", today)
	baseURL.RawQuery = params.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add headers
	token := fmt.Sprintf("enctoken %s", os.Getenv("enctoken"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token) // Replace with your authorization token, if needed

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(body))
	var hist History
	err = json.Unmarshal(body, &hist)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
	if len(hist.Data.Candles) < 3 {
		return
	}
	var closingPrices []float64
	for i := len(hist.Data.Candles) - 4; i < len(hist.Data.Candles)-1; i++ {
		closingPrice := hist.Data.Candles[i][4].(float64)
		closingPrices = append(closingPrices, closingPrice)
	}
	increased := true
	for i := 1; i < len(closingPrices); i++ {
		if closingPrices[i] <= closingPrices[i-1] {
			increased = false
			break
		}
	}
	if increased {
		incrementForLastThreeWeeks[instruments[instrument][0]] = "Yes"
	} else {
		incrementForLastThreeWeeks[instruments[instrument][0]] = "No"
	}
}

func calculateDMA(instrument uint32, hist History, period int) float64 {
	var closingPrices []float64
	for _, candle := range hist.Data.Candles {
		closingPrice := candle[4].(float64)
		closingPrices = append(closingPrices, closingPrice)
	}

	if len(closingPrices) < period {
		fmt.Printf("Insufficient data for %d-day DMA for instrument %s\n", period, instruments[instrument][0])
		return 0.0 // Return an empty slice
	}

	var dma float64
	// for i := period - 1; i < len(closingPrices); i++ {
	// 	sum := 0.0
	// 	for j := i - period + 1; j <= i; j++ {
	// 		sum += closingPrices[j]
	// 	}
	// 	dma = append(dma, sum/float64(period))
	// }
	sum := 0.0
	for i := len(closingPrices) - 1; i >= len(closingPrices)-period; i-- {
		sum += closingPrices[i]
		// dma = sum / float64(period)
	}
	dma = sum / float64(period)
	// max two digits after point
	dma = math.Round(dma*100) / 100
	return dma
}

func CalculateAndStoreDMA(instrument uint32, hist History) {
	dma50 := calculateDMA(instrument, hist, 50)
	dma200 := calculateDMA(instrument, hist, 200)

	dmaValues[instruments[instrument][0]+"_50DMA"] = dma50
	dmaValues[instruments[instrument][0]+"_200DMA"] = dma200

	if dma50 == 0 || dma200 == 0 {
		return
	}

	lastPrice := hist.Data.Candles[len(hist.Data.Candles)-1][4].(float64)
	lastVolume := hist.Data.Candles[len(hist.Data.Candles)-1][5].(float64)
	isInTop40Volumes := false
	isCurrentPricenear50DMA := lastPrice > dmaValues[instruments[instrument][0]+"_50DMA"]*0.96 && lastPrice < dmaValues[instruments[instrument][0]+"_50DMA"]*1.04
	// isCurrentPricenear200DMA := lastPrice > dmaValues[instruments[instrument][0]+"_200DMA"]*0.97 && lastPrice < dmaValues[instruments[instrument][0]+"_200DMA"]*1.03
	for _, vol := range top40Volumes[instruments[instrument][0]] {
		if uint32(lastVolume) > vol {
			isInTop40Volumes = true
			break
		}
	}
	if isInTop40Volumes && isCurrentPricenear50DMA {
		// write to file 50DMA picks
		f, err := os.OpenFile("50DMA.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		s := fmt.Sprintf("%s,%f,%f,%f,%f\n", instruments[instrument][0], lastPrice, lastVolume, dmaValues[instruments[instrument][0]+"_50DMA"], dmaValues[instruments[instrument][0]+"_200DMA"])
		f.WriteString(s)
	}
}

func increasingThreeDays(instrument uint32, hist History) {
	var closingPrices []float64
	if len(hist.Data.Candles) < 3 {
		return
	}
	for i := len(hist.Data.Candles) - 3; i < len(hist.Data.Candles); i++ {
		closingPrice := hist.Data.Candles[i][4].(float64)
		closingPrices = append(closingPrices, closingPrice)
	}

	increased := true
	for i := 1; i < len(closingPrices); i++ {
		if closingPrices[i] <= closingPrices[i-1] {
			increased = false
			break
		}
	}
	if increased {
		incrementForLastThreeDays[instruments[instrument][0]] = "Yes"
	} else {
		incrementForLastThreeDays[instruments[instrument][0]] = "No"
	}
}
