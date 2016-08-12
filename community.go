package steam

import (
	"net/url"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"strings"
	"errors"
	"sync"
	"net/http"
	"regexp"
	"fmt"
)

type PlayerAchievements []struct {
	Achieved	int
	Apiname		string
}

// Message sends a message to a specified steamid using a logged in Account.
func (acc *Account) Message(recipient SteamID64, message string) error {
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
		"steamid_dst":		{strconv.FormatUint(uint64(recipient), 10)},
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

// Broadcast sends a specified message to all SteamID's for Account.
func (acc *Account) Broadcast(message string) error {
	resp, err := acc.HttpClient.Get("http://steamcommunity.com/profiles/76561198193537875/friends/?xml=1")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	friends := make([]SteamID64, 0, 32)

	friendResponse := regexp.MustCompile(`<friend>(.*?)<\/friend>`).FindAllSubmatch(content, -1)
	for _, friendTag := range friendResponse {
		if len(friendTag) >= 2 {
			parsedFriend, err := strconv.ParseUint(string(friendTag[1]), 10, 64)
			if err != nil {
				continue
			}
			friends = append(friends, SteamID64(parsedFriend))
		}
	}

	var wg sync.WaitGroup

	for _, friend := range friends {
		wg.Add(1)
		go func(friendID SteamID64) {
			defer wg.Done()
			err = acc.Message(friendID, message)
			if err != nil {
				fmt.Println(friendID, err)
			}
		}(friend)
	}

	wg.Wait()
	return nil
}

// InviteToGroup invited a set of SteamID64's to a Steam group.
func (acc *Account) InviteToGroup(groupID GroupID, recipients ...SteamID64) error {
	sessionID, err := acc.getSessionId()
	if err != nil {
		return err
	}

	inviteeList := `[`
	for i, steam64 := range recipients {
		inviteeList += `"` + strconv.FormatUint(uint64(steam64), 10) + `"`
		if i < len(recipients)-1 {
			inviteeList += ","
		}
	}
	inviteeList += `]`

	resp, err := acc.HttpClient.PostForm("http://steamcommunity.com/actions/GroupInvite", url.Values{
		"json":		{"1"},
		"type":		{"groupInvite"},
		"group":	{strconv.FormatUint(uint64(groupID), 10)},
		"sessionID":	{sessionID},
		"invitee_list":	{inviteeList},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(content) == "" || string(content) == "null" {
		errors.New("Failed to invite user(s) to group")
	}

	var groupInviteResponse struct{
		Duplicate bool
		GroupId string
		Results string
	}

	if err := json.Unmarshal(content, &groupInviteResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	if groupInviteResponse.Results != "OK" {
		return errors.New("Error: " + groupInviteResponse.Results)
	}

	return nil
}

// ResolveGroupID tried to resolve the GroupID64 from a group custom url.
func ResolveGroupID(groupVanityURL string) (GroupID, error) {
	resp, err := http.Get("http://steamcommunity.com/groups/" + groupVanityURL + "/memberslistxml?xml=1")
	if err != nil {
		return GroupID(0), err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupID(0), err
	}

	groupIDTags := regexp.MustCompile(`<groupID64>(\w+)<\/groupID64>`).FindSubmatch(content)
	if len(groupIDTags) >= 2 {
		groupid, err := strconv.ParseUint(string(groupIDTags[1]), 10, 64)
		if err != nil {
			return GroupID(0), err
		}
		return GroupID(groupid), nil
	}

	return GroupID(0), errors.New("Unable to resolve groupid")
}

// TODO: Fix issue with user logging out
// ListenAndServe stops execution and loops listening to messages from other Steam
// users. When a message is received, the argument callback is called.
func (acc *Account) ListenAndServe(callback func(user SteamID64, message string)) error {
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
		acc.SteamID = SteamID64(steamid)
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
				steam64 := SteamID32ToSteamID64(SteamID32(message.Accountid_from))
				callback(SteamID64(steam64), message.Text)
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
func SearchForID(query, apikey string) SteamID64 {
	query = strings.Replace(query, " ", "", -1)

	if strings.Index(query, "steamcommunity.com/profiles/") != -1 {
		if string(query[len(query)-1]) == "/" {
			query = query[0:len(query)-1]
		}

		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/") + len("steamcommunity.com/profiles/"):], 10, 64)
		if err != nil {
			return SteamID64(0)
		}

		query = strings.Replace(query, "/", "", -1)

		if len(strconv.FormatInt(output, 10)) != 17 {
			return SteamID64(0)
		}

		return SteamID64(output)
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
			return SteamID64(0)
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return SteamID64(0)
		}

		var vanityUrlResponse struct{
			Response struct{
				Steamid string
				Success int
				 }
		}

		if err := json.Unmarshal(content, &vanityUrlResponse); err != nil {
			return SteamID64(0)
		}

		if vanityUrlResponse.Response.Success != 1 {
			return SteamID64(0)
		}

		if len(vanityUrlResponse.Response.Steamid) != 17 {
			return SteamID64(0)
		}

		output, err := strconv.ParseInt(vanityUrlResponse.Response.Steamid, 10, 64)
		if err != nil {
			return SteamID64(0)
		}

		return SteamID64(output)
	} else if regexp.MustCompile(`^STEAM_0:(0|1):[0-9]{1}[0-9]{0,8}$`).Match([]byte(query)) {
		steam64 := SteamIDToSteamID64(SteamID(query))

		if len(strconv.FormatUint(uint64(steam64), 10)) != 17 {
			return SteamID64(0)
		}

		return SteamID64(steam64)
	} else if regexp.MustCompile(`^\d+$`).Match([]byte(query)) && len(query) == 17 {
		output, err := strconv.ParseInt(query, 10, 64)
		if err != nil {
			return SteamID64(0)
		}

		if len(strconv.FormatInt(output, 10)) != 17 {
			return SteamID64(0)
		}

		return SteamID64(output)
	}

	resp, err := http.Get("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + url.Values{
		"key":		{apikey},
		"vanityurl":	{query},
	}.Encode())
	if err != nil {
		return SteamID64(0)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SteamID64(0)
	}

	var vanityUrlResponse struct{
		Response struct{
				 Steamid string
				 Success int
			 }
	}

	if err := json.Unmarshal(content, &vanityUrlResponse); err != nil {
		return SteamID64(0)
	}

	if vanityUrlResponse.Response.Success != 1 {
		return SteamID64(0)
	}

	if len(vanityUrlResponse.Response.Steamid) != 17 {
		return SteamID64(0)
	}

	output, err := strconv.ParseInt(vanityUrlResponse.Response.Steamid, 10, 64)
	if err != nil {
		return SteamID64(0)
	}

	return SteamID64(output)
}

// GetPlayerAchievements returns a type PlayerAchievements containing all achievements achieved by a specified steamid.
//
// If an error occurs then an empty PlayerAchievements will be returned along with the error.
func GetPlayerAchievements(steam64 SteamID64, appid int, apikey string) (PlayerAchievements, error) {
	var plyAchievements PlayerAchievements

	resp, err := http.Get("https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1?" + url.Values{
		"steamid":	{strconv.FormatUint(uint64(steam64), 10)},
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