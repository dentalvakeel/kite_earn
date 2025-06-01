package main

import (
	"fmt"
	"os"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kitemodels "github.com/zerodha/gokiteconnect/v4/models"

	//kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
	"sync"

	"github.com/joho/godotenv"
)

var (
	ticker     *Ticker
	tickerchan = make(chan kitemodels.Tick)
)

var (
	instToken = []uint32{}
)

// Triggered when any error is raised
func onError(err error) {
	fmt.Println("Error: ", err)
}

// Triggered when websocket connection is closed
func onClose(code int, reason string) {
	fmt.Println("Close: ", code, reason)
}

func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

// Triggered when connection is established and ready to send and accept data
func onConnect() {
	fmt.Println("Connected")
	err := ticker.Subscribe(instToken)
	if err != nil {
		fmt.Println("err: ", err)
	}
	// Set subscription mode for the subscribed token
	// Default mode is Quote
	err = ticker.SetMode(ModeFull, instToken)
	if err != nil {
		fmt.Println("err: ", err)
	}
}

var fiveMinTicker = time.NewTicker(60 * time.Second)
var writtenInst = map[uint32]bool{}
var mu sync.Mutex

func updateWrittenInst(key uint32, value bool) {
	mu.Lock()         // Lock the mutex before accessing writtenInst
	defer mu.Unlock() // Ensure the mutex is unlocked after the function returns

	// Update the shared resource
	writtenInst[key] = value
}

func readWrittenInst(key uint32) bool {
	mu.Lock()         // Lock the mutex before accessing writtenInst
	defer mu.Unlock() // Ensure the mutex is unlocked after the function returns

	// Read from the shared resource
	value := writtenInst[key]
	return value
}

// Triggered when tick is recevived
func onTick(tick kitemodels.Tick) {
	// fmt.Println("Tick: ", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity)
	tickerchan <- tick
}

func initiateTickerChannellistener() {
	for tick := range tickerchan {
		writeToFile(tick)
		writeGTVolumesToDashboard(tick)
	}

	select {

	case <-fiveMinTicker.C:
		// fmt.Println("Tick: ", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity)
		for key := range writtenInst {
			updateWrittenInst(key, false)
		}
	}
}

// Triggered when reconnection is attempted which is enabled by default
func onReconnect(attempt int, delay time.Duration) {
	fmt.Printf("Reconnect attempt %d in %fs\n", attempt, delay.Seconds())
}

// Triggered when maximum number of reconnect attempt is made and the program is terminated
func onNoReconnect(attempt int) {
	fmt.Printf("Maximum no of reconnect attempt reached: %d", attempt)
}

// Triggered when order update is received
func onOrderUpdate(order kiteconnect.Order) {
	fmt.Println("Order: ", order.OrderID)
}

func kiteCandleCalls() {
	for k := range instruments {
		instToken = append(instToken, k)
		time.Sleep(1 * time.Second)
		getHistory(k)
		getWeeklyHistory(k)
		findAccumulation1Week(k)
		// fetchDeliveryToTradedQuantity(instruments[k])
	}
	for k := range top10Volumes {
		fmt.Println(k, top10Volumes[k])
		fmt.Println(k, top10VolumesDates[k])
		fmt.Println("************************")
	}
	fmt.Println(incrementForLastThreeDays)
	fmt.Println(incrementForLastThreeWeeks)
}

func main() {

	testMode := os.Getenv("TEST_MODE")

	// Restrict to one stock in test mode
	if testMode == "true" {
		instruments = map[uint32]string{
			7398145: "KITEX",
		}
	}
	apiKey := "my_api_key"
	accessToken := "my_access_token"
	kiteCandleCalls()
	getNSECookie()
	fetchBulkDeals()
	initDeliveryTrend()
	// Create new Kite ticker instance
	ticker = New(apiKey, accessToken)

	// Assign callbacks
	ticker.OnError(onError)
	ticker.OnClose(onClose)
	ticker.OnConnect(onConnect)
	ticker.OnReconnect(onReconnect)
	ticker.OnNoReconnect(onNoReconnect)
	ticker.OnTick(onTick)
	ticker.OnOrderUpdate(onOrderUpdate)
	// Check if current time is between 9 AM and 3:30 PM
	currentTime := time.Now()
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 9, 0, 0, 0, currentTime.Location())
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 30, 0, 0, currentTime.Location())

	if currentTime.After(startTime) && currentTime.Before(endTime) {
		// Start the connection
		go initiateTickerChannellistener()
		ticker.Serve()
	}
	<-tickerchan
}
