package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type tibberResponse struct {
	Data tibberData `json:"data"`
}

type tibberData struct {
	Viewer tibberViewer `json:"viewer"`
}

type tibberViewer struct {
	Homes []tibberHome `json:"homes"`
}

type tibberHome struct {
	Id                  string             `json:"id"`
	Nickname            string             `json:"appNickname"`
	Timezone            string             `json:"timeZone"`
	Address             tibberAddress      `json:"address"`
	CurrentSubscription tibberSubscription `json:"currentSubscription"`
}

type tibberAddress struct {
	AddressLine1 string `json:"address1"`
	AddressLine2 string `json:"address2"`
	AddressLine3 string `json:"address3"`
	PostalCode   string `json:"postalCode"`
	City         string `json:"city"`
	Country      string `json:"country"`
	Latitude     string `json:"latitude"`
	Longitude    string `json:"longitude"`
}

type tibberSubscription struct {
	PriceInformation tibberPriceInformation `json:"priceInfo"`
}

type tibberPriceInformation struct {
	Current  tibberPrice   `json:"current"`
	Today    []tibberPrice `json:"today"`
	Tomorrow []tibberPrice `json:"tomorrow"`
}

type tibberPrice struct {
	Total    float64   `json:"total"`
	StartsAt time.Time `json:"startsAt"`
}

func readTibberPrices(token string, tibberHomeId string) ([]tibberPrice, error) {
	prices, err := readCurrentSbuscriptions(token)
	if err != nil {
		return []tibberPrice{}, err
	}

	if len(prices.Data.Viewer.Homes) == 0 {
		return []tibberPrice{}, fmt.Errorf("could not find any homes in %+v", prices)
	}

	log.Printf("Found %d home(s)...", len(prices.Data.Viewer.Homes))
	for _, home := range prices.Data.Viewer.Homes {
		log.Printf("Id: %s at %s (%s, %s)", home.Id, home.Address.AddressLine1, home.Address.City, home.Address.Country)
	}

	if len(prices.Data.Viewer.Homes) == 0 {
		return []tibberPrice{}, fmt.Errorf("could not find any homes in %+v", prices)
	}

	if len(prices.Data.Viewer.Homes) > 1 && len(tibberHomeId) == 0 {
		return []tibberPrice{}, fmt.Errorf("found more than one home and the requested one was not specified")
	}

	// If we only have one home, return it's prices
	if len(prices.Data.Viewer.Homes) == 1 {
		allPrices := prices.Data.Viewer.Homes[0].CurrentSubscription.PriceInformation.Today
		allPrices = append(allPrices, prices.Data.Viewer.Homes[0].CurrentSubscription.PriceInformation.Tomorrow...)
		return allPrices, nil
	}

	// Find the requested home by Id
	for _, home := range prices.Data.Viewer.Homes {
		if home.Id == tibberHomeId {
			allPrices := home.CurrentSubscription.PriceInformation.Today
			allPrices = append(allPrices, home.CurrentSubscription.PriceInformation.Tomorrow...)
			log.Printf("Found %d prices for home %s", len(allPrices), tibberHomeId)
			return allPrices, nil
		}
	}

	return []tibberPrice{}, fmt.Errorf("could not find prices matching home Id %s", tibberHomeId)
}

func readCurrentSbuscriptions(token string) (tibberResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.tibber.com/v1-beta/gql", nil)
	if err != nil {
		return tibberResponse{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Content-Type", "application/json")

	q := req.URL.Query()
	query := "{viewer{homes{id appNickname timeZone address{address1 address2 address3 postalCode city country latitude longitude} currentSubscription{ priceInfo{ current{total energy tax startsAt} today{total energy tax startsAt} tomorrow{total energy tax startsAt}}}}}}"
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return tibberResponse{}, err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return tibberResponse{}, err
	}

	var response tibberResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return tibberResponse{}, err
	}

	return response, nil
}
