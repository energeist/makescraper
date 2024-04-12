package main

import (
	// "encoding/csv"
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/jbrodriguez/mlog"
)

func main() {
	mlog.StartEx(mlog.LevelInfo, "makescraper.log", 5*1024*1024, 5)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var ticker, name, percentChange string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://finance.yahoo.com`),
		// Wait for the element to be visible
		chromedp.WaitVisible(`section[data-yaft-module="tdv2-applet-crypto_currencies"]`),
		// Extract the text of the element
		chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:first-child > a`, &ticker),
		chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:first-child > p`, &name),
		chromedp.Text(`section[data-yaft-module="tdv2-applet-crypto_currencies"] > table > tbody > tr > td:last-child > fin-streamer > span`, &percentChange),
	)
	if err != nil {
		mlog.Warning("Failed to scrape: %v", err)
		mlog.Error(err)
	}

	dataPoint := createDataPoint(ticker, name, percentChange)

	fmt.Printf("Scraped text: %s, %s, %s\n", dataPoint.Ticker, dataPoint.Name, dataPoint.PercentChange)
	mlog.Info("Scraped text: %s, %s, %s\n", dataPoint.Ticker, dataPoint.Name, dataPoint.PercentChange)
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
