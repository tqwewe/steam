package steam

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type AppNews []struct {
	Author          string
	Contents        string
	Date            int
	Feedlabel       string
	Feedname        string
	Gid             string
	Is_external_url bool
	Title           string
	Url             string
}

type GlobalAchievementPercentage []struct {
	Name    string
	Percent float64
}

type AppList []struct {
	Appid int
	Name  string
}

type AppInfo struct {
	Appid       int
	Name        string
	Playercount int
}

// GetNewsForApp returns a type AppNews containing all the news for a specific AppID in order from most recent.
// The count parameter specific how many news items to return.
// The maxLength parameter is used to specify how many characters of each news item to show.
// If 0 is used for maxLength then there will be no limit on how many characters to return.
func GetNewsForApp(appid, count, maxLength int) (AppNews, error) {
	var news AppNews

	resp, err := http.Get("http://api.steampowered.com/ISteamNews/GetNewsForApp/v0002/?" + url.Values{
		"appid":     {strconv.FormatInt(int64(appid), 10)},
		"count":     {strconv.FormatInt(int64(count), 10)},
		"maxlength": {strconv.FormatInt(int64(maxLength), 10)},
	}.Encode())
	if err != nil {
		return news, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return news, err
	}

	var newsForAppResponse struct {
		Appnews struct {
			Appid     int
			Newsitems []struct {
				Author          string
				Contents        string
				Date            int
				Feedlabel       string
				Feedname        string
				Gid             string
				Is_external_url bool
				Title           string
				Url             string
			}
		}
	}

	if err := json.Unmarshal(content, &newsForAppResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return news, jsonUnmarshallErrorCheck(content)
		}
		return news, err
	}

	news = newsForAppResponse.Appnews.Newsitems
	return news, nil
}

// GetGlobalAchievementPercentagesForApp returns a type GlobalAchievementPercentage containing all existing achievements
// on the Steam network and their global achieved percentage for an AppID.
func GetGlobalAchievementPercentagesForApp(appid int) (GlobalAchievementPercentage, error) {
	var achievements GlobalAchievementPercentage

	resp, err := http.Get("http://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?" + url.Values{
		"gameid": {strconv.FormatInt(int64(appid), 10)},
	}.Encode())
	if err != nil {
		return achievements, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return achievements, err
	}

	var globalAchievementPercentagesForAppResponse struct {
		Achievementpercentages struct {
			Achievements []struct {
				Name    string
				Percent float64
			}
		}
	}

	if err := json.Unmarshal(content, &globalAchievementPercentagesForAppResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return achievements, jsonUnmarshallErrorCheck(content)
		}
		return achievements, err
	}

	achievements = globalAchievementPercentagesForAppResponse.Achievementpercentages.Achievements
	return achievements, nil
}

// GetAppList returns a type AppList containing all existing AppID's on the Steam network.
func GetAppList() (AppList, error) {
	var appList AppList

	resp, err := http.Get("https://api.steampowered.com/ISteamApps/GetAppList/v1")
	if err != nil {
		return appList, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return appList, err
	}

	var appListResponse struct {
		Applist struct {
			Apps struct {
				App []struct {
					Appid int
					Name  string
				}
			}
		}
	}

	if err := json.Unmarshal(content, &appListResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return appList, jsonUnmarshallErrorCheck(content)
		}
		return appList, err
	}

	appList = appListResponse.Applist.Apps.App
	return appList, nil
}

// GetNumberOfCurrentPlayers returns the number of players which are playing a
// specified AppID open.
func GetNumberOfCurrentPlayers(appid int) (int, error) {
	resp, err := http.Get("https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1?" + url.Values{
		"appid": {strconv.FormatInt(int64(appid), 10)},
	}.Encode())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var numberOfCurrentPlayersResponse struct {
		Response struct {
			Player_count int
			Result       int
		}
	}

	if err := json.Unmarshal(content, &numberOfCurrentPlayersResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return 0, jsonUnmarshallErrorCheck(content)
		}
		return 0, err
	}

	if numberOfCurrentPlayersResponse.Response.Result != 1 {
		return 0, err
	}

	return numberOfCurrentPlayersResponse.Response.Player_count, nil
}

// GetNumberOfCurrentPlayersForAllApps returns the number of players for all existing apps on the Steam network.
// This function may take minutes to complete as it requests ~28000 http requests.
func GetNumberOfCurrentPlayersForAllApps() ([]AppInfo, error) {
	appList, err := GetAppList()
	if err != nil {
		return []AppInfo{}, err
	}

	var que int
	var queMax int = 800
	var apps []AppInfo

	var wg sync.WaitGroup
	for _, a := range appList {
		app := a
		for que > queMax {
			time.Sleep(time.Millisecond * 100)
		}
		que++
		wg.Add(1)
		go func() {
			defer func() {
				que--
				wg.Done()
			}()

			currentPlayers, err := GetNumberOfCurrentPlayers(app.Appid)
			if err != nil {
				return
			}

			apps = append(apps, AppInfo{
				Appid:       app.Appid,
				Name:        app.Name,
				Playercount: currentPlayers,
			})
		}()
	}

	wg.Wait()
	return apps, nil
}
