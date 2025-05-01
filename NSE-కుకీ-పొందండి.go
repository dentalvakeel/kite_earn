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

	req.Header.Add("Postman-Token", "8ee56bdc-1204-46d1-a552-579dc75723c3")
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
