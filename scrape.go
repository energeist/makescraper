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

type ScrapedItem struct {
	Symbol string
	Name string
	StockValue float64
	PercentChange float64
}

func main() {
	mlog.StartEx(mlog.LevelInfo, "makescraper.log", 5*1024*1024, 5)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	tables := []string{"crypto_currencies", "gainers_title", "losers_title"}
	
	var scrapedItemsSlice []ScrapedItem

	dataMap := make(map[string]map[string]ScrapedItem)
	fmt.Println(dataMap)

	for _, table := range tables {

		// store all of this in a map so that it's easier to recall

		// also do it concurrently instead of sequentially?

		// this function should take map in, mutate and return map
		symbols, names, stockValues, percentChanges := retrieveTargetAttributes(table, ctx)
		// dataMap = retrieveAndMapTargetAttributes(table, dataMap, ctx)

		// this will take a map now
		printResults(symbols, names, stockValues, percentChanges)
		
		parsedItems := createDataPoints(symbols, names, stockValues, percentChanges)
		
		for _, item := range parsedItems {
			scrapedItemsSlice = append(scrapedItemsSlice, item)
		}
	}


	jsonData := serializeDataPoints(scrapedItemsSlice)

	fmt.Println("Serialized JSON data:\n")
	fmt.Println(string(jsonData))

	writeJsonToFile(jsonData)
}

func generateScrapedItem(symbol, name string, stockValue, percentChange float64) ScrapedItem {
	return ScrapedItem{
		Symbol: symbol,
		Name: name,
		StockValue: stockValue,
		PercentChange: percentChange,
	}
}

func retrieveTargetAttributes(table string, ctx context.Context) (symbols, names, stockValues, percentChanges []*cdp.Node) {
	targetUrl := "https://finance.yahoo.com"

	selector := `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:first-child > a`	
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	symbols = scrapeData(ctx, targetUrl, selector)
	names = symbols // The []*cdp.Node slice returned from the first scrape contains both the symbol and full name needed

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:nth-child(2)> fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	stockValues = scrapeData(ctx, targetUrl, selector)

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:last-child > fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	percentChanges = scrapeData(ctx, targetUrl, selector)

	return symbols, names, stockValues, percentChanges
}

func retrieveAndMapTargetAttributes(table string, dataMap map[string]map[string]ScrapedItem , ctx context.Context) map[string]map[string]ScrapedItem {
	targetUrl := "https://finance.yahoo.com"

	selector := `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:first-child > a`	
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	symbols := scrapeData(ctx, targetUrl, selector)
	names := symbols // The []*cdp.Node slice returned from the first scrape contains both the symbol and full name needed

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:nth-child(2)> fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	stockValues := scrapeData(ctx, targetUrl, selector)

	selector = `section[data-yaft-module="tdv2-applet-` + table +`"] > table > tbody > tr > td:last-child > fin-streamer`
	fmt.Printf("Scraping on %s\n", selector)
	mlog.Info("Starting to scrape on selector: %s\n", selector)
	percentChanges := scrapeData(ctx, targetUrl, selector)

	for i, symbol := range symbols {
		symbolString := symbol.AttributeValue("href")
		symbolString = strings.TrimPrefix(symbolString, "/quote/")

		dataMap[table][symbolString] = generateScrapedItem(parseNodes(symbols[i], names[i], stockValues[i], percentChanges[i]))
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

func printResults(symbols, names, stockValues, percentChanges []*cdp.Node) {
	timestamp := time.Now()

	for i, _ := range symbols {
		symbol, name, floatValue, floatChange := parseNodes(symbols[i], names[i], stockValues[i], percentChanges[i])

		// using magic number (40) for the stock names here for now

		fmt.Printf("[%s] Element %d: %-8s | %-40s | %-9.2f$ | %-8.4f%%\n", timestamp.Format("2006/01/02 15:04:05 EDT"), i+1, symbol, name, floatValue, floatChange)
	}
	fmt.Println("\n")
}

func createDataPoints(symbols, names, stockValues, percentChanges []*cdp.Node) []ScrapedItem {
	var scrapedItems []ScrapedItem
	
	for i, _ := range symbols {
		symbol, name, floatValue, floatChange := parseNodes(symbols[i], names[i], stockValues[i], percentChanges[i])

		item := generateScrapedItem(symbol, name, floatValue, floatChange)

		scrapedItems = append(scrapedItems, item)
	}

	return scrapedItems
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

func serializeDataPoints(scrapedItems []ScrapedItem) []byte {
	jsonData, _ := json.Marshal(scrapedItems)

	return jsonData
}

func writeJsonToFile(jsonData []byte) {
	err := os.WriteFile("output.json", jsonData, 0644)
	if err != nil {
		mlog.Warning("Error writing JSON data to file")
		mlog.Error(err)

		panic(err)
	}

	mlog.Info("Successfully wrote serialized scraped data to scrapedData.json")
}

