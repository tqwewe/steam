package steam

import (
	"net/url"
	"io/ioutil"
	"regexp"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"strings"
	"math/big"
)

type Account struct {
	Username string
	Password string
	SteamId int64
	HttpClient *http.Client
	ApiKey string
	Umqid string
	AccessToken string
}

// getSessionId returns the Steam sessionid cookie.
//
// If no sessionid cookie is found, an empty string will be returned.
func (acc *Account) getSessionId() string {
	steamUrl, err := url.Parse("https://steamcommunity.com")
	if err != nil {
		return ""
	}

	cookies := acc.HttpClient.Jar.Cookies(steamUrl)
	for _, cookie := range cookies {
		if cookie.Name == "sessionid" {
			return cookie.Value
		}
	}

	return ""
}

// getAccessToken returns the accesstoken of an Account.
//
// If no accesstoken is found then an empty string is returned.
func (acc *Account) getAccessToken() string {
	resp, err := acc.HttpClient.Get("https://steamcommunity.com/chat")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	tokenSlice := regexp.MustCompile(`CWebAPI\s*\(\s*(?:[^,]+,){2}\s*"([0-9a-f]{32})"\s*\)`).FindSubmatch(content)
	if len(tokenSlice) < 2 {
		return ""
	}

	return string(tokenSlice[1])
}

// getUmqid returns the umqid of an Account.
//
// If no accessToken is found then an empty string is returned.
func (acc *Account) getUmqid() string {
	accessToken := acc.getAccessToken()
	if accessToken == "" {
		return ""
	}

	resp, err := acc.HttpClient.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001", url.Values {
		"jsonp":	[]string{"1"},
		"ui_mode":	[]string{"web"},
		"access_token":	[]string{accessToken},
	})
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var umqidResponse struct {
		Error		string
		Message		int
		Push		int
		Steamid		string
		Timestamp	int64
		Umqid		string
		Utc_timestamp	int64
	}
	if err := json.Unmarshal(content, &umqidResponse); err != nil {
		return ""
	}

	return umqidResponse.Umqid
}

// apiKeyCheck returns a bool indicating weather the Account has an APIKey set.
func (acc *Account) apiKeyCheck() bool {
	if acc.ApiKey != "" {
		return true
	}
	return false
}

// TODO Rename the steamid functions correctly
// Steamid64To32 converts a given steam id
// formatted in 64 bit to 32 bit form.
func Steamid64To32(steamid int64) (int, error) {
	steamid32, err := strconv.ParseInt(strconv.FormatInt(steamid, 10)[3:], 10, 64)
	if err != nil {
		return 0, err
	}
	return int(steamid32 - 61197960265728), nil
}

// Steamid32To64 converts a given steam id
// formatted in 32 bit to 64 bit form.
func Steamid32To64(steamid int) (int64, error) {
	steamid64, err := strconv.ParseInt("765" + strconv.FormatInt(int64(steamid) + 61197960265728, 10), 10, 64)
	if err != nil {
		return 0, err
	}
	return steamid64, nil
}

// SteamidTo64 converts a given regular steam id to 64 bit.
// E.g. STEAM_0:0:86173181 -> 76561198132612090
func SteamidTo64(steamid string) int64 {
	p := strings.Split(steamid, ":")
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steam64, _ := new(big.Int).SetString(p[2], 10)
	steam64 = steam64.Mul(steam64, big.NewInt(2))
	steam64 = steam64.Add(steam64, magic)
	auth, _ := new(big.Int).SetString(p[1], 10)
	return steam64.Add(steam64, auth).Int64()
}

// Steamid64ToSteamid converts a given steam id 64 bit
// to regular steam id.
// E.g. 76561198132612090 -> STEAM_0:0:86173181
func Steamid64ToSteamid(steamid int64) string {
	id := new(big.Int).SetInt64(steamid)
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	id = id.Sub(id, magic)
	isServer := new(big.Int).And(id, big.NewInt(1))
	id = id.Sub(id, isServer)
	id = id.Div(id, big.NewInt(2))
	return "STEAM_0:" + isServer.String() + ":" + id.String()
}

// makeTimestamp returns the current Unix timestamp.
func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}