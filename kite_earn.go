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
// Package kiteticker provides kite ticker access using callbacks.
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/zerodha/gokiteconnect/v4/models"
)

// Mode represents available ticker modes.
type Mode string

// Ticker is a Kite connect ticker instance.
type Ticker struct {
	Conn *websocket.Conn

	apiKey      string
	accessToken string

	url                 url.URL
	callbacks           callbacks
	lastPingTime        atomicTime
	autoReconnect       bool
	reconnectMaxRetries int
	reconnectMaxDelay   time.Duration
	connectTimeout      time.Duration

	reconnectAttempt int

	subscribedTokens map[uint32]Mode

	cancel context.CancelFunc
}

// atomicTime is wrapper over time.Time to safely access
// an updating timestamp concurrently.
type atomicTime struct {
	v atomic.Value
}

// Get returns the current timestamp.
func (b *atomicTime) Get() time.Time {
	return b.v.Load().(time.Time)
}

// Set sets the current timestamp.
func (b *atomicTime) Set(value time.Time) {
	b.v.Store(value)
}

// callbacks represents callbacks available in ticker.
type callbacks struct {
	onTick        func(models.Tick)
	onMessage     func(int, []byte)
	onNoReconnect func(int)
	onReconnect   func(int, time.Duration)
	onConnect     func()
	onClose       func(int, string)
	onError       func(error)
	onOrderUpdate func(kiteconnect.Order)
}

type tickerInput struct {
	Type string      `json:"a"`
	Val  interface{} `json:"v"`
}

type message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

const (
	// Segment constants.
	NseCM = 1 + iota
	NseFO
	NseCD
	BseCM
	BseFO
	BseCD
	McxFO
	McxSX
	Indices

	// ModeLTP subscribes for last price.
	ModeLTP Mode = "ltp"
	// ModeFull subscribes for all the available fields.
	ModeFull Mode = "full"
	// ModeQuote represents quote mode.
	ModeQuote Mode = "quote"

	// Mode empty is used internally for storing tokens which doesn't have any modes
	modeEmpty Mode = "empty"

	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10

	// packet length for each mode.
	modeLTPLength              = 8
	modeQuoteIndexPacketLength = 28
	modeFullIndexLength        = 32
	modeQuoteLength            = 44
	modeFullLength             = 184

	// Message types
	messageError = "error"
	messageOrder = "order"

	// Auto reconnect defaults
	// Default maximum number of reconnect attempts
	defaultReconnectMaxAttempts = 300
	// Auto reconnect min delay. Reconnect delay can't be less than this.
	reconnectMinDelay time.Duration = 5000 * time.Millisecond
	// Default auto reconnect delay to be used for auto reconnection.
	defaultReconnectMaxDelay time.Duration = 60000 * time.Millisecond
	// Connect timeout for initial server handshake.
	defaultConnectTimeout time.Duration = 7000 * time.Millisecond
	// Interval in which the connection check is performed periodically.
	connectionCheckInterval time.Duration = 2000 * time.Millisecond
	// Interval which is used to determine if the connection is still active. If last ping time exceeds this then
	// connection is considered as dead and reconnection is initiated.
	dataTimeoutInterval time.Duration = 5000 * time.Millisecond
)

var (
	// Default ticker url.
	tickerURL = url.URL{Scheme: "wss", Host: "ws.kite.trade"}
)

// New creates a new ticker instance.
func New(apiKey string, accessToken string) *Ticker {
	ticker := &Ticker{
		apiKey:              apiKey,
		accessToken:         accessToken,
		url:                 tickerURL,
		autoReconnect:       true,
		reconnectMaxDelay:   defaultReconnectMaxDelay,
		reconnectMaxRetries: defaultReconnectMaxAttempts,
		connectTimeout:      defaultConnectTimeout,
		subscribedTokens:    map[uint32]Mode{},
	}

	return ticker
}

// SetRootURL sets ticker root url.
func (t *Ticker) SetRootURL(u url.URL) {
	t.url = u
}

// SetAccessToken set access token.
func (t *Ticker) SetAccessToken(aToken string) {
	t.accessToken = aToken
}

// SetConnectTimeout sets default timeout for initial connect handshake
func (t *Ticker) SetConnectTimeout(val time.Duration) {
	t.connectTimeout = val
}

// SetAutoReconnect enable/disable auto reconnect.
func (t *Ticker) SetAutoReconnect(val bool) {
	t.autoReconnect = val
}

// SetReconnectMaxDelay sets maximum auto reconnect delay.
func (t *Ticker) SetReconnectMaxDelay(val time.Duration) error {
	if val > reconnectMinDelay {
		return fmt.Errorf("ReconnectMaxDelay can't be less than %fms", reconnectMinDelay.Seconds()*1000)
	}

	t.reconnectMaxDelay = val
	return nil
}

// SetReconnectMaxRetries sets maximum reconnect attempts.
func (t *Ticker) SetReconnectMaxRetries(val int) {
	t.reconnectMaxRetries = val
}

// OnConnect callback.
func (t *Ticker) OnConnect(f func()) {
	t.callbacks.onConnect = f
}

// OnError callback.
func (t *Ticker) OnError(f func(err error)) {
	t.callbacks.onError = f
}

// OnClose callback.
func (t *Ticker) OnClose(f func(code int, reason string)) {
	t.callbacks.onClose = f
}

// OnMessage callback.
func (t *Ticker) OnMessage(f func(messageType int, message []byte)) {
	t.callbacks.onMessage = f
}

// OnReconnect callback.
func (t *Ticker) OnReconnect(f func(attempt int, delay time.Duration)) {
	t.callbacks.onReconnect = f
}

// OnNoReconnect callback.
func (t *Ticker) OnNoReconnect(f func(attempt int)) {
	t.callbacks.onNoReconnect = f
}

// OnTick callback.
func (t *Ticker) OnTick(f func(tick models.Tick)) {
	t.callbacks.onTick = f
}

// OnOrderUpdate callback.
func (t *Ticker) OnOrderUpdate(f func(order kiteconnect.Order)) {
	t.callbacks.onOrderUpdate = f
}

// Serve starts the connection to ticker server. Since its blocking its
// recommended to use it in a go routine.
func (t *Ticker) Serve() {
	t.ServeWithContext(context.Background())
}

// ServeWithContext starts the connection to ticker server and additionally
// accepts a context. Since its blocking its recommended to use it in a go
// routine.
func (t *Ticker) ServeWithContext(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// If reconnect attempt exceeds max then close the loop
			if t.reconnectAttempt > t.reconnectMaxRetries {
				t.triggerNoReconnect(t.reconnectAttempt)
				return
			}

			// If its a reconnect then wait exponentially based on reconnect attempt
			if t.reconnectAttempt > 0 {
				nextDelay := time.Duration(math.Pow(2, float64(t.reconnectAttempt))) * time.Second
				if nextDelay > t.reconnectMaxDelay || nextDelay <= 0 {
					nextDelay = t.reconnectMaxDelay
				}

				t.triggerReconnect(t.reconnectAttempt, nextDelay)

				time.Sleep(nextDelay)

				// Close the previous connection if exists
				if t.Conn != nil {
					t.Conn.Close()
				}
			}

			// Prepare ticker URL with required params.
			q := t.url.Query()
			t.url.Host = "ws.zerodha.com"
			q.Set("v", "3.0.0")
			q.Set("user-agent", "kite3-web")
			q.Set("uid", "1738210133447")
			q.Set("user_id", "YA0828")
			q.Set("api_key", "kitefront")
			q.Set("enctoken", os.Getenv("enctoken"))
			t.url.RawQuery = q.Encode()

			// create a dialer
			d := websocket.DefaultDialer
			d.HandshakeTimeout = t.connectTimeout
			fmt.Println("Dialing: ", t.url.String())
			conn, _, err := d.Dial(t.url.String(), nil)
			if err != nil {
				t.triggerError(err)

				// If auto reconnect is enabled then try reconneting else return error
				if t.autoReconnect {
					t.reconnectAttempt++
					continue
				}
			}

			// Close the connection when its done.
			defer func() {
				if t.Conn != nil {
					t.Conn.Close()
				}
			}()

			// Assign the current connection to the instance.
			t.Conn = conn

			// Trigger connect callback.
			t.triggerConnect()

			// Resubscribe to stored tokens
			if t.reconnectAttempt > 0 {
				t.Resubscribe()
			}

			// Reset auto reconnect vars
			t.reconnectAttempt = 0

			// Set current time as last ping time
			t.lastPingTime.Set(time.Now())

			// Set on close handler
			t.Conn.SetCloseHandler(t.handleClose)

			var wg sync.WaitGroup

			// Receive ticker data in a go routine.
			wg.Add(1)
			go t.readMessage(ctx, &wg)

			// Run watcher to check last ping time and reconnect if required
			if t.autoReconnect {
				wg.Add(1)
				go t.checkConnection(ctx, &wg)
			}

			// Wait for go routines to finish before doing next reconnect
			wg.Wait()
		}
	}
}

func (t *Ticker) handleClose(code int, reason string) error {
	t.triggerClose(code, reason)
	return nil
}

// Trigger callback methods
func (t *Ticker) triggerError(err error) {
	if t.callbacks.onError != nil {
		t.callbacks.onError(err)
	}
}

func (t *Ticker) triggerClose(code int, reason string) {
	if t.callbacks.onClose != nil {
		t.callbacks.onClose(code, reason)
	}
}

func (t *Ticker) triggerConnect() {
	if t.callbacks.onConnect != nil {
		t.callbacks.onConnect()
	}
}

func (t *Ticker) triggerReconnect(attempt int, delay time.Duration) {
	if t.callbacks.onReconnect != nil {
		t.callbacks.onReconnect(attempt, delay)
	}
}

func (t *Ticker) triggerNoReconnect(attempt int) {
	if t.callbacks.onNoReconnect != nil {
		t.callbacks.onNoReconnect(attempt)
	}
}

func (t *Ticker) triggerMessage(messageType int, message []byte) {
	if t.callbacks.onMessage != nil {
		t.callbacks.onMessage(messageType, message)
	}
}

func (t *Ticker) triggerTick(tick models.Tick) {
	if t.callbacks.onTick != nil {
		t.callbacks.onTick(tick)
	}
}

func (t *Ticker) triggerOrderUpdate(order kiteconnect.Order) {
	if t.callbacks.onOrderUpdate != nil {
		t.callbacks.onOrderUpdate(order)
	}
}

// Periodically check for last ping time and initiate reconnect if applicable.
func (t *Ticker) checkConnection(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Sleep before doing next check
			time.Sleep(connectionCheckInterval)

			// If last ping time is greater then timeout interval then close the
			// existing connection and reconnect
			if time.Since(t.lastPingTime.Get()) > dataTimeoutInterval {
				// Close the current connection without waiting for close frame
				if t.Conn != nil {
					t.Conn.Close()
				}

				// Increase reconnect attempt for next reconnection
				t.reconnectAttempt++
				// Mark it as done in wait group
				return
			}
		}
	}
}

// readMessage reads the data in a loop.
func (t *Ticker) readMessage(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mType, msg, err := t.Conn.ReadMessage()
			if err != nil {
				t.triggerError(fmt.Errorf("error reading data: %v", err))
				return
			}

			// Update last ping time to check for connection
			t.lastPingTime.Set(time.Now())

			// Trigger message.
			t.triggerMessage(mType, msg)

			// If binary message then parse and send tick.
			if mType == websocket.BinaryMessage {
				ticks, err := t.parseBinary(msg)
				if err != nil {
					t.triggerError(fmt.Errorf("error parsing data received: %v", err))
				}

				// Trigger individual tick.
				for _, tick := range ticks {
					t.triggerTick(tick)
				}
			} else if mType == websocket.TextMessage {
				t.processTextMessage(msg)
			}
		}
	}
}

// Close tries to close the connection gracefully. If the server doesn't close it
func (t *Ticker) Close() error {
	return t.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// Stop the ticker instance and all the goroutines it has spawned.
func (t *Ticker) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

// Subscribe subscribes tick for the given list of tokens.
func (t *Ticker) Subscribe(tokens []uint32) error {
	if len(tokens) == 0 {
		return nil
	}

	out, err := json.Marshal(tickerInput{
		Type: "subscribe",
		Val:  tokens,
	})
	if err != nil {
		return err
	}

	// Store tokens to current subscriptions
	for _, ts := range tokens {
		t.subscribedTokens[ts] = modeEmpty
	}

	return t.Conn.WriteMessage(websocket.TextMessage, out)
}

// Unsubscribe unsubscribes tick for the given list of tokens.
func (t *Ticker) Unsubscribe(tokens []uint32) error {
	if len(tokens) == 0 {
		return nil
	}

	out, err := json.Marshal(tickerInput{
		Type: "unsubscribe",
		Val:  tokens,
	})
	if err != nil {
		return err
	}

	// Remove tokens from current subscriptions
	for _, ts := range tokens {
		delete(t.subscribedTokens, ts)
	}

	return t.Conn.WriteMessage(websocket.TextMessage, out)
}

// SetMode changes mode for given list of tokens and mode.
func (t *Ticker) SetMode(mode Mode, tokens []uint32) error {
	if len(tokens) == 0 {
		return nil
	}

	out, err := json.Marshal(tickerInput{
		Type: "mode",
		Val:  []interface{}{mode, tokens},
	})
	if err != nil {
		return err
	}

	// Set mode in current subscriptions stored
	for _, ts := range tokens {
		t.subscribedTokens[ts] = mode
	}

	return t.Conn.WriteMessage(websocket.TextMessage, out)
}

// Resubscribe resubscribes to the current stored subscriptions
func (t *Ticker) Resubscribe() error {
	var tokens []uint32
	modes := map[Mode][]uint32{
		ModeFull:  []uint32{},
		ModeQuote: []uint32{},
		ModeLTP:   []uint32{},
	}

	// Make a map of mode and corresponding tokens
	for to, mo := range t.subscribedTokens {
		tokens = append(tokens, to)
		if mo != modeEmpty {
			modes[mo] = append(modes[mo], to)
		}
	}

	fmt.Println("Subscribe again: ", tokens, t.subscribedTokens)

	// Subscribe to tokens
	if len(tokens) > 0 {
		if err := t.Subscribe(tokens); err != nil {
			return err
		}
	}

	// Set mode to tokens
	for mo, tos := range modes {
		if len(tos) > 0 {
			if err := t.SetMode(mo, tos); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Ticker) processTextMessage(inp []byte) {
	var msg message
	if err := json.Unmarshal(inp, &msg); err != nil {
		// May be error should be triggered
		return
	}

	if msg.Type == messageError {
		// Trigger text error
		t.triggerError(fmt.Errorf(msg.Data.(string)))
	} else if msg.Type == messageOrder {
		// Parse order update data
		order := struct {
			Data kiteconnect.Order `json:"data"`
		}{}

		if err := json.Unmarshal(inp, &order); err != nil {
			// May be error should be triggered
			return
		}

		t.triggerOrderUpdate(order.Data)
	}
}

// parseBinary parses the packets to ticks.
func (t *Ticker) parseBinary(inp []byte) ([]models.Tick, error) {
	pkts := t.splitPackets(inp)
	var ticks []models.Tick

	for _, pkt := range pkts {
		tick, err := parsePacket(pkt)
		if err != nil {
			return nil, err
		}

		ticks = append(ticks, tick)
	}

	return ticks, nil
}

// splitPackets splits packet dump to individual tick packet.
func (t *Ticker) splitPackets(inp []byte) [][]byte {
	var pkts [][]byte
	if len(inp) < 2 {
		return pkts
	}

	pktLen := binary.BigEndian.Uint16(inp[0:2])

	j := 2
	for i := 0; i < int(pktLen); i++ {
		pLen := binary.BigEndian.Uint16(inp[j : j+2])
		pkts = append(pkts, inp[j+2:j+2+int(pLen)])
		j = j + 2 + int(pLen)
	}

	return pkts
}

// Parse parses a tick byte array into a tick struct.
func parsePacket(b []byte) (models.Tick, error) {
	var (
		tk         = binary.BigEndian.Uint32(b[0:4])
		seg        = tk & 0xFF
		isIndex    = seg == Indices
		isTradable = seg != Indices
	)

	// Mode LTP parsing
	if len(b) == modeLTPLength {
		return models.Tick{
			Mode:            string(ModeLTP),
			InstrumentToken: tk,
			IsTradable:      isTradable,
			IsIndex:         isIndex,
			LastPrice:       convertPrice(seg, float64(binary.BigEndian.Uint32(b[4:8]))),
		}, nil
	}

	// Parse index mode full and mode quote data
	if len(b) == modeQuoteIndexPacketLength || len(b) == modeFullIndexLength {
		var (
			lastPrice  = convertPrice(seg, float64(binary.BigEndian.Uint32(b[4:8])))
			closePrice = convertPrice(seg, float64(binary.BigEndian.Uint32(b[20:24])))
		)

		tick := models.Tick{
			Mode:            string(ModeQuote),
			InstrumentToken: tk,
			IsTradable:      isTradable,
			IsIndex:         isIndex,
			LastPrice:       lastPrice,
			NetChange:       lastPrice - closePrice,
			OHLC: models.OHLC{
				High:  convertPrice(seg, float64(binary.BigEndian.Uint32(b[8:12]))),
				Low:   convertPrice(seg, float64(binary.BigEndian.Uint32(b[12:16]))),
				Open:  convertPrice(seg, float64(binary.BigEndian.Uint32(b[16:20]))),
				Close: closePrice,
			}}

		// On mode full set timestamp
		if len(b) == modeFullIndexLength {
			tick.Mode = string(ModeFull)
			tick.Timestamp = models.Time{time.Unix(int64(binary.BigEndian.Uint32(b[28:32])), 0)}
		}

		return tick, nil
	}

	// Parse mode quote.
	var (
		lastPrice  = convertPrice(seg, float64(binary.BigEndian.Uint32(b[4:8])))
		closePrice = convertPrice(seg, float64(binary.BigEndian.Uint32(b[40:44])))
	)

	// Mode quote data.
	tick := models.Tick{
		Mode:               string(ModeQuote),
		InstrumentToken:    tk,
		IsTradable:         isTradable,
		IsIndex:            isIndex,
		LastPrice:          lastPrice,
		LastTradedQuantity: binary.BigEndian.Uint32(b[8:12]),
		AverageTradePrice:  convertPrice(seg, float64(binary.BigEndian.Uint32(b[12:16]))),
		VolumeTraded:       binary.BigEndian.Uint32(b[16:20]),
		TotalBuyQuantity:   binary.BigEndian.Uint32(b[20:24]),
		TotalSellQuantity:  binary.BigEndian.Uint32(b[24:28]),
		OHLC: models.OHLC{
			Open:  convertPrice(seg, float64(binary.BigEndian.Uint32(b[28:32]))),
			High:  convertPrice(seg, float64(binary.BigEndian.Uint32(b[32:36]))),
			Low:   convertPrice(seg, float64(binary.BigEndian.Uint32(b[36:40]))),
			Close: closePrice,
		},
	}

	// Parse full mode.
	if len(b) == modeFullLength {
		tick.Mode = string(ModeFull)
		tick.LastTradeTime = models.Time{time.Unix(int64(binary.BigEndian.Uint32(b[44:48])), 0)}
		tick.OI = binary.BigEndian.Uint32(b[48:52])
		tick.OIDayHigh = binary.BigEndian.Uint32(b[52:56])
		tick.OIDayLow = binary.BigEndian.Uint32(b[56:60])
		tick.Timestamp = models.Time{time.Unix(int64(binary.BigEndian.Uint32(b[60:64])), 0)}
		tick.NetChange = lastPrice - closePrice

		// Depth Information.
		var (
			buyPos     = 64
			sellPos    = 124
			depthItems = (sellPos - buyPos) / 12
		)

		for i := 0; i < depthItems; i++ {
			tick.Depth.Buy[i] = models.DepthItem{
				Quantity: binary.BigEndian.Uint32(b[buyPos : buyPos+4]),
				Price:    convertPrice(seg, float64(binary.BigEndian.Uint32(b[buyPos+4:buyPos+8]))),
				Orders:   uint32(binary.BigEndian.Uint16(b[buyPos+8 : buyPos+10])),
			}

			tick.Depth.Sell[i] = models.DepthItem{
				Quantity: binary.BigEndian.Uint32(b[sellPos : sellPos+4]),
				Price:    convertPrice(seg, float64(binary.BigEndian.Uint32(b[sellPos+4:sellPos+8]))),
				Orders:   uint32(binary.BigEndian.Uint16(b[sellPos+8 : sellPos+10])),
			}

			buyPos += 12
			sellPos += 12
		}
	}

	return tick, nil
}

// convertPrice converts prices of stocks from paise to rupees
// with varying decimals based on the segment.
func convertPrice(seg uint32, val float64) float64 {
	switch seg {
	case NseCD:
		return val / 10000000.0
	case BseCD:
		return val / 10000.0
	default:
		return val / 100.0
	}
}
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
package main

var instruments = map[uint32][]string{
	415745:    {"IOC", "IOC"},
	589569:    {"HAL", "HAL"},
	633601:    {"ONGC", "ONG"},
	1522689:   {"SOUTHBANK", "SIB"},
	2493697:   {"PARADEEP"},
	3834113:   {"POWERGRID", "PGC"},
	3884545:   {"TARIL", "TRI"},
	4834049:   {"SJVN", "S14"},
	5331201:   {"HUDCO", "HUD"},
	6322433:   {"SSEGL"},
	6564609:   {"KRN"},
	7350785:   {"TRANSRAILL", "TLL01"},
	4676609:   {"AEROFLEX", "AI77"},
	3095553:   {"KAYNES", "KTI"},
	779521:    {"SBIN", "SBI"},
	877057:    {"TATAPOWER", "TPC"},
	1215745:   {"CONCOR", "CCI"},
	1517057:   {"INTELLECT", "IDA"},
	5168129:   {"WABAG", "VTW"},
	5461505:   {"JYOTICNC", "JCA"},
	3826433:   {"MOTILALOFS", "MOF01"},
	2905857:   {"PETRONET", "PLN"},
	3764993:   {"TIMETECHNO", "TT08"},
	7264769:   {"IGIL", "IGIIL"},
	3575297:   {"HBLENGINE", "SNP"},
	194561:    {"CGPOWER", "CG"},
	98049:     {"BEL", "BE03"},
	6637313:   {"DANISHPOWER"},
	5186817:   {"IREDA", "IREDAL"},
	3660545:   {"PFC", "PFC02"},
	2730497:   {"PNB"},
	341505:    {"MARINE", "ME21"},
	3670785:   {"GANESHHOUSING"},
	177665:    {"CIPLA"},
	225537:    {"DRREDDY"},
	237569:    {"ELECTCAST", "EC02"},
	692481:    {"PRAJIND", "PI17"},
	6491649:   {"PGEL", "PE06"},
	3353857:   {"VIMTALABS", "VL04"},
	5409537:   {"TEJASNET", "TN"},
	1649921:   {"TECHNOE", "TEE"},
	3026177:   {"WELCORP", "WGS"},
	1147137:   {"AARTIDRUGS"},
	135964676: {"BALUFORGE"},
	2763265:   {"CANBK"},
	4923905:   {"LAURUSLABS"},
	951809:    {"VOLTAS", "V"},
	1007617:   {"HINDZINC"},
	837889:    {"SRF"},
	3536897:   {"LTFOODS", "LTO"},
	3650561:   {"PGIL"},
	6445569:   {"TDPOWERSYS", "TPS"},
	3031041:   {"NH", "NH"},
	361473:    {"AGI"},
	548865:    {"BDL", "BDL01"},
	4840449:   {"PNBHOUSING"},
	7398145:   {"KITEX", "KG01"},
	3832833:   {"KSCL"},
	6194177:   {"VADILALIND"},
	3550209:   {"AARTIPHARM"},
	1234567:   {"AMBUJACEM"},
	2345678:   {"AWHCL"},
	3456789:   {"APOLLOTYRE"},
	6599681:   {"APLAPOLLO"},
	5678901:   {"AMIORG"},
	6789012:   {"AXISBANK"},
	2912513:   {"MAHABANK"},
	8901234:   {"BBOX"},
	9012345:   {"BLSE"},
	101121:    {"BEML", "BEM03"},
	9234567:   {"GAIL"},
	3047681:   {"GENUSPOWER", "GOE"},
	9456789:   {"GLS"},
	9567890:   {"GOKEX"},
	5256705:   {"GRAVITA", "GI24"},
	9789012:   {"HCC"},
	9890123:   {"HCLTECH"},
	428033:    {"HGINFRA"},
	348929:    {"HINDALCO", "H"},
	9923456:   {"BLUESTARCO"},
	1946369:   {"CMSINFO"},
	9945678:   {"CAPLIPOINT"},
	5506049:   {"COCHINSHIP"},
	152321:    {"CARBORUNIV"},
	9978901:   {"CERA"},
	9989012:   {"CONCORDBIO"},
	211713:    {"DEEPAKFERT"},
	10001234:  {"DEEPAKNTR"},
	10012345:  {"DHANUKA"},
	4569857:   {"DLINKINDIA", "DLI01"},
	10034567:  {"IDBI", "IDB05"},
	1216257:   {"INDRAMEDCO"},
	10056789:  {"INDUSTOWER"},
	10067890:  {"ITC"},
	417281:    {"IONEXCHANG"},
	2745857:   {"INDIAMART"},
	10090123:  {"INFY"},
	10101234:  {"ICICIBANK"},
	10112345:  {"JIOFIN"},
	10123456:  {"PATELENG"},
	2962177:   {"PPLPHARMA"},
	2402561:   {"PNCINFRA", "PI29"},
	5870081:   {"PLATIND"},
	10167890:  {"WAAREE"},
	2880769:   {"WELSPUNLIV", "WI03"},
	10189012:  {"WINDLAS", "WB01"},
	10190123:  {"ZYDUSLIFE", "CHC"},
	2951681:   {"EMIL", "EMI01"},
	10212345:  {"EXIDEIND"},
	2287873:   {"E2E"},
	10234567:  {"ELGIEQUIP"},
	261889:    {"FEDERALBNK"},
	10256789:  {"FINCABLES"},
	10267890:  {"JWL"},
	10278901:  {"JKLAKSHMI"},
	10289012:  {"JLHL"},
	134764292: {"KRITI"},
	3886081:   {"KIRLPNU"},
	// 10312345: {"DAAWAT"},
	993281:    {"SOMANYCERA"},
	4421377:   {"SENCO"},
	6546945:   {"SHAKTIPUMP", "SPI08"},
	6936321:   {"SWANENERGY", "SM09"},
	10367890:  {"SYNGENE"},
	1883649:   {"DATAPATTNS", "DPI01"},
	3945985:   {"TITAGARH", "TW04"},
	6088449:   {"ABSMARINE", "AMSL"},
	7374081:   {"SANATHAN", "STL09"},
	108033:    {"BHARATFORG", "BFC"},
	3682561:   {"POCL", "PO03"},
	2307585:   {"ANUP", "ANUP54246"},
	4801281:   {"KALAMANDIR", "SSK01"},
	3712257:   {"BALAMINES", "BA05"},
	3780097:   {"V2RETAIL", "VR02"},
	7481345:   {"ONESOURCE", "OSPL"},
	3926273:   {"RAIN", "PC13"},
	3608577:   {"SIYSIL", "SSM05"},
	6925313:   {"SAGILITY", "SIL25"},
	866305:    {"SWARAJENG", "SE"},
	290305:    {"APOLLO", "AMS"},
	7126785:   {"SAILIFE", ""},
	138443524: {"DYCL", "DC13"},
	6139905:   {"AIMTRON"},
	5552641:   {"DIXON", "DT07"},
	2962689:   {"FORCEMOT" /* "FORCEMOT" */},
}
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Response struct {
	NoBlockDeals        bool                `json:"noBlockDeals"`
	BulkBlockDeals      []BulkBlockDeal     `json:"bulkBlockDeals"`
	MarketDeptOrderBook MarketDeptOrderBook `json:"marketDeptOrderBook"`
	SecurityWiseDP      SecurityWiseDP      `json:"securityWiseDP"`
}

type BulkBlockDeal struct {
	Name string `json:"name"`
}

type MarketDeptOrderBook struct {
	TotalBuyQuantity  int         `json:"totalBuyQuantity"`
	TotalSellQuantity int         `json:"totalSellQuantity"`
	Open              float64     `json:"open"`
	Bid               []Order     `json:"bid"`
	Ask               []Order     `json:"ask"`
	TradeInfo         TradeInfo   `json:"tradeInfo"`
	ValueAtRisk       ValueAtRisk `json:"valueAtRisk"`
}

type Order struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type TradeInfo struct {
	TotalTradedVolume  json.Number `json:"totalTradedVolume"`
	TotalTradedValue   float64     `json:"totalTradedValue"`
	TotalMarketCap     float64     `json:"totalMarketCap"`
	Ffmc               float64     `json:"ffmc"`
	ImpactCost         float64     `json:"impactCost"`
	CMDailyVolatility  string      `json:"cmDailyVolatility"`
	CMAnnualVolatility string      `json:"cmAnnualVolatility"`
	MarketLot          string      `json:"marketLot"`
	ActiveSeries       string      `json:"activeSeries"`
}

type ValueAtRisk struct {
	SecurityVar       float64 `json:"securityVar"`
	IndexVar          float64 `json:"indexVar"`
	VarMargin         float64 `json:"varMargin"`
	ExtremeLossMargin float64 `json:"extremeLossMargin"`
	AdhocMargin       float64 `json:"adhocMargin"`
	ApplicableMargin  float64 `json:"applicableMargin"`
}

type SecurityWiseDP struct {
	QuantityTraded           int     `json:"quantityTraded"`
	DeliveryQuantity         int     `json:"deliveryQuantity"`
	DeliveryToTradedQuantity float64 `json:"deliveryToTradedQuantity"`
	SeriesRemarks            *string `json:"seriesRemarks"` // Pointer to handle null
	SecWiseDelPosDate        string  `json:"secWiseDelPosDate"`
}

var DeliveryQuantityTrend = make(map[string][]float64)

type Trend string

const (
	UpTrend   Trend = "↑"
	DownTrend Trend = "↓"
	FlatTrend Trend = "↔"
)

var DeliveryTrend = make(map[string]Trend)
var DeliveryValue = make(map[string]float64)

var cache = make(map[string]struct {
	value     float64
	timestamp time.Time
})

var count = make(map[string]int)
var globalRateLimitHit time.Time

// Function to make an HTTP call to the NSE historical data API
func fetchDeliveryToTradedQuantity(symbol string) float64 {

	if !globalRateLimitHit.IsZero() && time.Since(globalRateLimitHit) < 5*time.Minute {
		if cached, found := cache[symbol]; found {
			fmt.Println("Using cached value due to global rate limit for symbol:", symbol)
			return cached.value
		}
		return 0
	}
	// Check if the symbol is in the cache and if the cache is still valid
	if cached, found := cache[symbol]; found {
		if time.Since(cached.timestamp) < 25*time.Minute {
			return cached.value
		}
		// Clear the cache if it's older than 5 minutes
		delete(cache, symbol)
	}
	count[symbol]++
	url := fmt.Sprintf("https://www.nseindia.com/api/quote-equity?symbol=%s&section=trade_info", symbol)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return 0
	}
	req.Header.Add("Cookie", os.Getenv("Cookie"))
	req.Header.Add("User-Agent", "PostmanRuntime/7.43.0")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	// req.Header.Add("Postman-Token", "8ee56bdc-1204-46d1-a552-579dc75723c3")
	req.Header.Add("Host", "www.nseindia.com")
	req.Header.Add("Connection", "keep-alive")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		//
		return 0
	}
	defer res.Body.Close()

	var body []byte
	body, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		str := string(body)
		if includes := "Resource not found"; str == includes {
			fmt.Println("Resource not found for symbol:", symbol)
			return 0
		}
		getNSECookie()
		if includes := "Access Denied"; str == includes {
			fmt.Println("Rate limit exceeded. Please try after some time for symbol:", symbol)
			globalRateLimitHit = time.Now()
			if cached, found := cache[symbol]; found {
				return cached.value
			}
			return 0
		}
		fmt.Println("Error unmarshalling response:", err)
		return 0
	}

	// Cache the response
	cache[symbol] = struct {
		value     float64
		timestamp time.Time
	}{
		value:     response.SecurityWiseDP.DeliveryToTradedQuantity,
		timestamp: time.Now(),
	}

	fmt.Printf("Response Body:%s %f\n", symbol, response.SecurityWiseDP.DeliveryToTradedQuantity)
	return response.SecurityWiseDP.DeliveryToTradedQuantity
}

var every5MinChannel = time.Tick(5 * time.Minute)

func initDeliveryTrend() {
	go func() {
		for range every5MinChannel {
			for _, k := range instruments {
				time.Sleep(time.Second * 30)
				val := fetchDeliveryToTradedQuantity(k[0])
				DeliveryValue[k[0]] = val
				DeliveryQuantityTrend[k[0]] = append(DeliveryQuantityTrend[k[0]], val)
				if len(DeliveryQuantityTrend[k[0]]) > 20 {
					DeliveryQuantityTrend[k[0]] = DeliveryQuantityTrend[k[0]][:20]
				}
				DeliveryTrend[k[0]] = determineTrend(DeliveryQuantityTrend[k[0]])
			}
		}
	}()
}

func determineTrend(values []float64) Trend {
	if len(values) < 2 {
		return FlatTrend // Not enough data to determine trend
	}

	upCount := 0
	downCount := 0

	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1] {
			upCount++
		} else if values[i] < values[i-1] {
			downCount++
		}
	}

	if upCount > downCount {
		return UpTrend
	} else if downCount > upCount {
		return DownTrend
	} else {
		return FlatTrend
	}
}
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Struct to represent the entire JSON response
type BulkDealsResponse struct {
	AsOnDate       string          `json:"as_on_date"`
	BulkDealsData  []BulkDealData  `json:"BULK_DEALS_DATA"`
	BulkDeals      string          `json:"BULK_DEALS"`
	ShortDeals     string          `json:"SHORT_DEALS"`
	BlockDeals     string          `json:"BLOCK_DEALS"`
	ShortDealsData []ShortDealData `json:"SHORT_DEALS_DATA"`
	BlockDealsData []BlockDealData `json:"BLOCK_DEALS_DATA"`
}

// Struct for bulk deals data
type BulkDealData struct {
	Date       string `json:"date"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	ClientName string `json:"clientName"`
	BuySell    string `json:"buySell"`
	Qty        string `json:"qty"`
	WATP       string `json:"watp"`
	Remarks    string `json:"remarks"`
}

// Struct for short deals data
type ShortDealData struct {
	Date       string  `json:"date"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	ClientName *string `json:"clientName"`
	BuySell    *string `json:"buySell"`
	Qty        string  `json:"qty"`
	WATP       *string `json:"watp"`
	Remarks    *string `json:"remarks"`
}

// Struct for block deals data
type BlockDealData struct {
	Date       string  `json:"date"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	ClientName string  `json:"clientName"`
	BuySell    string  `json:"buySell"`
	Qty        string  `json:"qty"`
	WATP       string  `json:"watp"`
	Remarks    *string `json:"remarks"`
}

func fetchBulkDeals() {
	// url := "https://www.nseindia.com/api/snapshot-capital-market-largedeal"
	url := "https://www.nseindia.com/api/snapshot-capital-market-largedeal"
	// Create an HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add necessary headers
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Cookie", os.Getenv("Cookie")) // Use a valid NSE cookie if required

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Parse the JSON response
	var bulkDealsResponse BulkDealsResponse
	err = json.Unmarshal(body, &bulkDealsResponse)
	if err != nil {
		fmt.Println("Error unmarshalling response:", err)
		fmt.Println("Response Body:", string(body)) // Print raw response for debugging
		return
	}
	stocks := []string{} // Example stock symbols
	for _, val := range instruments {
		stocks = append(stocks, val[0])
	}
	// Print the bulk deals
	for _, deal := range bulkDealsResponse.BulkDealsData {
		// if symbol in stocks array
		// Print the details of each bulk deal
		if contains(stocks, deal.Symbol) {
			fmt.Printf("\033[31mDate: %s, Symbol: %s, Name: %s, Client: %s, Buy/Sell: %s, Quantity: %s, WATP: %s, Remarks: %s\033[0m\n",
				deal.Date, deal.Symbol, deal.Name, deal.ClientName, deal.BuySell, deal.Qty, deal.WATP, deal.Remarks)
		}
	}

	findBulkBoughtStock(bulkDealsResponse)
}

func findBulkBoughtStock(bulkDeals BulkDealsResponse) {
	quantities := make(map[string]struct {
		buyQty  int64
		sellQty int64
	})
	for _, deal := range bulkDeals.BulkDealsData {
		qty, err := strconv.ParseInt(deal.Qty, 10, 64)
		if err != nil {
			return
		}

		if deal.BuySell == "BUY" {
			quantities[deal.Symbol] = struct {
				buyQty  int64
				sellQty int64
			}{
				buyQty:  quantities[deal.Symbol].buyQty + qty,
				sellQty: quantities[deal.Symbol].sellQty,
			}
		} else if deal.BuySell == "SELL" {
			quantities[deal.Symbol] = struct {
				buyQty  int64
				sellQty int64
			}{
				buyQty:  quantities[deal.Symbol].buyQty,
				sellQty: quantities[deal.Symbol].sellQty + qty,
			}
		}
	}
	for symbol, qty := range quantities {
		if qty.buyQty > 0 && qty.sellQty == 0 {
			fmt.Printf("\033[32mBulk Bought Stock: %s, Buy Quantity: %d\033[0m\n", symbol, qty.buyQty)
		} else if qty.sellQty > 0 && qty.buyQty == 0 {
			fmt.Printf("\033[31mBulk Sold Stock: %s, Sell Quantity: %d\033[0m\n", symbol, qty.sellQty)
		} else if qty.buyQty > 0 && qty.sellQty > 0 {
			// fmt.Printf("\033[33mBulk Traded Stock: %s, Buy Quantity: %d, Sell Quantity: %d\033[0m\n", symbol, qty.buyQty, qty.sellQty)
		}
	}
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// LatestAnnouncements holds the latest announcements data
type LatestAnnouncements struct {
	Data []Announcement `json:"data"`
}

// Announcement represents a single announcement
type Announcement struct {
	Symbol        string `json:"symbol"`
	BroadcastDate string `json:"broadcastdate"`
	Subject       string `json:"subject"`
}

// CorporateActions holds the corporate actions data
type CorporateActions struct {
	Data []CorporateAction `json:"data"`
}

// CorporateAction represents a single corporate action
type CorporateAction struct {
	Symbol  string `json:"symbol"`
	ExDate  string `json:"exdate"`
	Purpose string `json:"purpose"`
}

// ShareholdingPatterns holds the shareholding patterns data
type ShareholdingPatterns struct {
	Data map[string][]Shareholding `json:"data"`
}

// Shareholding represents a single shareholding entry
type Shareholding struct {
	Category   string `json:"category"`
	Percentage string `json:"percentage"`
}

// FinancialResults holds the financial results data
type FinancialResults struct {
	Data []FinancialResult `json:"data"`
}

// FinancialResult represents a single financial result
type FinancialResult struct {
	FromDate        *string `json:"from_date"`
	ToDate          string  `json:"to_date"`
	Expenditure     *string `json:"expenditure"`
	Income          string  `json:"income"`
	Audited         string  `json:"audited"`
	Cumulative      *string `json:"cumulative"`
	Consolidated    *string `json:"consolidated"`
	ReDilEPS        string  `json:"reDilEPS"`
	ReProLossBefTax string  `json:"reProLossBefTax"`
	ProLossAftTax   string  `json:"proLossAftTax"`
	ReBroadcastTime string  `json:"re_broadcast_timestamp"`
	XBRLAttachment  *string `json:"xbrl_attachment"`
	NAAttachment    *string `json:"na_attachment"`
}

// BoardMeetings holds the board meeting data
type BoardMeetings struct {
	Data []BoardMeeting `json:"data"`
}

// BoardMeeting represents a single board meeting
type BoardMeeting struct {
	Symbol      string `json:"symbol"`
	Purpose     string `json:"purpose"`
	MeetingDate string `json:"meetingdate"`
}

var corporateActionsCache = make(map[string]string)

func cacheCorporateActions() error {
	for symbol := range instruments {
		// Fetch corporate actions for each symbol
		actions, err := fetchCorporateActions(instruments[symbol][0])
		if err != nil {
			// log.Printf("Error fetching corporate actions for %s: %v", instruments[symbol], err)
			continue
		}

		// Cache the corporate actions in the map'
		if actions == "" {
			// log.Printf("No upcoming corporate actions found for %s", instruments[symbol])
			continue
		}
		corporateActionsCache[instruments[symbol][0]] = actions
		log.Printf("Cached %s corporate actions for %s", actions, instruments[symbol][0])
	}
	return nil
}

func fetchCorporateActions(symbol string) (string, error) {
	url := fmt.Sprintf("https://www.nseindia.com/api/top-corp-info?market=equities&symbol=%s", symbol)
	// log.Printf("Fetching latest announcements for %s with URL: %s", symbol, url)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Set headers to mimic browser request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", symbol, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Cookie", os.Getenv("Cookie")) // Use a valid NSE cookie if required

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest announcements for %s: %v", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code for %s: %d", symbol, resp.StatusCode)
	}

	// Unmarshal the JSON response
	var response struct {
		CorporateActions CorporateActions `json:"corporate_actions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode latest announcements for %s: %v", symbol, err)
	}

	// Log the announcements for debugging
	actions := response.CorporateActions.Data
	if len(actions) == 0 {
		return "", fmt.Errorf("no corporate actions found for %s", symbol)
	}
	exDate, err := time.Parse("02-Jan-2006", actions[0].ExDate)
	if err != nil {
		log.Printf("Error parsing ExDate for %s: %v", symbol, err)
		return "", fmt.Errorf("invalid ExDate format for %s: %v", symbol, err)
	}

	if exDate.After(time.Now()) {
		statement := fmt.Sprintf("Date: %s, Purpose: %s", actions[0].ExDate, actions[0].Purpose)
		return statement, nil
	}
	return "", fmt.Errorf("no upcoming corporate actions found for %s", symbol)
}
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
package main

import (
	"fmt"
	"net/http"
	"os"
)

func getNSECookie() {
	url := "https://www.nseindia.com/get-quotes/equity?symbol=POWERGRID"

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "PostmanRuntime/7.43.0")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")

	// req.Header.Add("Postman-Token", "8ee56bdc-1204-46d1-a552-579dc75723c3")
	req.Header.Add("Host", "www.nseindia.com")
	//req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Connection", "keep-alive")
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	// _, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Println("Error reading response body:", err)
	// 	return
	// }
	var allCookies string
	for _, cookie := range resp.Cookies() {
		// fmt.Printf("Cookie: %s=%s\n", cookie.Name, cookie.Value)
		allCookies += fmt.Sprintf("%s=%s; ", cookie.Name, cookie.Value)
	}
	os.Setenv("Cookie", allCookies)
	// fmt.Println(allCookies)
}
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

type PriceForecastResponse struct {
	Success int               `json:"success"`
	Data    PriceForecastData `json:"data"`
}

type PriceForecastData struct {
	GraphData   [][]float64 `json:"graphData"`
	High        string      `json:"high"`
	Mean        string      `json:"mean"`
	Low         string      `json:"low"`
	DisplayLock string      `json:"displayLock"`
}

type PriceForecast struct {
	Symbol        string
	CurrentPrice  float64
	OneYearTarget float64
	UpsidePercent float64
}

func fetchPriceForecast(symbol string) (PriceForecast, error) {
	scId := symbol
	url := fmt.Sprintf("https://api.moneycontrol.com/mcapi/v1/stock/estimates/price-forecast?scId=%s&ex=N&deviceType=W", scId)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PriceForecast{}, err
	}
	req.Header.Set("auth-token", "BzjBBOiKtB0savNfOXSZN96ffcSZtX69uJRhu4blKWhAudp9RwGfxPAZhNmU8kG37qpKenM0YSIhz6ctWhluIw")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.4 Safari/605.1.15")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.moneycontrol.com")
	req.Header.Set("Referer", "https://www.moneycontrol.com/")

	resp, err := client.Do(req)
	if err != nil {
		return PriceForecast{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PriceForecast{}, fmt.Errorf("status %d for %s", resp.StatusCode, symbol)
	}

	var pfResp PriceForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&pfResp); err != nil {
		return PriceForecast{}, err
	}

	// Extract current price from the last graphData point
	graph := pfResp.Data.GraphData
	var currentPrice float64
	if len(graph) > 0 && len(graph[len(graph)-1]) > 1 {
		currentPrice = graph[len(graph)-1][1]
	}

	// Parse target price (mean) and calculate upside
	mean, _ := strconv.ParseFloat(pfResp.Data.Mean, 64)
	var upside float64
	if currentPrice > 0 {
		upside = ((mean - currentPrice) / currentPrice) * 100
	}

	return PriceForecast{
		Symbol:        symbol,
		CurrentPrice:  currentPrice,
		OneYearTarget: mean,
		UpsidePercent: upside,
	}, nil
}

func printAllPriceForecasts() {
	fileName := fmt.Sprintf("PriceForecasts_%s", time.Now().Format("02-01-2006"))
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer f.Close()
	w := tabwriter.NewWriter(f, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Stock Symbol\tCurrent Price (₹)\t1-Year Target Price (₹)\t1-Year Upside (%)")

	type forecastRow struct {
		Symbol        string
		CurrentPrice  float64
		OneYearTarget float64
		UpsidePercent float64
	}

	var forecasts []forecastRow

	for _, v := range instruments {
		if len(v) < 2 {
			continue
		}
		forecast, err := fetchPriceForecast(v[1])
		if err != nil {
			continue
		}
		forecasts = append(forecasts, forecastRow{
			Symbol:        v[0],
			CurrentPrice:  forecast.CurrentPrice,
			OneYearTarget: forecast.OneYearTarget,
			UpsidePercent: forecast.UpsidePercent,
		})
		time.Sleep(1 * time.Second)
	}

	// Sort in descending order of UpsidePercent
	sort.Slice(forecasts, func(i, j int) bool {
		return forecasts[i].UpsidePercent > forecasts[j].UpsidePercent
	})

	for _, forecast := range forecasts {
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f%%\n",
			forecast.Symbol, forecast.CurrentPrice, forecast.OneYearTarget, forecast.UpsidePercent)
	}
	w.Flush()
}
package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendTelegramMessage(message string) {
	bot, err := tgbotapi.NewBotAPI("8487412493:AAGiIJWP_VKEfkJ6cbanAmfE_szoLDfNZE4")
	if err != nil {
		log.Fatal(err)
	}

	msg := tgbotapi.NewMessage(1784293951, message)
	if _, err := bot.Send(msg); err != nil {
		log.Fatal(err)
	}
}
