package steam

import (
	"net/http"
	"net/url"
	"strconv"
	"io/ioutil"
	"encoding/json"
)

type AppNews struct {
	Appnews struct{
		Appid int
		Newsitems []struct{
			Gid string
			Title string
			Url string
			Is_external_url bool
			Author string
			Contents string
			Feedlabel string
			Date int
			Feedname string
		}
		}
}

type GlobalAchievementPercentage struct {
	Achievementpercentages struct{
				       Achievements []struct{
					       Name string
					       Percent float64
				       }
			       }
}

type AppList struct {
	Applist struct{
		Apps struct{
			App []struct{
				Appid int
				Name string
			}
		     }
		}
}

type NumberOfCurrentPlayers struct {
	Response struct{
		Player_count int
		Result int
		 }
}

// GetNewsForApp returns the latest of a game specified by its appid.
// The count parameter specifies how many news items to return. It
// is returned in order from most recent. The maxLength parameter
// is used to specify max length of the news content to be returned.
func GetNewsForApp(appid, count, maxLength int) (*AppNews, error) {
	var news AppNews

	resp, err := http.Get("http://api.steampowered.com/ISteamNews/GetNewsForApp/v0002/?" + url.Values{
		"appid":	{strconv.FormatInt(int64(appid), 10)},
		"count":	{strconv.FormatInt(int64(count), 10)},
		"maxlength":	{strconv.FormatInt(int64(maxLength), 10)},
		"format":	{"json"},
	}.Encode())
	if err != nil {
		return &news, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &news, err
	}

	if err := json.Unmarshal(content, &news); err != nil {
		return &news, err
	}

	return &news, err
}

// GetGlobalAchievementPercentagesForApp returns on global achievements overview of a
// specific appid in percentages.
func GetGlobalAchievementPercentagesForApp(appid int) (*GlobalAchievementPercentage, error) {
	var achievements GlobalAchievementPercentage

	resp, err := http.Get("http://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?" + url.Values{
		"gameid":	{strconv.FormatInt(int64(appid), 10)},
		"format":	{"json"},
	}.Encode())
	if err != nil {
		return &achievements, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &achievements, err
	}

	if err := json.Unmarshal(content, &achievements); err != nil {
		return &achievements, err
	}

	return &achievements, err
}

// GetAppList returns a type AppList
// containing every appid on Steam.
func GetAppList() (*AppList, error) {
	var appList AppList

	resp, err := http.Get("https://api.steampowered.com/ISteamApps/GetAppList/v1")
	if err != nil {
		return &appList, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &appList, err
	}

	if err := json.Unmarshal(content, &appList); err != nil {
		return &appList, err
	}

	return &appList, nil
}

// GetNumberOfCurrentPlayers returns the number of players for a specific
// appid.
func GetNumberOfCurrentPlayers(appid int) (*NumberOfCurrentPlayers, error) {
	var currentPlayers NumberOfCurrentPlayers

	resp, err := http.Get("https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1?" + url.Values{
		"appid": {strconv.FormatInt(int64(appid), 10)},
	}.Encode())
	if err != nil {
		return &currentPlayers, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &currentPlayers, err
	}

	if err := json.Unmarshal(content, &currentPlayers); err != nil {
		return &currentPlayers, err
	}

	return &currentPlayers, nil
}