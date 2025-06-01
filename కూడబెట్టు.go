package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// function to find if stock is accumilated over period of week, 3 weeks and month
// algo is get history volume and devliery percent and if closing price < than previous day closing price
// call it sell volume else buy volume
// do summation of both sell volumes and buy volumes if sell volume is greater than buy volume then no accumilation

var accumilation1Week = make(map[uint32]float32)

func findAccumulation1Week(instrument uint32) {
	// Generate a unique file name with a timestamp
	timestamp := time.Now().Format("02-01-2006")
	fileName := fmt.Sprintf("accumilations_%s.txt", timestamp)

	// Open the file for writing
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	path := fmt.Sprintf("https://kite.zerodha.com/oms/instruments/historical/%d/day", instrument)
	baseURL, err := url.Parse(path)
	if err != nil {
		fmt.Fprintf(file, "Error parsing URL: %v\n", err)
		return
	}
	params := url.Values{}
	params.Add("user_id", "YA0828")
	params.Add("oi", "1")
	lastoneweek := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	hour := time.Now().Hour()
	minute := time.Now().Minute()
	if hour == 15 && minute > 30 {
		yesterday = time.Now().Format("2006-01-02")
	}
	if hour > 15 {
		yesterday = time.Now().Format("2006-01-02")
	}
	params.Add("from", lastoneweek)
	params.Add("to", yesterday)
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		fmt.Fprintf(file, "Error creating request: %v\n", err)
		return
	}

	token := fmt.Sprintf("enctoken %s", os.Getenv("enctoken"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(file, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var hist History
	if err := json.Unmarshal(body, &hist); err != nil {
		fmt.Fprintf(file, "Cannot unmarshal JSON: %v\n", err)
		fmt.Fprintln(file, "----------------------------------------")
		return
	}

	var sellVolume, buyVolume uint32
	var accumilationTrend []float32
	for i := 1; i < len(hist.Data.Candles); i++ {
		currentClose := hist.Data.Candles[i][4].(float64)
		previousClose := hist.Data.Candles[i-1][4].(float64)
		volume := uint32(hist.Data.Candles[i][5].(float64))

		if currentClose < previousClose {
			sellVolume += volume
		} else {
			buyVolume += volume
		}
		accumilationTrend = append(accumilationTrend, float32(buyVolume)/float32(sellVolume))
	}

	fmt.Fprintf(file, "Instrument:%s, Trend: %v, Buy Volume: %d, Sell Volume: %d\n", instruments[instrument], accumilationTrend, buyVolume, sellVolume)

	// fmt.Printf("Instrument: %s, Sell Volume: %d, Buy Volume: %d\n", instruments[instrument], sellVolume, buyVolume)
	// Check if the last three values of accumilationTrend are increasing
	if len(accumilationTrend) >= 3 {
		n := len(accumilationTrend)
		if accumilationTrend[n-3] < accumilationTrend[n-2] && accumilationTrend[n-2] < accumilationTrend[n-1] {
			fmt.Fprintf(file, "Instrument: %s, Increasing Trend: %v\n", instruments[instrument], accumilationTrend[n-3:])
		}
	}
	if buyVolume == 0 {
		// print in red color
		fmt.Fprintf(file, "No buy volume for %s %v\n", instruments[instrument], accumilationTrend)
		fmt.Println("No buy volume")
		return
	}
	if buyVolume > sellVolume {
		accumilationpercent := float32(buyVolume) / float32(sellVolume)
		if accumilationpercent > 1.5 {
			fmt.Fprintf(file, "Accumulation for %s %f\n", instruments[instrument], accumilationpercent)
		} else {
			fmt.Fprintf(file, "May be selling for %s %f\n", instruments[instrument], accumilationpercent)
		}
		accumilation1Week[instrument] = accumilationpercent
	}

	fmt.Fprintln(file, "----------------------------------------")
}
