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
