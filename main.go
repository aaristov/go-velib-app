package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	endpoint      = "https://velib-metropole-opendata.smoove.pro/opendata/Velib_Metropole/station_status.json"
	supabaseTable = "rest/v1/stations"
)

var (
	supabaseURL    = os.Getenv("SUPABASE_URL")
	supabaseAPIKey = os.Getenv("SUPABASE_API_KEY")
)

type StationData struct {
	Data struct {
		Stations []struct {
			StationCode                 string           `json:"stationCode"`
			StationID                   int              `json:"station_id"`
			NumBikesAvailable           int              `json:"numBikesAvailable"`
			NumBikesAvailableTypes      []map[string]int `json:"num_bikes_available_types"`
			NumDocksAvailable           int              `json:"numDocksAvailable"`
			IsInstalled                 int              `json:"is_installed"`
			IsReturning                 int              `json:"is_returning"`
			IsRenting                   int              `json:"is_renting"`
			LastReported                int64            `json:"last_reported"`
			NumMechanicalBikesAvailable int              // Custom field to hold the count of mechanical bikes
			NumEBikesAvailable          int              // Custom field to hold the count of e-bikes
		} `json:"stations"`
	} `json:"data"`
}

func fetchData() (*StationData, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data StationData
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	for i, station := range data.Data.Stations {
		for _, bikeType := range station.NumBikesAvailableTypes {
			for bike, count := range bikeType {
				if bike == "mechanical" {
					data.Data.Stations[i].NumMechanicalBikesAvailable = count
				} else if bike == "ebike" {
					data.Data.Stations[i].NumEBikesAvailable = count
				}
			}
		}
	}
	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)

	return &data, nil
}

func pushToSupabase(data *StationData) error {
	client := &http.Client{}
	for _, station := range data.Data.Stations {
		stationJSON, err := json.Marshal(station)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", supabaseURL+"/"+supabaseTable, bytes.NewBuffer(stationJSON))
		if err != nil {
			return err
		}

		req.Header.Set("apikey", supabaseAPIKey)
		// req.Header.Set("Prefer", "resolution=merge-duplicates")
		req.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to insert data: %s, %s", resp.Status, resp.Body)
		}
	}

	return nil
}

func main() {
	data, err := fetchData()
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return
	}

	err = pushToSupabase(data)
	if err != nil {
		fmt.Println("Error pushing data to Supabase:", err)
	}
}