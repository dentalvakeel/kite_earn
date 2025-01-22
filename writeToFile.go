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
	s := fmt.Sprintf("%s	%f	%f	%d	%d\n", instruments[tick.InstrumentToken], tick.OHLC.Open, tick.LastPrice, tick.TotalBuyQuantity, tick.TotalSellQuantity)
	_, err = f.WriteString(s)
	if err != nil {
		panic(err)
	}
}
