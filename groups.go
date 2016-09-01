package steam

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
)

// GetGroupMembers uses a group url name (http://steamcommunity.com/groups/GOLANG) and returns a slice of
// the group members.
func GetGroupMembers(groupName string) ([]SteamID64, error) {
	resp, err := http.Get("http://steamcommunity.com/groups/" + groupName + "/memberslistxml?json=1&xml=1")
	if err != nil {
		return []SteamID64{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []SteamID64{}, err
	}

	var membersListXmlResponse struct {
		SteamID64 []SteamID64 `xml:"members>steamID64"`
	}

	if err := xml.Unmarshal(body, &membersListXmlResponse); err != nil {
		return []SteamID64{}, err
	}

	return membersListXmlResponse.SteamID64, nil
}
