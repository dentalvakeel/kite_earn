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
	path := fmt.Sprintf("https://kite.zerodha.com/oms/instruments/historical/%d/day", instrument)
	baseURL, err := url.Parse(path) // Replace with your base URL
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}
	params := url.Values{}
	params.Add("user_id", "YA0828")
	params.Add("oi", "1")
	// lastOneYear := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	lastoneweek := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	// if time is after 3:30 pm set yesterday to today
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
	// fmt.Println("Response Status:", string(body))
	var hist History
	if err := json.Unmarshal(body, &hist); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
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

	fmt.Printf("Trend: %f, Buy Volume: %d, Sell Volume: %d\n", accumilationTrend, buyVolume, sellVolume)

	// fmt.Printf("Instrument: %s, Sell Volume: %d, Buy Volume: %d\n", instruments[instrument], sellVolume, buyVolume)
	// Check if the last three values of accumilationTrend are increasing
	if len(accumilationTrend) >= 3 {
		n := len(accumilationTrend)
		if accumilationTrend[n-3] < accumilationTrend[n-2] && accumilationTrend[n-2] < accumilationTrend[n-1] {
			fmt.Printf("\033[32m Instrument: %s, Increasing Trend: %v\n \033[0m", instruments[instrument], accumilationTrend[n-3:])
		}
	}
	if buyVolume == 0 {
		fmt.Println("No buy volume")
		return
	}
	if buyVolume > sellVolume {
		accumilationpercent := float32(buyVolume) / float32(sellVolume)
		if accumilationpercent > 1.5 {
			fmt.Printf("Accumulation for %s %f \n", instruments[instrument], accumilationpercent)
		}
		if accumilationpercent < 1 {
			fmt.Printf("May be selling for %s %f \n", instruments[instrument], accumilationpercent)
		}
		accumilation1Week[instrument] = accumilationpercent
	}

	// if sellVolume > buyVolume {
	// 	fmt.Printf("No accumulation for %s %f \n", instruments[instrument], float32(buyVolume-sellVolume)/float32(buyVolume)*100)
	// } else {
	// 	fmt.Printf("Accumulation for %s %f \n", instruments[instrument], float32(buyVolume-sellVolume)/float32(buyVolume)*100)
	// }
	fmt.Println("----------------------------------------")
}
