package steam

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/devusSs/steamquery-v2/logging"
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

	var resp types.SteamAPIResponse

	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return false, err
	}

	var steamStatusSessions steamStatus
	var steamStatusCommunity steamStatus

	// TODO: this always defaults, why?
	switch resp.Result.Services.SessionsLogon {
	case "normal":
		steamStatusSessions = steamNormal
	case "delayed":
		steamStatusSessions = steamDelayed
		logging.LogWarning("Steam sessions logon delayed, expect problems")
	default:
		steamStatusSessions = steamDown
	}

	// TODO: this always defaults, why?
	switch resp.Result.Services.SteamCommunity {
	case "normal":
		steamStatusCommunity = steamNormal
	case "delayed":
		steamStatusCommunity = steamDelayed
		logging.LogWarning("Steam community delayed, expect problems")
	default:
		steamStatusCommunity = steamDown
	}

	return steamStatusSessions < 3 && steamStatusCommunity < 3, nil
}
