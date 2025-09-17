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
	// ticker     *Ticker
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
		// print the dma for testing
		fmt.Println("50 DMA: ", instruments[k][0], dmaValues[instruments[k][0]+"_50DMA"])
		fmt.Println("200 DMA: ", instruments[k][0], dmaValues[instruments[k][0]+"_200DMA"])
		getWeeklyHistory(k)
		findAccumulation1Week(k)
		// fetchDeliveryToTradedQuantity(instruments[k])
	}
	// for k := range top10Volumes {
	// 	fmt.Println(k, top10Volumes[k])
	// 	fmt.Println(k, top10VolumesDates[k])
	// 	fmt.Println("************************")
	// }
	// fmt.Println(incrementForLastThreeDays)
	// fmt.Println(incrementForLastThreeWeeks)
}

func main() {
	testMode := os.Getenv("TEST_MODE")
	SendTelegramMessage("Bot started")
	// Restrict to one stock in test mode
	if testMode == "true" {
		instruments = map[uint32][]string{
			4834049: {"SJVN", "S14"},
		}
	}

	apiKey := "my_api_key"
	accessToken := "my_access_token"

	// Split instrument tokens into two slices
	var tokens1, tokens2 []uint32
	i := 0
	for k := range instruments {
		if i%2 == 0 {
			tokens1 = append(tokens1, k)
		} else {
			tokens2 = append(tokens2, k)
		}
		i++
	}

	// Create two ticker instances
	ticker1 := New(apiKey, accessToken)
	ticker2 := New(apiKey, accessToken)

	// Assign callbacks for ticker1
	ticker1.OnError(onError)
	ticker1.OnClose(onClose)
	ticker1.OnConnect(func() {
		fmt.Println("Connected ticker1")
		err := ticker1.Subscribe(tokens1)
		if err != nil {
			fmt.Println("err: ", err)
		}
		err = ticker1.SetMode(ModeFull, tokens1)
		if err != nil {
			fmt.Println("err: ", err)
		}
	})
	ticker1.OnReconnect(onReconnect)
	ticker1.OnNoReconnect(onNoReconnect)
	ticker1.OnTick(onTick)
	ticker1.OnOrderUpdate(onOrderUpdate)

	// Assign callbacks for ticker2
	ticker2.OnError(onError)
	ticker2.OnClose(onClose)
	ticker2.OnConnect(func() {
		fmt.Println("Connected ticker2")
		err := ticker2.Subscribe(tokens2)
		if err != nil {
			fmt.Println("err: ", err)
		}
		err = ticker2.SetMode(ModeFull, tokens2)
		if err != nil {
			fmt.Println("err: ", err)
		}
	})
	ticker2.OnReconnect(onReconnect)
	ticker2.OnNoReconnect(onNoReconnect)
	ticker2.OnTick(onTick)
	ticker2.OnOrderUpdate(onOrderUpdate)

	go kiteCandleCalls()
	go printAllPriceForecasts()
	getNSECookie()
	cacheCorporateActions()
	fetchBulkDeals()
	initDeliveryTrend()

	currentTime := time.Now()
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 9, 0, 0, 0, currentTime.Location())
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 30, 0, 0, currentTime.Location())

	if currentTime.After(startTime) && currentTime.Before(endTime) {
		go initiateTickerChannellistener()
		go ticker1.Serve()
		go ticker2.Serve()
		// select {} // block forever
	}
	select {}
}
