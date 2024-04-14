package main

import (
	"encoding/json"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp" // need this for the Type to capture multiple nodes
	"github.com/chromedp/chromedp"
	"github.com/jbrodriguez/mlog"
)

var tables = [3]string{"crypto_currencies", "gainers_title", "losers_title"} // defined globally for ease of use

type ScrapedItem struct {
	Symbol string
	Name string
	StockValue float64
	PercentChange float64
	LastScraped time.Time
}

func main() {
	mlog.StartEx(mlog.LevelInfo, "makescraper.log", 5*1024*1024, 5)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	dataMap := make(map[string]map[string]ScrapedItem)

	for _, table := range tables {
		dataMap = retrieveAndMapTargetAttributes(table, dataMap, ctx)
	}
	
	fmt.Println("\nResults of data scraping:\n")
	printResults(dataMap)

	jsonData := serializeDataPoints(dataMap)

	fmt.Println("\nSerialized JSON data:\n")
	fmt.Println(string(jsonData))

	writeJsonToFile(jsonData)
}

func generateScrapedItem(symbol, name string, stockValue, percentChange float64, lastScraped time.Time) ScrapedItem {
	return ScrapedItem{
		Symbol: symbol,
		Name: name,
		StockValue: stockValue,
		PercentChange: percentChange,
		LastScraped: lastScraped,
	}
}

func retrieveAndMapTargetAttributes(table string, dataMap map[string]map[string]ScrapedItem , ctx context.Context) map[string]map[string]ScrapedItem {
	targetUrl := "https://finance.yahoo.com"
	timestamp := time.Now()

	if dataMap[table] == nil {
		dataMap[table] = make(map[string]ScrapedItem)
	}

	selector := `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:first-child > a`	
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	symbols := scrapeData(ctx, targetUrl, selector)
	names := symbols // The []*cdp.Node slice returned from the first scrape contains both the symbol and full name needed

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:nth-child(2)> fin-streamer`
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	stockValues := scrapeData(ctx, targetUrl, selector)

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:last-child > fin-streamer`
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	percentChanges := scrapeData(ctx, targetUrl, selector)

	for i, symbol := range symbols {
		symbolString := symbol.AttributeValue("href")
		symbolString = strings.TrimPrefix(symbolString, "/quote/")

		ticker, name, floatValue, floatChange := parseNodes(symbols[i], names[i], stockValues[i], percentChanges[i])

		dataMap[table][symbolString] = generateScrapedItem(ticker, name, floatValue, floatChange, timestamp)
	}
	
	return dataMap
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

		panic(err)
	}

	mlog.Info("Finished scraping with selector: %s", selector)

	return results
}

func printResults(dataMap map[string]map[string]ScrapedItem) {
	i := 1
	for _, table := range tables {
		for _, item := range dataMap[table] {
			symbol, name, floatValue, floatChange, lastScraped := item.Symbol, item.Name, item.StockValue, item.PercentChange, item.LastScraped

			// using magic number (40) for the stock names here for now
			fmt.Printf("[%s] Element %-2d: %-8s | %-40s | %-9.2f$ | %-8.4f%%\n", lastScraped.Format("2006/01/02 15:04:05 EDT"), i, symbol, name, floatValue, floatChange)
			i++
		}
	}
}

func parseNodes(symbolNode, nameNode, stockValueNode, percentChangeNode *cdp.Node) (symbol, name string, floatChange, floatValue float64) {	
	symbol = symbolNode.AttributeValue("href")
	symbol = strings.TrimPrefix(symbol, "/quote/")

	name = nameNode.AttributeValue("title")
	
	stockValue := stockValueNode.AttributeValue("value")
	floatValue, err := strconv.ParseFloat(string(stockValue), 64)
	if err != nil {
		fmt.Printf("Error while parsing stockValue to float64: %s", err)
		mlog.Warning("Error while parsing stockValue to float64")
		mlog.Error(err)

		panic(err)
	}
	
	percentChange := percentChangeNode.AttributeValue("value")
	floatChange, err = strconv.ParseFloat(string(percentChange), 64)
	if err != nil {
		fmt.Printf("Error while parsing percentChange to float64: %s", err)
		mlog.Warning("Error while parsing percentChange to float64")
		mlog.Error(err)

		panic(err)
	}

	return symbol, name, floatValue, floatChange
}

func serializeDataPoints(dataMap map[string]map[string]ScrapedItem) []byte {
	jsonData, _ := json.MarshalIndent(dataMap, "", "  ")

	return jsonData
}

func writeJsonToFile(jsonData []byte) {
	fileName := "output.json"

	err := os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		mlog.Warning("Error writing JSON data to file")
		mlog.Error(err)

		panic(err)
	}

	mlog.Info("Successfully wrote serialized scraped data to %s", fileName)
}

