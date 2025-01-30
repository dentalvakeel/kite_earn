package main

import (
	"fmt"
	"os"

	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
)

func writeToFile(tick kitemodels.Tick) {
	fileName := instruments[tick.InstrumentToken]
	f, err := os.OpenFile("ticks/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if tick.TotalBuyQuantity > tick.TotalSellQuantity {
		f, err := os.OpenFile("Dashboard", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		s := fmt.Sprintf("%s		%f		%f		%d		%d		%d\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity, tick.VolumeTraded)
		_, err = f.WriteString(s)
		if err != nil {
			panic(err)
		}
	}
	s := fmt.Sprintf("%s	%f	%f	%d	%d	%d\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity, tick.VolumeTraded)
	_, err = f.WriteString(s)
	if err != nil {
		panic(err)
	}
}

var DashboardMap = map[string]int{}

func writeGTVolumesToDashboard(tick kitemodels.Tick) {
	lastYearsVolumes := top10Volumes[instruments[tick.InstrumentToken]]
	for index, v := range lastYearsVolumes {
		// fmt.Println(tick.VolumeTraded, instruments[tick.InstrumentToken], v)
		instrumentdayKey := fmt.Sprintf("%d-%d", tick.InstrumentToken, index)
		_, ok := DashboardMap[instrumentdayKey]
		if ok {
			DashboardMap[instrumentdayKey]++
			if DashboardMap[instrumentdayKey] > 3 {
				delete(DashboardMap, instrumentdayKey)
			}
			continue
		}
		if tick.VolumeTraded > v {
			f, err := os.OpenFile("Dashboard", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			DashboardMap[instrumentdayKey] = 1
			s := fmt.Sprintf("%s	%f	%f	%d	%d	%d %d %s\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity, tick.VolumeTraded, index, top10VolumesDates[instruments[tick.InstrumentToken]][index])
			_, err = f.WriteString(s)
			if err != nil {
				panic(err)
			}
		}
	}
}
