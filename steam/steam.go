package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/system"
	"github.com/devusSs/steamquery-v2/types"
)

type steamStatus int

const (
	statusAPIURL = "https://api.steampowered.com/ICSGOServers_730/GetGameServersStatus/v1/?key="

	steamNormal  steamStatus = iota
	steamDelayed steamStatus = iota
	steamDown    steamStatus = iota
)

// Actual check on the Steam API for status of CSGO servers.
func IsSteamCSGOAPIUp(apiKey string) (bool, error) {
	startTime := time.Now()

	logging.LogInfo("Fetching Steam API status, please wait")

	res, err := http.Get(fmt.Sprintf("%s%s", statusAPIURL, apiKey))
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf(
			"unwanted Steam status response: %s (code: %d)",
			res.Status,
			res.StatusCode,
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	system.BytesUsed += len(body)

	var resp types.SteamAPIResponse

	if err := json.Unmarshal(body, &resp); err != nil {
		return false, err
	}

	var steamStatusSessions steamStatus
	var steamStatusCommunity steamStatus

	switch resp.Result.Services.SessionsLogon {
	case "normal":
		steamStatusSessions = steamNormal
	case "delayed":
		steamStatusSessions = steamDelayed
		logging.LogWarning("Steam sessions logon delayed, expect problems")
	default:
		steamStatusSessions = steamDown
	}

	switch resp.Result.Services.SteamCommunity {
	case "normal":
		steamStatusCommunity = steamNormal
	case "delayed":
		steamStatusCommunity = steamDelayed
		logging.LogWarning("Steam community delayed, expect problems")
	default:
		steamStatusCommunity = steamDown
	}

	logging.LogDebug(fmt.Sprintf("took %.2f second(s)", time.Since(startTime).Seconds()))

	return steamStatusSessions < 3 && steamStatusCommunity < 3, nil
}

func GetAndCompareSteamInventory(
	apiKey string, steamID64 uint64,
	itemListMap map[string]int,
	itemAmountMap map[int]int,
) (map[string]int, error) {
	startTime := time.Now()

	steamUp, err := IsSteamCSGOAPIUp(apiKey)
	if err != nil {
		return nil, err
	}

	if !steamUp {
		return nil, errors.New("steam down, retry later")
	}

	logging.LogSuccess("Steam is up and running")

	logging.LogInfo("Fetching Steam CSGO inventory, please wait")

	logging.LogWarning("NOTE: this will not work for storage units")

	// This function already only fetches marketable items, no need to remove anything.
	inventoryMap, err := getSteamInventory(steamID64)
	if err != nil {
		return nil, err
	}

	logging.LogSuccess("Successfully fetched Steam CSGO inventory")

	logging.LogInfo("Mapping amounts to name from sheets")

	itemNameAmountMap := make(map[string]int)

	for item, cell := range itemListMap {
		if item == "" {
			continue
		}

		if strings.Contains(item, "empty_cell") {
			continue
		}

		amount, ok := itemAmountMap[cell]
		if !ok {
			return nil, fmt.Errorf("missing amount for item %s", item)
		}

		itemNameAmountMap[item] = amount
	}

	logging.LogSuccess("Done mapping amounts to name from sheets")

	logging.LogDebug(fmt.Sprintf("ITEM NAME AMOUNT MAP: %v", itemNameAmountMap))
	logging.LogDebug(fmt.Sprintf("INVENTORY MAP: %v", inventoryMap))

	missingAddMap := make(map[string]int)

	logging.LogInfo("Comparing Steam inventory with provided sheets list now, please wait")

	for item, amount := range itemNameAmountMap {
		amountInInv, ok := inventoryMap[item]
		if !ok {
			missingAddMap[item] = amount
		}

		if amountInInv > amount {
			missingAddMap[item] = amountInInv
		}

		if amountInInv < amount {
			missingAddMap[item] = amount
		}
	}

	for item, amount := range inventoryMap {
		_, ok := itemNameAmountMap[item]
		if !ok {
			missingAddMap[item] = amount
		}
	}

	logging.LogSuccess(
		"Successfully compared lists and added missing",
	)

	logging.LogDebug(fmt.Sprintf("MISSING ITEM MAP: %v", missingAddMap))

	logging.LogDebug(fmt.Sprintf("took %.2f second(s)", time.Since(startTime).Seconds()))

	return missingAddMap, nil
}

func getSteamInventory(steamID64 uint64) (map[string]int, error) {
	startTime := time.Now()

	url := fmt.Sprintf("http://steamcommunity.com/inventory/%d/%d/%d", steamID64, 730, 2)

	client := http.Client{}
	client.Timeout = 2 * time.Second

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"got unwanted Steam response status: %s (code: %d)",
			res.Status,
			res.StatusCode,
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	system.BytesUsed += len(body)

	var steamReturn types.SteamInventoryReturn

	if err := json.Unmarshal(body, &steamReturn); err != nil {
		return nil, err
	}

	itemCountMap := make(map[string]int)

	for _, item := range steamReturn.Descriptions {
		if item.Marketable == 1 {
			itemCountMap[item.Name]++
		}
	}

	logging.LogDebug(fmt.Sprintf("took %.2f second(s)", time.Since(startTime).Seconds()))

	return itemCountMap, nil
}
