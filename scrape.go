package main

import (
	"encoding/json"
	"context"
	"fmt"
	"strconv"
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
	
	tables := []string{"crypto_currencies", "gainers_title", "losers_title"}
	
	var scrapedItemsSlice []ScrapedItem

	for _, table := range tables {
		tickers, names, stockValues, percentChanges := retrieveTargetAttributes(table, ctx)

		printResults(tickers, names, stockValues, percentChanges)

		parsedItems := createDataPoints(tickers, names, stockValues, percentChanges)

		for _, item := range parsedItems {
			scrapedItemsSlice = append(scrapedItemsSlice, item)
		}
	}

	jsonData := serializeDataPoints(scrapedItemsSlice)

	fmt.Println("Serialized JSON data:\n")
	fmt.Println(string(jsonData))
}

type ScrapedItem struct {
	Ticker string
	Name string
	StockValue float64
	PercentChange float64
}

func generateScrapedItem(ticker, name string, stockValue, percentChange float64) ScrapedItem {
	return ScrapedItem{
		Ticker: ticker,
		Name: name,
		StockValue: stockValue,
		PercentChange: percentChange,
	}
}

func retrieveTargetAttributes(table string, ctx context.Context) (tickers, names, stockValues, percentChanges []*cdp.Node) {
	targetUrl := "https://finance.yahoo.com"

	selector := `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:first-child > a`	
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	tickers = scrapeData(ctx, targetUrl, selector)
	names = tickers // The []*cdp.Node slice returned from the first scrape contains both the ticker and full name needed

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:nth-child(2)> fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	stockValues = scrapeData(ctx, targetUrl, selector)

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:last-child > fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	percentChanges = scrapeData(ctx, targetUrl, selector)

	return tickers, names, stockValues, percentChanges
}

func scrapeData(ctx context.Context, targetUrl, selector string) []*cdp.Node {
	var results []*cdp.Node

	err := chromedp.Run(ctx,
		chromedp.Navigate(targetUrl),
		chromedp.WaitVisible(selector),
		chromedp.Nodes(selector, &results, chromedp.ByQueryAll),
	)
	if err != nil {
		mlog.Warning("Failed to scrape with selector: %s\n", selector)
		mlog.Error(err)
	}

	mlog.Info("Finished scraping with selector: %s", selector)

	return results
}

// Print and log the results
func printResults(tickers, names, stockValues, percentChanges []*cdp.Node) {
	timestamp := time.Now()

	for i, _ := range tickers {
		ticker, name, floatValue, floatChange := parseNodes(tickers[i], names[i], stockValues[i], percentChanges[i])

		// using magic number (40) for the stock names here for now

		fmt.Printf("[%s] Element %d: %-8s | %-40s | %-9.2f$ | %-8.4f%%\n", timestamp.Format("2006/01/02 15:04:05 EDT"), i+1, ticker, name, floatValue, floatChange)
	}
	fmt.Println("\n")
}

func createDataPoints(tickers, names, stockValues, percentChanges []*cdp.Node) []ScrapedItem {
	var scrapedItems []ScrapedItem
	
	for i, _ := range tickers {
		ticker, name, floatValue, floatChange := parseNodes(tickers[i], names[i], stockValues[i], percentChanges[i])

		item := generateScrapedItem(ticker, name, floatValue, floatChange)

		scrapedItems = append(scrapedItems, item)
	}

	return scrapedItems
}

func parseNodes(tickerNode, nameNode, stockValueNode, percentChangeNode *cdp.Node) (ticker, name string, floatChange, floatValue float64) {	
	ticker = tickerNode.AttributeValue("href")
	ticker = strings.TrimPrefix(ticker, "/quote/")

	name = nameNode.AttributeValue("title")
	
	stockValue := stockValueNode.AttributeValue("value")
	floatValue, err := strconv.ParseFloat(string(stockValue), 64)
	if err != nil {
		fmt.Printf("Error while parsing stockValue to float64: %s", err)
		mlog.Warning("Error while parsing stockValue to float64")
		mlog.Error(err)
	}
	
	percentChange := percentChangeNode.AttributeValue("value")
	floatChange, err = strconv.ParseFloat(string(percentChange), 64)
	if err != nil {
		fmt.Printf("Error while parsing percentChange to float64: %s", err)
		mlog.Warning("Error while parsing percentChange to float64")
		mlog.Error(err)
	}

	return ticker, name, floatValue, floatChange
}

func serializeDataPoints(scrapedItems []ScrapedItem) []byte {
	jsonData, _ := json.Marshal(scrapedItems)

	return jsonData
}

