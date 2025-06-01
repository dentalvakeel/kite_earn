package main

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
)

var (
	lastWriteTime = make(map[uint32]time.Time) // Tracks last write time for each instrument token
	writeMutex    sync.Mutex                   // Ensures thread-safe access to lastWriteTime
	writeInterval = 1 * time.Minute            // Minimum interval between writes for the same script
)

func writeToFile(tick kitemodels.Tick) {
	if readWrittenInst(tick.InstrumentToken) {
		return
	}

	writeMutex.Lock()
	lastWrite, exists := lastWriteTime[tick.InstrumentToken]
	if exists && time.Since(lastWrite) < writeInterval {
		writeMutex.Unlock()
		return // Skip writing if the interval has not passed
	}
	defer writeMutex.Unlock()

	findDepthUptrend(tick)
	fileName := instruments[tick.InstrumentToken]
	f, err := os.OpenFile("ticks/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	fileName = fmt.Sprintf("Dashboard_%s", time.Now().Format("02-01-2006"))
	dashboardfile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	defer f.Close()
	defer dashboardfile.Close()
	if len(top10Volumes[instruments[tick.InstrumentToken]]) == 0 {
		return
	}
	deliverPercent := fetchDeliveryToTradedQuantity(instruments[tick.InstrumentToken])
	dashboardPass := deliverPercent > 30 || deliverPercent == 0
	if tick.LastPrice > tick.OHLC.Close && findDepthFavourable(tick) && dashboardPass {
		// fetchHistoricalData(instruments[tick.InstrumentToken])
		w := tabwriter.NewWriter(dashboardfile, 0, 0, 1, ' ', tabwriter.Debug)
		// fmt.Fprintln(w, "Instrument\tOpen\tLast Price\tTotal Buy Quantity\tTotal Sell Quantity\tVolume Traded\tDelivery Trend\tIncrement For Last Three Weeks\tIncrement for Last three days\tBuyQtyTrend\tSellQtyTrend")
		fmt.Fprintf(w, "%s\t%f\t%f\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%d\t%f\t\n",
			instruments[tick.InstrumentToken],
			tick.OHLC.Open,
			tick.LastPrice,
			tick.TotalBuyQuantity,
			tick.TotalSellQuantity,
			tick.VolumeTraded,
			DeliveryTrend[instruments[tick.InstrumentToken]],
			incrementForLastThreeWeeks[instruments[tick.InstrumentToken]],
			incrementForLastThreeDays[instruments[tick.InstrumentToken]],
			buyDepthTrend[tick.InstrumentToken],
			sellDepthTrend[tick.InstrumentToken],
			top10Volumes[instruments[tick.InstrumentToken]][len(top10Volumes[instruments[tick.InstrumentToken]])-1],
			accumilation1Week[tick.InstrumentToken],
		)
		w.Flush()
		lastWriteTime[tick.InstrumentToken] = time.Now()
	}

	s := fmt.Sprintf("%s	%f	%f	%d	%d	%d\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity, tick.VolumeTraded)
	_, err = f.WriteString(s)
	if err != nil {
		panic(err)
	}
}

var DashboardMap = map[string]int{}

func writeGTVolumesToDashboard(tick kitemodels.Tick) {
	if readWrittenInst(tick.InstrumentToken) {
		return
	}
	// updateWrittenInst(tick.InstrumentToken, true)
	lastYearsVolumes := top10Volumes[instruments[tick.InstrumentToken]]
	for index, v := range lastYearsVolumes {
		// fmt.Println(tick.VolumeTraded, instruments[tick.InstrumentToken], v)

		instrumentdayKey := fmt.Sprintf("%d-%d", tick.InstrumentToken, index)
		_, ok := DashboardMap[instrumentdayKey]
		if ok {
			DashboardMap[instrumentdayKey]++
			if DashboardMap[instrumentdayKey] > 1000 {
				delete(DashboardMap, instrumentdayKey)
			}
			continue
		} else {
			DashboardMap[instrumentdayKey] = 1
		}
		if tick.VolumeTraded > v && tick.LastPrice > tick.OHLC.Close {
			f, err := os.OpenFile("Dashboard", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			s := fmt.Sprintf("%s	%f	%f	%d	%d	%d %d %s\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity, tick.VolumeTraded, index, top10VolumesDates[instruments[tick.InstrumentToken]][index])
			_, err = f.WriteString(s)
			if err != nil {
				panic(err)
			}
		}
	}
}

func findDepthFavourable(tick kitemodels.Tick) bool {
	buyDepth := tick.Depth.Buy
	sellDepth := tick.Depth.Sell
	var totalBuyDepth float32
	var totalSellDepth float32
	for _, v := range buyDepth {
		totalBuyDepth += float32(v.Quantity) * float32(v.Orders) * float32(v.Price)
	}

	for _, v := range sellDepth {
		totalSellDepth += float32(v.Quantity) * float32(v.Orders) * float32(v.Price)
	}

	return totalBuyDepth > totalSellDepth
}

var buyDepthQuantityBuffer = make(map[uint32][]uint32)
var sellDepthQuantityBuffer = make(map[uint32][]uint32)
var buyDepthTrend = make(map[uint32]Trend)
var sellDepthTrend = make(map[uint32]Trend)

func findDepthUptrend(tick kitemodels.Tick) {
	buyDepth := tick.TotalBuyQuantity
	sellDepth := tick.TotalSellQuantity
	buyDepthQuantityBuffer[tick.InstrumentToken] = append(buyDepthQuantityBuffer[tick.InstrumentToken], buyDepth)
	sellDepthQuantityBuffer[tick.InstrumentToken] = append(sellDepthQuantityBuffer[tick.InstrumentToken], sellDepth)

	// Check if the buy depth is increasing for the last 100 ticks
	if len(buyDepthQuantityBuffer[tick.InstrumentToken]) > 50 {
		buyDepthQuantityBuffer[tick.InstrumentToken] = buyDepthQuantityBuffer[tick.InstrumentToken][:50]
		// for i := 1; i < len(buyDepthQuantityBuffer[tick.InstrumentToken]); i++ {
		// 	if buyDepthQuantityBuffer[tick.InstrumentToken][i] < buyDepthQuantityBuffer[tick.InstrumentToken][i-1] {
		// 		buyDepthTrend[tick.InstrumentToken] = DownTrend
		// 		break
		// 	} else {
		// 		buyDepthTrend[tick.InstrumentToken] = UpTrend
		// 	}
		// }
		x := make([]float64, len(buyDepthQuantityBuffer[tick.InstrumentToken]))
		y := make([]float64, len(x))
		for i := range x {
			x[i] = float64(i)
			y[i] = float64(buyDepthQuantityBuffer[tick.InstrumentToken][i])
		}
		// iterate over y, subtract prev element and do sum of delta
		// if sum is positive, then it is uptrend
		// if sum is negative, then it is downtrend
		sumy := 0.0
		for i := range y {
			if i == 0 {
				continue
			}
			y[i] = y[i] - y[i-1]
			sumy += y[i]
		}
		// slope, _ := stat.LinearRegression(x, y, nil, false)

		// Determine the trend based on the slope
		if sumy > 0 {
			buyDepthTrend[tick.InstrumentToken] = UpTrend
		} else {
			buyDepthTrend[tick.InstrumentToken] = DownTrend
		}
	} else {
		buyDepthTrend[tick.InstrumentToken] = FlatTrend
	}

	// check if the sell depth is decreasing for the last 100 ticks
	if len(sellDepthQuantityBuffer[tick.InstrumentToken]) > 50 {
		sellDepthQuantityBuffer[tick.InstrumentToken] = sellDepthQuantityBuffer[tick.InstrumentToken][:50]
		// for i := 1; i < len(sellDepthQuantityBuffer[tick.InstrumentToken]); i++ {
		// 	if sellDepthQuantityBuffer[tick.InstrumentToken][i] > sellDepthQuantityBuffer[tick.InstrumentToken][i-1] {
		// 		sellDepthTrend[tick.InstrumentToken] = DownTrend
		// 		break
		// 	} else {
		// 		sellDepthTrend[tick.InstrumentToken] = UpTrend
		// 	}
		// }

		x := make([]float64, len(sellDepthQuantityBuffer[tick.InstrumentToken]))
		y := make([]float64, len(x))
		for i := range x {
			x[i] = float64(i)
			y[i] = float64(sellDepthQuantityBuffer[tick.InstrumentToken][i])
		}

		sumy := 0.0
		for i := range y {
			if i == 0 {
				continue
			}
			y[i] = y[i] - y[i-1]
			sumy += y[i]
		}
		// slope, _ := stat.LinearRegression(x, y, nil, false)

		// Determine the trend based on the slope
		if sumy < 0 {
			sellDepthTrend[tick.InstrumentToken] = UpTrend
		} else {
			sellDepthTrend[tick.InstrumentToken] = DownTrend
		}
	} else {
		sellDepthTrend[tick.InstrumentToken] = FlatTrend
	}
}
