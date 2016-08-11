package steam

import (
	"net/url"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"strings"
	"errors"
	"fmt"
	"sync"
	"net/http"
	"regexp"
)

type PlayerAchievements []struct {
	Achieved	int
	Apiname		string
}

// Message sends a message to a specified steamid using a logged in Account.
func (acc *Account) Message(recipient int64, message string) error {
	if len(acc.Umqid) <= 0 {
		if umqid := acc.getUmqid(); umqid == "" {
			return errors.New("unable to retrieve umqid")
		} else {
			acc.Umqid = umqid
		}
	}
	if len(acc.AccessToken) <= 0 {
		if accessToken := acc.getAccessToken(); accessToken == "" {
			return errors.New("unable to retrieve umqid")
		} else {
			acc.AccessToken = accessToken
		}
	}

	resp, err := acc.HttpClient.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Message/v0001/", url.Values{
		"steamid_dst":		{strconv.FormatInt(recipient, 10)},
		"text":			{message},
		"umqid":		{acc.Umqid},
		"access_token":		{acc.AccessToken},
		"type":			{"saytext"},
		"jsonp":		{"1"},
		"_":			{strconv.FormatInt(makeTimestamp(), 10)},

	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var messageResponse struct {
		Utc_timestamp int64
		Error string
	}
	if err := json.Unmarshal(content, &messageResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	if strings.ToLower(messageResponse.Error) != "ok" {
		return errors.New(messageResponse.Error)
	}

	return nil
}

// TODO: Retrieve friends without using API key
// Broadcast sends a specified message to all
// steamid's for Account
func (acc *Account) Broadcast(message string) error {
	if !acc.apiKeyCheck() {
		return errors.New("missing API key")
	}
	resp, err := acc.HttpClient.Get("http://api.steampowered.com/ISteamUser/GetFriendList/v0001?" + url.Values{
		"key":		{acc.ApiKey},
		"steamid":	{strconv.Itoa(int(acc.SteamId))},
		"relationship": {"friend"},
	}.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var friendsResponse struct{
		Friendslist struct{
			Friends []struct{
				Steamid string
				Relationship string
				Friend_since int64
			}
			    }
	}
	if err = json.Unmarshal(content, &friendsResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	var wg sync.WaitGroup

	for _, friend := range friendsResponse.Friendslist.Friends {
		if steamId, err := strconv.Atoi(friend.Steamid); err == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = acc.Message(int64(steamId), message)
				if err != nil {
					fmt.Println(steamId, err)
				}
			}()
		}
	}

	wg.Wait()
	return nil
}

// TODO: Fix issue with user logging out
// ListenAndServe stops execution and loops listening to messages from other Steam
// users. When a message is received, the argument callback is called.
func (acc *Account) ListenAndServe(callback func(user int64, message string)) error {
	if umqid := acc.getUmqid(); umqid == "" {
		return errors.New("unable to retrieve umqid")
	} else {
		acc.Umqid = umqid
	}
	if accessToken := acc.getAccessToken(); accessToken == "" {
		return errors.New("unable to retrieve accessToken")
	} else {
		acc.AccessToken = accessToken
	}

	resp, err := acc.HttpClient.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001/?" + url.Values{
		"jsonp":	{"1"},
		"ui_mode":	{"web"},
		"access_token":	{acc.AccessToken},
		"_":		{strconv.FormatInt(makeTimestamp(), 10)},
	}.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	content = []byte(string(content)[strings.Index(string(content), `{`):len(string(content))-1])

	var logonResponse struct{
		Steamid		string
		Error		string
		Umqid		string
		Timestamp	int64
		Utc_timestamp	int64
		Message		int
		Push		int
	}
	if err = json.Unmarshal(content, &logonResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	if logonResponse.Error != "OK" {
		return errors.New(logonResponse.Error)
	}

	acc.Umqid = logonResponse.Umqid
	if steamid, err := strconv.ParseInt(logonResponse.Steamid, 10, 64); err == nil {
		acc.SteamId = steamid
	}
	var pollid int64 = 1
	var message int64 = int64(logonResponse.Message)

	for {
		resp, err = acc.HttpClient.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Poll/v0001/?" + url.Values{
			"jsonp":		{"1"},
			"umqid":		{acc.Umqid},
			"message":		{strconv.FormatInt(message, 10)},
			"pollid":		{strconv.FormatInt(pollid, 10)},
			"sectimeout":		{"25"},
			"secidletime":		{"12"},
			"use_accountids":	{"1"},
			"access_token":		{acc.AccessToken},
			"_":			{strconv.FormatInt(makeTimestamp(), 10)},
		}.Encode())
		if err != nil {
			return err
		}

		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		content = []byte(string(content)[strings.Index(string(content), `{`):len(string(content))-1])

		resp.Body.Close()

		var pollResponse struct{
			Pollid int64
			Sectimeout int64
			Error string
			Messages []struct{
				Type string
				Timestamp int64
				Utc_timestamp int64
				Accountid_from int64
				Text string
				Status_flags int
				Persona_state int
				Persona_name string
			}
			Messagelast int
			Timestamp int64
			Utc_timestamp int64
			Messagebase int
		}
		if err = json.Unmarshal(content, &pollResponse); err != nil {
			if err.Error() == "invalid character '<' looking for beginning of value" {
				return jsonUnmarshallErrorCheck(content)
			}
			return err
		}

		if pollResponse.Error != "OK" && pollResponse.Error != "Timeout" {
			return errors.New(pollResponse.Error)
		}

		for _, message := range pollResponse.Messages {
			if message.Type == "saytext" && len(message.Text) > 0 {
				steamid, _ := Steamid32To64(int(message.Accountid_from))
				callback(steamid, message.Text)
			}
		}

		pollid = pollResponse.Pollid + 1
		message = int64(pollResponse.Messagelast)
	}
	return nil
}

// SearchForID tries to retrieve a Steamid64 using a query (search).
//
// If an error occurs or the steamid was unable to be resolved from the query then a 0 is returned.
func SearchForID(query, apikey string) int64 {
	query = strings.Replace(query, " ", "", -1)

	if strings.Index(query, "steamcommunity.com/profiles/") != -1 {
		if string(query[len(query)-1]) == "/" {
			query = query[0:len(query)-1]
		}

		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/") + len("steamcommunity.com/profiles/"):], 10, 64)
		if err != nil {
			return 0
		}

		query = strings.Replace(query, "/", "", -1)

		if len(strconv.FormatInt(output, 10)) != 17 {
			return 0
		}

		return output
	} else if strings.Index(query, "steamcommunity.com/id/") != -1 {
		if string(query[len(query)-1]) == "/" {
			query = query[0:len(query)-1]
		}

		query = query[strings.Index(query, "steamcommunity.com/id/") + len("steamcommunity.com/id/"):]

		resp, err := http.Get("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + url.Values{
			"key":		{apikey},
			"vanityurl":	{query},
		}.Encode())
		if err != nil {
			return 0
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0
		}

		var vanityUrlResponse struct{
			Response struct{
				Steamid string
				Success int
				 }
		}

		if err := json.Unmarshal(content, &vanityUrlResponse); err != nil {
			return 0
		}

		if vanityUrlResponse.Response.Success != 1 {
			return 0
		}

		if len(vanityUrlResponse.Response.Steamid) != 17 {
			return 0
		}

		output, err := strconv.ParseInt(vanityUrlResponse.Response.Steamid, 10, 64)
		if err != nil {
			return 0
		}

		return output
	} else if regexp.MustCompile(`^STEAM_0:(0|1):[0-9]{1}[0-9]{0,8}$`).Match([]byte(query)) {
		steamid := SteamidTo64(query)

		if len(strconv.FormatInt(steamid, 10)) != 17 {
			return 0
		}

		return steamid
	} else if regexp.MustCompile(`^\d+$`).Match([]byte(query)) && len(query) == 17 {
		output, err := strconv.ParseInt(query, 10, 64)
		if err != nil {
			return 0
		}

		if len(strconv.FormatInt(output, 10)) != 17 {
			return 0
		}

		return output
	}

	resp, err := http.Get("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + url.Values{
		"key":		{apikey},
		"vanityurl":	{query},
	}.Encode())
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0
	}

	var vanityUrlResponse struct{
		Response struct{
				 Steamid string
				 Success int
			 }
	}

	if err := json.Unmarshal(content, &vanityUrlResponse); err != nil {
		return 0
	}

	if vanityUrlResponse.Response.Success != 1 {
		return 0
	}

	if len(vanityUrlResponse.Response.Steamid) != 17 {
		return 0
	}

	output, err := strconv.ParseInt(vanityUrlResponse.Response.Steamid, 10, 64)
	if err != nil {
		return 0
	}

	return output
}

// GetPlayerAchievements returns a type PlayerAchievements containing all achievements achieved by a specified steamid.
//
// If an error occurs then an empty PlayerAchievements will be returned along with the error.
func GetPlayerAchievements(steamid int64, appid int, apikey string) (PlayerAchievements, error) {
	var plyAchievements PlayerAchievements

	resp, err := http.Get("https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1?" + url.Values{
		"steamid":	{strconv.FormatInt(steamid, 10)},
		"appid":	{strconv.FormatInt(int64(appid), 10)},
		"key":		{apikey},
	}.Encode())
	if err != nil {
		return plyAchievements, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return plyAchievements, err
	}

	var playerAchievementsResponse struct {
		Playerstats struct{
				    SteamID string
				    GameName string
				    Success bool
				    Achievements []struct {
					    Achieved int
					    Apiname string
				    }
			    }
	}

	if err := json.Unmarshal(content, &playerAchievementsResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return plyAchievements, jsonUnmarshallErrorCheck(content)
		}
		return plyAchievements, err
	}

	if playerAchievementsResponse.Playerstats.Success != true {
		return plyAchievements, err
	}

	plyAchievements = playerAchievementsResponse.Playerstats.Achievements
	return plyAchievements, nil
}