// Steam provides functions to convert Steam ID's to any existing Steam ID
// format, login and send messages to users, and a large number of Steam API
// functions to collect information about Steam.
package steam

import (
	"encoding/json"
	"errors"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Account is a struct containing information about a Steam account including
// the login details and http client.
type Account struct {
	Username    string
	Password    string
	SteamID     SteamID64
	HttpClient  *http.Client
	Umqid       string
	AccessToken string
	ApiKey      string
}

type SteamID string   // STEAM_0:0:86173181
type SteamID64 uint64 // 76561198132612090
type SteamID32 uint32 // 172346362
type SteamID3 string  // [U:1:172346362]
type GroupID uint64   // 103582791453729676

// getSessionId returns the Steam sessionid cookie.
// If no sessionid cookie is found, an empty string will be returned.
func (acc *Account) getSessionId() (string, error) {
	resp, err := acc.HttpClient.Get("https://steamcommunity.com/")
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	sessionid := regexp.MustCompile(`g_sessionID\s\=\s\"(\w+)\"\;`).FindSubmatch(content)
	if sessionid == nil {
		return "", errors.New("No sessionid available")
	}

	return string(sessionid[1]), nil
}

// getAccessToken returns the accesstoken of an Account.
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
// If no accessToken is found then an empty string is returned.
func (acc *Account) getUmqid() string {
	accessToken := acc.getAccessToken()
	if accessToken == "" {
		return ""
	}

	resp, err := acc.HttpClient.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001", url.Values{
		"jsonp":        []string{"1"},
		"ui_mode":      []string{"web"},
		"access_token": []string{accessToken},
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
		Error         string
		Message       int
		Push          int
		Steamid       string
		Timestamp     int64
		Umqid         string
		Utc_timestamp int64
	}
	if err := json.Unmarshal(content, &umqidResponse); err != nil {
		return ""
	}

	return umqidResponse.Umqid
}

// makeTimestamp returns the current Unix timestamp.
func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// stringBetween returns a substring located between the first occurrence of both the provided start and end strings.
// An error will be returned if str does not include both start and end as a substring.
func stringBetween(str, start, end string) (string, error) {
	if strings.Index(str, start) == -1 {
		return str, errors.New("String does not include start as substring.")
	}
	str = str[len(start)+strings.Index(str, start):]
	if strings.Index(str, end) == -1 {
		return str, errors.New("String does not include end as substring.")
	}
	return str[:strings.Index(str, end)], nil
}

// jsonUnmarshallError is used to manage steam errors which are no json and return the message given.
func jsonUnmarshallErrorCheck(content []byte) error {
	var errorPage string
	if strings.Index(strings.ToLower(string(content)), "<body>") == -1 {
		return errors.New(html.UnescapeString(string(content)))
	}
	errorPage = string(content)[len("<body>")+strings.Index(strings.ToLower(string(content)), "<body>"):]
	if strings.Index(strings.ToLower(string(content)), "</body>") == -1 {
		return errors.New(html.UnescapeString(string(content)))
	}
	errorPage = errorPage[:strings.Index(strings.ToLower(errorPage), "</body>")]

	htmlTags := regexp.MustCompile(`<\/?\w+>`).FindAllString(errorPage, -1)
	for _, tag := range htmlTags {
		switch tag {
		case "</h1>":
			errorPage = strings.Replace(errorPage, tag, ": ", -1)
		case "<pre>":
			errorPage = strings.Replace(errorPage, tag, "'", -1)
		case "</pre>":
			errorPage = strings.Replace(errorPage, tag, "'", -1)
		default:
			errorPage = strings.Replace(errorPage, tag, "", -1)
		}
	}

	return errors.New(html.UnescapeString(strings.Replace(errorPage, "\n", " ", -1)))
}
