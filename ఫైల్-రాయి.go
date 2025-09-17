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
	writeInterval = 1 * time.Minute
	DMAwriteTime  = make(map[uint32]time.Time) // Minimum interval between writes for the same script
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
	fileName := instruments[tick.InstrumentToken][0]
	f, err := os.OpenFile("ticks/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	fileName = fmt.Sprintf("Dashboard_%s", time.Now().Format("02-01-2006"))
	dashboardfile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	defer dashboardfile.Close()
	if len(top10Volumes[instruments[tick.InstrumentToken][0]]) == 0 {
		return
	}
	deliverPercent := fetchDeliveryToTradedQuantity(instruments[tick.InstrumentToken][0])
	dashboardPass := deliverPercent > 30 || deliverPercent == 0
	if tick.LastPrice > tick.OHLC.Close && findDepthFavourable(tick) && dashboardPass {
		// fetchHistoricalData(instruments[tick.InstrumentToken])
		w := tabwriter.NewWriter(dashboardfile, 0, 0, 1, ' ', tabwriter.Debug)
		// fmt.Fprintln(w, "Instrument\tOpen\tLast Price\tTotal Buy Quantity\tTotal Sell Quantity\tVolume Traded\tDelivery Trend\tIncrement For Last Three Weeks\tIncrement for Last three days\tBuyQtyTrend\tSellQtyTrend")
		fmt.Fprintf(w, "%s\t%f\t%f\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%f\t%f\t%f\t%f\t\n",
			instruments[tick.InstrumentToken],
			tick.OHLC.Open,
			tick.LastPrice,
			tick.TotalBuyQuantity,
			tick.TotalSellQuantity,
			tick.VolumeTraded,
			top10Volumes[instruments[tick.InstrumentToken][0]][len(top10Volumes[instruments[tick.InstrumentToken][0]])-1],
			DeliveryTrend[instruments[tick.InstrumentToken][0]],
			incrementForLastThreeWeeks[instruments[tick.InstrumentToken][0]],
			incrementForLastThreeDays[instruments[tick.InstrumentToken][0]],
			buyDepthTrend[tick.InstrumentToken],
			sellDepthTrend[tick.InstrumentToken],
			accumilation1Week[tick.InstrumentToken],
			deliverPercent,
			dmaValues[instruments[tick.InstrumentToken][0]+"_50DMA"],
			dmaValues[instruments[tick.InstrumentToken][0]+"_200DMA"],
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
	lastYearsVolumes := top10Volumes[instruments[tick.InstrumentToken][0]]

	isCurrentPricenear50DMA := tick.LastPrice > dmaValues[instruments[tick.InstrumentToken][0]+"_50DMA"]*0.97 && tick.LastPrice < dmaValues[instruments[tick.InstrumentToken][0]+"_50DMA"]*1.03
	isCurrentPricenear200DMA := tick.LastPrice > dmaValues[instruments[tick.InstrumentToken][0]+"_200DMA"]*0.97 && tick.LastPrice < dmaValues[instruments[tick.InstrumentToken][0]+"_200DMA"]*1.03
	isInTop40Volumes := false
	for _, vol := range top40Volumes[instruments[tick.InstrumentToken][0]] {
		if tick.VolumeTraded > vol {
			isInTop40Volumes = true
			break
		}
	}
	deliverPercent := fetchDeliveryToTradedQuantity(instruments[tick.InstrumentToken][0])
	dashboardPass := deliverPercent > 30 || deliverPercent == 0
	// if tick.currentPrice in range of top40volume
	if isInTop40Volumes && (isCurrentPricenear200DMA || isCurrentPricenear50DMA) && dashboardPass {
		// Write to dashboard
		f, err := os.OpenFile("Dashboard", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		// use DMAticker map to store time when last written to Dashboard file if time is < 3 mins avoid writing
		// Write to dashboard
		lastDMAwrite, ok := DMAwriteTime[tick.InstrumentToken]
		if ok && time.Since(lastDMAwrite) < 3*time.Minute {
			return
		}
		s := fmt.Sprintf("%s %s	%f	%f	%d %d %d %f %f\n",
			"Range",
			instruments[tick.InstrumentToken],
			tick.OHLC.Open,
			tick.LastPrice,
			tick.TotalBuyQuantity,
			tick.TotalSellQuantity,
			tick.VolumeTraded,
			dmaValues[instruments[tick.InstrumentToken][0]+"_50DMA"],
			dmaValues[instruments[tick.InstrumentToken][0]+"_200DMA"])
		_, err = f.WriteString(s)
		SendTelegramMessage(s)
		DMAwriteTime[tick.InstrumentToken] = time.Now()
		if err != nil {
			panic(err)
		}
		// continue
	}

	rank := 21
	// iterate range in reverse order to find the index
	for _, v := range lastYearsVolumes {
		if tick.VolumeTraded > v {
			rank--
		}
	}

	// fmt.Println(tick.VolumeTraded, instruments[tick.InstrumentToken], v)

	instrumentdayKey := fmt.Sprintf("%d", tick.InstrumentToken)
	_, ok := DashboardMap[instrumentdayKey]
	if ok {
		DashboardMap[instrumentdayKey]++
		if DashboardMap[instrumentdayKey] > 1000 {
			delete(DashboardMap, instrumentdayKey)
		}
		return
	} else {
		DashboardMap[instrumentdayKey] = 1
	}

	if rank < 21 && tick.LastPrice > tick.OHLC.Close {
		f, err := os.OpenFile("Dashboard", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		s := fmt.Sprintf("%s	%f	%f	%d	%d	%d %d %s\n",
			instruments[tick.InstrumentToken],
			tick.OHLC.Open,
			tick.LastPrice,
			tick.TotalBuyQuantity,
			tick.TotalSellQuantity,
			tick.VolumeTraded,
			rank,
			top10VolumesDates[instruments[tick.InstrumentToken][0]][rank-1])
		_, err = f.WriteString(s)
		SendTelegramMessage(s)
		if err != nil {
			panic(err)
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
