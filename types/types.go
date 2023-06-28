package types

import "time"

type GithubRelease struct {
	URL       string `json:"url"`
	AssetsURL string `json:"assets_url"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
	ID        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		URL      string `json:"url"`
		ID       int    `json:"id"`
		NodeID   string `json:"node_id"`
		Name     string `json:"name"`
		Label    string `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadURL string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}

type SteamItemResponse struct {
	Success     bool   `json:"success"`
	LowestPrice string `json:"lowest_price"`
	Volume      string `json:"volume"`
	MedianPrice string `json:"median_price"`
}

type SteamAPIResponse struct {
	Result struct {
		App struct {
			Version   int    `json:"version"`
			Timestamp int    `json:"timestamp"`
			Time      string `json:"time"`
		} `json:"app"`
		Services struct {
			SessionsLogon  string `json:"SessionsLogon"`
			SteamCommunity string `json:"SteamCommunity"`
			IEconItems     string `json:"IEconItems"`
			Leaderboards   string `json:"Leaderboards"`
		} `json:"services"`
		Datacenters struct {
			Peru struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Peru"`
			EUWest struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"EU West"`
			EUEast struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"EU East"`
			Poland struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Poland"`
			IndiaEast struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"India East"`
			HongKong struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Hong Kong"`
			Spain struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Spain"`
			Chile struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Chile"`
			USSouthwest struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"US Southwest"`
			USSoutheast struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"US Southeast"`
			India struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"India"`
			EUNorth struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"EU North"`
			Emirates struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Emirates"`
			USNorthwest struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"US Northwest"`
			SouthAfrica struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"South Africa"`
			Brazil struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Brazil"`
			USNortheast struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"US Northeast"`
			USNorthcentral struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"US Northcentral"`
			Japan struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Japan"`
			Argentina struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Argentina"`
			SouthKorea struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"South Korea"`
			Singapore struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Singapore"`
			Australia struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"Australia"`
			ChinaShanghai struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"China Shanghai"`
			ChinaTianjin struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"China Tianjin"`
			ChinaGuangzhou struct {
				Capacity string `json:"capacity"`
				Load     string `json:"load"`
			} `json:"China Guangzhou"`
		} `json:"datacenters"`
		Matchmaking struct {
			Scheduler        string `json:"scheduler"`
			OnlineServers    int    `json:"online_servers"`
			OnlinePlayers    int    `json:"online_players"`
			SearchingPlayers int    `json:"searching_players"`
			SearchSecondsAvg int    `json:"search_seconds_avg"`
		} `json:"matchmaking"`
		Perfectworld struct {
			Logon struct {
				Availability string `json:"availability"`
				Latency      string `json:"latency"`
			} `json:"logon"`
			Purchase struct {
				Availability string `json:"availability"`
				Latency      string `json:"latency"`
			} `json:"purchase"`
		} `json:"perfectworld"`
	} `json:"result"`
}

type SteamInventoryReturn struct {
	Assets []struct {
		Appid      int    `json:"appid"`
		Contextid  string `json:"contextid"`
		Assetid    string `json:"assetid"`
		Classid    string `json:"classid"`
		Instanceid string `json:"instanceid"`
		Amount     string `json:"amount"`
	} `json:"assets"`
	Descriptions []struct {
		Appid           int    `json:"appid"`
		Classid         string `json:"classid"`
		Instanceid      string `json:"instanceid"`
		Currency        int    `json:"currency"`
		BackgroundColor string `json:"background_color"`
		IconURL         string `json:"icon_url"`
		IconURLLarge    string `json:"icon_url_large,omitempty"`
		Descriptions    []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"descriptions"`
		Tradable int `json:"tradable"`
		Actions  []struct {
			Link string `json:"link"`
			Name string `json:"name"`
		} `json:"actions,omitempty"`
		Name           string `json:"name"`
		NameColor      string `json:"name_color"`
		Type           string `json:"type"`
		MarketName     string `json:"market_name"`
		MarketHashName string `json:"market_hash_name"`
		MarketActions  []struct {
			Link string `json:"link"`
			Name string `json:"name"`
		} `json:"market_actions,omitempty"`
		Commodity                 int `json:"commodity"`
		MarketTradableRestriction int `json:"market_tradable_restriction"`
		Marketable                int `json:"marketable"`
		Tags                      []struct {
			Category              string `json:"category"`
			InternalName          string `json:"internal_name"`
			LocalizedCategoryName string `json:"localized_category_name"`
			LocalizedTagName      string `json:"localized_tag_name"`
			Color                 string `json:"color,omitempty"`
		} `json:"tags"`
		Fraudwarnings []string `json:"fraudwarnings,omitempty"`
	} `json:"descriptions"`
	TotalInventoryCount int `json:"total_inventory_count"`
	Success             int `json:"success"`
	Rwgrsn              int `json:"rwgrsn"`
}
