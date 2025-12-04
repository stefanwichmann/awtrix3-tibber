package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"time"
)

var tibberDemoToken = "3A77EECF61BD445F47241A5A36202185C35AF3AF58609E19B53F3A8872AD7BE1-1"
var flagTibberToken = flag.String("tibberToken", lookupEnv("TIBBER_TOKEN", tibberDemoToken), "Your Tibber developer API token")
var flagTibberHomeId = flag.String("tibberHomeId", lookupEnv("TIBBER_HOME_ID", ""), "The Id of your Tibber home")
var flagAwtrixIP = flag.String("awtrixIP", lookupEnv("AWTRIX_IP", "127.0.0.1"), "The IPv4 address of your Awtrix3 device")

var customAppName = "tibberPrices"
var chartBarCount = 36 - 12

var knownPrices []tibberPrice

func main() {
	flag.Parse()
	if *flagTibberToken == tibberDemoToken {
		log.Print("Using Tibber demo token. Please provide your own developer token via --tibberToken for real data")
	}

	for {
		fetchPrices()
		updateKnowPrices()
		updateDisplay()

		// Make sure to update at the next full hour to update the bar chart correctly
		nextUpdate := durationUntilNextFullHour()
		log.Printf("Sleeping for %s", nextUpdate)
		time.Sleep(nextUpdate)
	}
}

func fetchPrices() {
	log.Println("Fetching Tibber prices...")
	prices, err := readTibberPrices(*flagTibberToken, *flagTibberHomeId)
	if err != nil {
		log.Fatalf("Could not fetch prices: %v", err)
	}

	knownPrices = prices
}

func updateKnowPrices() {
	if len(knownPrices) == 0 {
		return
	}

	historicPrices, upcomingPrices := splitPrices(knownPrices)

	// Limit historic prices to the last 4
	if len(historicPrices) >= 4 {
		historicPrices = historicPrices[len(historicPrices)-4:]
	}
	relevantPrices := append(historicPrices, upcomingPrices...)

	log.Print("Updating known prices")
	knownPrices = relevantPrices
}

func updateDisplay() {
	relevantPrices := knownPrices
	if len(relevantPrices) > chartBarCount {
		relevantPrices = relevantPrices[:chartBarCount]
	}

	// Print prices
	log.Printf("Identified the following relevant prices")
	for _, price := range relevantPrices {
		log.Printf("Starting at %s: %f", price.StartsAt, price.Total)
	}

	currentPriceString := "?"
	currentPrice, err := currentPrice(relevantPrices)
	if err == nil {
		currentPriceString = fmt.Sprintf("%d¢", roundedPrice(currentPrice.Total))
	}

	commandsText := []AwtrixDrawCommand{{Command: "dt", X: 0, Y: 2, Text: currentPriceString, Color: "#FFFFFF"}}
	commandsChart := mapToDrawingCommands(relevantPrices)
	app := AwtrixApp{Draw: append(commandsText, commandsChart...)}

	log.Printf("Drawing %d prices...", len(commandsChart))
	err = postApplication(*flagAwtrixIP, customAppName, app)
	if err != nil {
		log.Fatalf("Could not update custom application: %v", err)
	}
}

func splitPrices(prices []tibberPrice) ([]tibberPrice, []tibberPrice) {
	var historicPrices []tibberPrice
	var upcomingPrices []tibberPrice

	for _, price := range prices {
		if price.StartsAt.Before(time.Now()) {
			historicPrices = append(historicPrices, price)
		} else if price.StartsAt.After(time.Now()) {
			upcomingPrices = append(upcomingPrices, price)
		} else {
			log.Fatalf("Can't place price %+v", price)
		}
	}

	return historicPrices, upcomingPrices
}

func currentPrice(prices []tibberPrice) (tibberPrice, error) {
	for _, price := range prices {
		if price.StartsAt.Day() == time.Now().Day() && price.StartsAt.Hour() == time.Now().Hour() {
			return price, nil
		}
	}

	return tibberPrice{}, fmt.Errorf("could not find current price")
}

func mapToDrawingCommands(prices []tibberPrice) []AwtrixDrawCommand {
	var commands []AwtrixDrawCommand

	if len(prices) == 0 {
		return commands
	}

	// Find min and max price
	minPrice := prices[0].Total
	maxPrice := prices[0].Total
	for _, price := range prices {
		if price.Total < minPrice {
			minPrice = price.Total
		}
		if price.Total > maxPrice {
			maxPrice = price.Total
		}

	}

	// Map price range to pixel range
	yMin := 1
	yMax := 8
	slope := 1.0 * float64(yMax-yMin) / (maxPrice - minPrice)
	xOffset := 12

	for i, price := range prices {
		scaledPrice := float64(yMin) + slope*(price.Total-minPrice)
		color := mapPriceToColor(price)
		log.Printf("Mapping price %f to %d (Min: %f, Max: %f, Color: %s)", price.Total, int(scaledPrice), minPrice, maxPrice, color)
		command := AwtrixDrawCommand{Command: "df", X: xOffset + i, Y: yMax - int(scaledPrice), Width: 1, Height: yMax, Color: color}
		commands = append(commands, command)
	}

	return commands
}

func roundedPrice(price float64) int {
	return int(math.Round(price * 100))
}

func mapPriceToColor(price tibberPrice) string {
	if price.StartsAt.Day() == time.Now().Day() && price.StartsAt.Hour() == time.Now().Hour() {
		return "#FFFFFF"
	}

	switch {
	case price.Total <= 0:
		return "#215d6e"
	case price.Total < 0.25:
		return "#5ba023"
	case price.Total < 0.28:
		return "#7b9632"
	case price.Total < 0.30:
		return "#9b8c41"
	case price.Total < 0.33:
		return "#ba8250"
	case price.Total < 0.35:
		return "#da785f"
	default:
		return "#fa6e6e"
	}
}
