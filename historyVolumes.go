package main

import (
	"encoding/json"
	"fmt"
	"io"
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
var top10VolumesDates = map[string][]string{}

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
	var hist History
	if err := json.Unmarshal(body, &hist); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
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
	for _, k := range volumes[:10] {
		v, ok := volumeDatemap[k]
		if ok {
			dates = append(dates, v)
		}
	}
	top10Volumes[instruments[instrument]] = volumes[:10]
	top10VolumesDates[instruments[instrument]] = dates
	// fmt.Println(volumes[:20])
}
