package main

import (
	// "encoding/csv"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp" // need this for the Type to capture multiple nodes
	"github.com/chromedp/chromedp"
	"github.com/jbrodriguez/mlog"
)

func main() {
	mlog.StartEx(mlog.LevelInfo, "makescraper.log", 5*1024*1024, 5)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var results []*cdp.Node

	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://finance.yahoo.com`),
		// Use chromedp.Nodes to retrieve all nodes that match the selector
		chromedp.WaitVisible(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:last-child > fin-streamer > span`),
		chromedp.Nodes(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:first-child > a`, &results, chromedp.ByQueryAll),
)
if err != nil {
	mlog.Warning("Failed to scrape: %v", err)
	mlog.Error(err)
}

// Print the results
for i, result := range results {
	ticker := result.AttributeValue("href")
	ticker = strings.TrimPrefix(ticker, "/quote/")
	fmt.Printf("Element %d: %v\n", i+1, ticker)
}
	// This was successfully grabbing one result from the many available

	// var tickers, names, percentChanges string
	// err := chromedp.Run(ctx,
	// 	chromedp.Navigate(`https://finance.yahoo.com`),
	// 	// Wait for the element to be visible
	// 	chromedp.WaitVisible(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:last-child > fin-streamer > span`),
	// 	// Extract the text of the element
	// 	chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:first-child > a`, &tickers, chromedp.ByQueryAll),
	// 	chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:first-child > p`, &names, chromedp.ByQueryAll),
	// 	chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:last-child > fin-streamer > span`, &percentChanges, chromedp.ByQueryAll),
	// )
	// if err != nil {
	// 	mlog.Warning("Failed to scrape: %v", err)
	// 	mlog.Error(err)
	// }

	// fmt.Printf("scraped tickers: %v\n", tickers)
	// for _, item := range tickers {
	// 	dataPoint := createDataPoint(string(tickers[item]), string(names[item]), string(percentChanges[item]))

	// 	fmt.Printf("Scraped text: %s, %s, %s\n", dataPoint.Ticker, dataPoint.Name, dataPoint.PercentChange)
	// 	mlog.Info("Scraped text: %s, %s, %s\n", dataPoint.Ticker, dataPoint.Name, dataPoint.PercentChange)
	// }
}

type ScrapedItem struct {
	Ticker string
	Name string
	PercentChange string
}

func createDataPoint(ticker, name, percentChange string) ScrapedItem {
	return ScrapedItem{
		Ticker: ticker,
		Name: name,
		PercentChange: percentChange,
	}
}
