package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// A PlayerSummaries stores all general profile information for a steam user.
type PlayerSummaries struct {
	SteamID64      SteamID64
	DisplayName    string
	ProfileURL     string
	AvatarSmallURL string
	AvatarMedURL   string
	AvatarFullURL  string
	State          int // 0 - Offline, 1 - Online, 2 - Busy, 3 - Away, 4 - Snooze, 5 - looking to trade, 6 - looking to play
	Public         bool
	Configured     bool
	LastLogOff     int64

	RealName           string
	PrimaryGroupID     GroupID
	TimeCreated        int64
	CurrentlyPlayingID int
	CurrentlyPlaying   string
	ServerIP           string
	CountryCode        string
}

// PlayerAchievements holds a slice of achievements and stores weather the related player has achieved each achievement.
type PlayerAchievements []struct {
	Achieved        bool
	AchievementName string
}

// FriendsList stores a slice storing a specific user's friend's list.
type FriendsList []struct {
	SteamID     SteamID64
	FriendSince int64
}

// Message sends a message to a specified SteamID64 using a logged in Account.
func (acc *Account) Message(recipient SteamID64, message string) error {
	if len(acc.Umqid) <= 0 {
		umqid := acc.getUmqid()
		if umqid == "" {
			return errors.New("unable to retrieve umqid")
		}

		acc.Umqid = umqid
	}
	if len(acc.AccessToken) <= 0 {
		accessToken := acc.getAccessToken()
		if accessToken == "" {
			return errors.New("unable to retrieve umqid")
		}

		acc.AccessToken = accessToken
	}

	resp, err := acc.HttpClient.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Message/v0001/", url.Values{
		"steamid_dst":  {strconv.FormatUint(uint64(recipient), 10)},
		"text":         {message},
		"umqid":        {acc.Umqid},
		"access_token": {acc.AccessToken},
		"type":         {"saytext"},
		"jsonp":        {"1"},
		"_":            {strconv.FormatInt(makeTimestamp(), 10)},
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
	resp, err := acc.HttpClient.Get("http://steamcommunity.com/profiles/" + strconv.FormatUint(uint64(acc.SteamID), 10) + "/friends")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	friends := make([]SteamID64, 0, 32)

	friendResponse := regexp.MustCompile(`name="friends\[(\d+)\]"`).FindAllSubmatch(content, -1)
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

// InviteToGroup invites a set of SteamID64's to a Steam group.
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
		"json":         {"1"},
		"type":         {"groupInvite"},
		"group":        {strconv.FormatUint(uint64(groupID), 10)},
		"sessionID":    {sessionID},
		"invitee_list": {inviteeList},
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
		return errors.New("Failed to invite user(s) to group")
	}

	var groupInviteResponse struct {
		Duplicate bool
		GroupId   string
		Results   string
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

// ResolveGroupID tried to resolve the GroupID from a group custom url.
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

// StateToString converts a profile state (offline/online/looking to play, etc)
// to the correct string. If a number that is not between 0 and 6 is parsed as
// the argument, then an empty string is returned.
func StateToString(state int) string {
	var stateStr string

	switch state {
	case 0:
		stateStr = "Offline"

	case 1:
		stateStr = "Online"

	case 2:
		stateStr = "Busy"

	case 3:
		stateStr = "Away"

	case 4:
		stateStr = "Snooze"

	case 5:
		stateStr = "Looking to Trade"

	case 6:
		stateStr = "Looking to Play"
	}

	return stateStr
}

// ListenAndServe stops execution and loops listening to messages from other Steam
// users. When a message is received, the argument callback is called.
func (acc *Account) ListenAndServe(callback func(user SteamID64, message string)) error {
	umqid := acc.getUmqid()
	if umqid == "" {
		return errors.New("unable to retrieve umqid")
	}

	acc.Umqid = umqid

	accessToken := acc.getAccessToken()
	if accessToken == "" {
		return errors.New("unable to retrieve accessToken")
	}

	acc.AccessToken = accessToken

	resp, err := acc.HttpClient.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001/?" + url.Values{
		"jsonp":        {"1"},
		"ui_mode":      {"web"},
		"access_token": {acc.AccessToken},
		"_":            {strconv.FormatInt(makeTimestamp(), 10)},
	}.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	content = []byte(string(content)[strings.Index(string(content), `{`) : len(string(content))-1])

	var logonResponse struct {
		Steamid       string
		Error         string
		Umqid         string
		Timestamp     int64
		Utc_timestamp int64
		Message       int
		Push          int
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

	steamid, err := strconv.ParseInt(logonResponse.Steamid, 10, 64)
	if err == nil {
		acc.SteamID = SteamID64(steamid)
	}
	var pollid int64 = 1
	var secttimeout int64 = 20
	message := int64(logonResponse.Message)

	for {
		resp, err = acc.HttpClient.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Poll/v0001/?" + url.Values{
			"jsonp":          {"1"},
			"umqid":          {acc.Umqid},
			"message":        {strconv.FormatInt(message, 10)},
			"pollid":         {strconv.FormatInt(pollid, 10)},
			"sectimeout":     {strconv.FormatInt(secttimeout, 10)},
			"secidletime":    {"0"},
			"use_accountids": {"1"},
			"access_token":   {acc.AccessToken},
			"_":              {strconv.FormatInt(makeTimestamp(), 10)},
		}.Encode())
		if err != nil {
			return err
		}

		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		content = []byte(string(content)[strings.Index(string(content), `{`) : len(string(content))-1])

		resp.Body.Close()

		var pollResponse struct {
			Pollid     int64
			Sectimeout int64
			Error      string
			Messages   []struct {
				Type           string
				Timestamp      int64
				Utc_timestamp  int64
				Accountid_from int64
				Text           string
				Status_flags   int
				Persona_state  int
				Persona_name   string
			}
			Messagelast   int
			Timestamp     int64
			Utc_timestamp int64
			Messagebase   int
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

		if pollResponse.Error == "Timeout" {
			if pollResponse.Sectimeout > 20 {
				secttimeout = pollResponse.Sectimeout
			}

			if pollResponse.Sectimeout < 120 {
				if secttimeout+5 < 120 {
					secttimeout = secttimeout + 5
				} else {
					secttimeout = 120
				}
			}
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
}

// SearchForID tries to retrieve a SteamID64 using a query (search).
//
// If an error occurs or the SteamID was unable to be resolved from the query then a 0 is returned.
func SearchForID(query, apikey string) SteamID64 {
	query = strings.Replace(query, " ", "", -1)

	if strings.Index(query, "steamcommunity.com/profiles/") != -1 {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}

		output, err := strconv.ParseInt(query[strings.Index(query, "steamcommunity.com/profiles/")+len("steamcommunity.com/profiles/"):], 10, 64)
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
			query = query[0 : len(query)-1]
		}

		query = query[strings.Index(query, "steamcommunity.com/id/")+len("steamcommunity.com/id/"):]

		resp, err := http.Get("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + url.Values{
			"key":       {apikey},
			"vanityurl": {query},
		}.Encode())
		if err != nil {
			return SteamID64(0)
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return SteamID64(0)
		}

		var vanityUrlResponse struct {
			Response struct {
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
	} else if regexp.MustCompile(`^STEAM_0:(0|1):[0-9]{1}[0-9]{0,8}$`).MatchString(query) {
		steam64 := SteamIDToSteamID64(SteamID(query))

		if len(strconv.FormatUint(uint64(steam64), 10)) != 17 {
			return SteamID64(0)
		}

		return SteamID64(steam64)
	} else if regexp.MustCompile(`^\d+$`).MatchString(query) && len(query) == 17 {
		output, err := strconv.ParseInt(query, 10, 64)
		if err != nil {
			return SteamID64(0)
		}

		if len(strconv.FormatInt(output, 10)) != 17 {
			return SteamID64(0)
		}

		return SteamID64(output)
	} else if regexp.MustCompile(`(\[)?U:1:\d+(\])?`).MatchString(strings.ToUpper(query)) {
		return SteamID3ToSteamID64(SteamID3(query))
	}

	resp, err := http.Get("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + url.Values{
		"key":       {apikey},
		"vanityurl": {query},
	}.Encode())
	if err != nil {
		return SteamID64(0)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SteamID64(0)
	}

	var vanityUrlResponse struct {
		Response struct {
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

// GetPlayerAchievements returns a type PlayerAchievements containing all achievements achieved by a specified SteamID64.
func GetPlayerAchievements(steam64 SteamID64, appid int, apikey string) (PlayerAchievements, error) {
	var plyAchievements PlayerAchievements

	resp, err := http.Get("https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1?" + url.Values{
		"steamid": {strconv.FormatUint(uint64(steam64), 10)},
		"appid":   {strconv.FormatInt(int64(appid), 10)},
		"key":     {apikey},
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
		Playerstats struct {
			SteamID      string
			GameName     string
			Success      bool
			Achievements []struct {
				Achieved int
				Apiname  string
			}
		}
	}

	if err = json.Unmarshal(content, &playerAchievementsResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return plyAchievements, jsonUnmarshallErrorCheck(content)
		}
		return plyAchievements, err
	}

	if playerAchievementsResponse.Playerstats.Success != true {
		return plyAchievements, err
	}

	for _, achievement := range playerAchievementsResponse.Playerstats.Achievements {
		var achievementDetails struct {
			Achieved        bool
			AchievementName string
		}
		if achievement.Achieved > 0 {
			achievementDetails.Achieved = true
		}
		achievementDetails.AchievementName = achievement.Apiname
		plyAchievements = append(plyAchievements, achievementDetails)
	}

	return plyAchievements, nil
}

// GetPlayersSummaries returns a slice of PlayerSummaries with the same length of how many valid SteamID64's were parsed
// as arguments.
func GetPlayersSummaries(apiKey string, steam64 ...SteamID64) ([]PlayerSummaries, error) {
	var plySummaries []PlayerSummaries

	var steamIDs string
	for i, id := range steam64 {
		steamIDs += `"` + strconv.FormatUint(uint64(id), 10) + `"`
		if i < len(steam64)-1 {
			steamIDs += ","
		}
	}

	resp, err := http.Get("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?" + url.Values{
		"steamids": {steamIDs},
		"key":      {apiKey},
	}.Encode())
	if err != nil {
		return plySummaries, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return plySummaries, err
	}

	var playerSummariesResponse struct {
		Response struct {
			Players []struct {
				Avatar                   string `json:"avatar"`
				Avatarfull               string `json:"avatarfull"`
				Avatarmedium             string `json:"avatarmedium"`
				Communityvisibilitystate int    `json:"communityvisibilitystate"`
				Gameextrainfo            string `json:"gameextrainfo"`
				Gameid                   string `json:"gameid"`
				Gameserverip             string `json:"gameserverip"`
				Lastlogoff               int    `json:"lastlogoff"`
				Loccountrycode           string `json:"loccountrycode"`
				Locstatecode             string `json:"locstatecode"`
				Personaname              string `json:"personaname"`
				Personastate             int    `json:"personastate"`
				Personastateflags        int    `json:"personastateflags"`
				Primaryclanid            string `json:"primaryclanid"`
				Profilestate             int    `json:"profilestate"`
				Profileurl               string `json:"profileurl"`
				Realname                 string `json:"realname"`
				Steamid                  string `json:"steamid"`
				Timecreated              int    `json:"timecreated"`
			} `json:"players"`
		} `json:"response"`
	}

	if err := json.Unmarshal(content, &playerSummariesResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return plySummaries, jsonUnmarshallErrorCheck(content)
		}
		return plySummaries, err
	}

	for _, ply := range playerSummariesResponse.Response.Players {
		id, _ := strconv.ParseUint(ply.Steamid, 10, 64)
		var public bool
		if ply.Communityvisibilitystate == 3 {
			public = true
		}
		var configured bool
		if ply.Profilestate == 1 {
			configured = true
		}
		groupID, _ := strconv.ParseUint(ply.Primaryclanid, 10, 64)
		gameID, _ := strconv.ParseInt(ply.Gameid, 10, 64)
		plySummaries = append(plySummaries, PlayerSummaries{
			SteamID64:      SteamID64(id),
			DisplayName:    ply.Personaname,
			ProfileURL:     ply.Profileurl,
			AvatarSmallURL: ply.Avatar,
			AvatarMedURL:   ply.Avatarmedium,
			AvatarFullURL:  ply.Avatarfull,
			State:          ply.Personastate,
			Public:         public,
			Configured:     configured,
			LastLogOff:     int64(ply.Lastlogoff),

			RealName:           ply.Realname,
			PrimaryGroupID:     GroupID(groupID),
			TimeCreated:        int64(ply.Timecreated),
			CurrentlyPlayingID: int(gameID),
			CurrentlyPlaying:   ply.Gameextrainfo,
			ServerIP:           ply.Gameserverip,
			CountryCode:        ply.Loccountrycode,
		})
	}

	return plySummaries, nil
}

// GetPlayerSummaries returns a PlayerSummaries.
func GetPlayerSummaries(apiKey string, steam64 SteamID64) (PlayerSummaries, error) {
	var plySummaries PlayerSummaries

	resp, err := http.Get("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?" + url.Values{
		"steamids": {strconv.FormatUint(uint64(steam64), 10)},
		"key":      {apiKey},
	}.Encode())
	if err != nil {
		return plySummaries, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return plySummaries, err
	}

	var playerSummariesResponse struct {
		Response struct {
			Players []struct {
				Avatar                   string `json:"avatar"`
				Avatarfull               string `json:"avatarfull"`
				Avatarmedium             string `json:"avatarmedium"`
				Communityvisibilitystate int    `json:"communityvisibilitystate"`
				Gameextrainfo            string `json:"gameextrainfo"`
				Gameid                   string `json:"gameid"`
				Gameserverip             string `json:"gameserverip"`
				Lastlogoff               int    `json:"lastlogoff"`
				Loccountrycode           string `json:"loccountrycode"`
				Locstatecode             string `json:"locstatecode"`
				Personaname              string `json:"personaname"`
				Personastate             int    `json:"personastate"`
				Personastateflags        int    `json:"personastateflags"`
				Primaryclanid            string `json:"primaryclanid"`
				Profilestate             int    `json:"profilestate"`
				Profileurl               string `json:"profileurl"`
				Realname                 string `json:"realname"`
				Steamid                  string `json:"steamid"`
				Timecreated              int    `json:"timecreated"`
			} `json:"players"`
		} `json:"response"`
	}

	if err := json.Unmarshal(content, &playerSummariesResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return plySummaries, jsonUnmarshallErrorCheck(content)
		}
		return plySummaries, err
	}

	if len(playerSummariesResponse.Response.Players) > 0 {
		id, _ := strconv.ParseUint(playerSummariesResponse.Response.Players[0].Steamid, 10, 64)
		var public bool
		if playerSummariesResponse.Response.Players[0].Communityvisibilitystate == 3 {
			public = true
		}
		var configured bool
		if playerSummariesResponse.Response.Players[0].Profilestate == 1 {
			configured = true
		}
		groupID, _ := strconv.ParseUint(playerSummariesResponse.Response.Players[0].Primaryclanid, 10, 64)
		gameID, _ := strconv.ParseInt(playerSummariesResponse.Response.Players[0].Gameid, 10, 64)
		plySummaries = PlayerSummaries{
			SteamID64:      SteamID64(id),
			DisplayName:    playerSummariesResponse.Response.Players[0].Personaname,
			ProfileURL:     playerSummariesResponse.Response.Players[0].Profileurl,
			AvatarSmallURL: playerSummariesResponse.Response.Players[0].Avatar,
			AvatarMedURL:   playerSummariesResponse.Response.Players[0].Avatarmedium,
			AvatarFullURL:  playerSummariesResponse.Response.Players[0].Avatarfull,
			State:          playerSummariesResponse.Response.Players[0].Personastate,
			Public:         public,
			Configured:     configured,
			LastLogOff:     int64(playerSummariesResponse.Response.Players[0].Lastlogoff),

			RealName:           playerSummariesResponse.Response.Players[0].Realname,
			PrimaryGroupID:     GroupID(groupID),
			TimeCreated:        int64(playerSummariesResponse.Response.Players[0].Timecreated),
			CurrentlyPlayingID: int(gameID),
			CurrentlyPlaying:   playerSummariesResponse.Response.Players[0].Gameextrainfo,
			ServerIP:           playerSummariesResponse.Response.Players[0].Gameserverip,
			CountryCode:        playerSummariesResponse.Response.Players[0].Loccountrycode,
		}
	} else {
		return plySummaries, errors.New("No player summaries found")
	}

	return plySummaries, nil
}

// GetFriendsList returns a type FriendsList containing all friends for a specific SteamID64.
func GetFriendsList(steam64 SteamID64, apiKey string) (FriendsList, error) {
	var friends FriendsList

	resp, err := http.Get("https://api.steampowered.com/ISteamUser/GetFriendList/v1/?" + url.Values{
		"key":     {apiKey},
		"steamid": {strconv.FormatUint(uint64(steam64), 10)},
	}.Encode())
	if err != nil {
		return friends, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return friends, err
	}

	var friendsListResponse struct {
		Friendslist struct {
			Friends []struct {
				FriendSince  int    `json:"friend_since"`
				Relationship string `json:"relationship"`
				Steamid      string `json:"steamid"`
			} `json:"friends"`
		} `json:"friendslist"`
	}

	if err := json.Unmarshal(content, &friendsListResponse); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return friends, jsonUnmarshallErrorCheck(content)
		}
		return friends, err
	}

	for _, friend := range friendsListResponse.Friendslist.Friends {
		steamID, err := strconv.ParseUint(friend.Steamid, 10, 64)
		if err != nil {
			continue
		}
		friends = append(friends, struct {
			SteamID     SteamID64
			FriendSince int64
		}{
			SteamID:     SteamID64(steamID),
			FriendSince: int64(friend.FriendSince),
		})
	}

	return friends, nil
}
