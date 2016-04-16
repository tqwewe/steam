package steam

import (
	"time"
	"net/http"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"errors"
	"crypto/rsa"
	"crypto/rand"
	"math/big"
	"encoding/base64"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"fmt"
)

const apiKey = "2B2A0C37AC20B5DC2234E579A2ABB11C"
var Steamid string
var Jar *cookiejar.Jar
var Client *http.Client
var err error

type callBack(string)

func init() {
	Jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	Client = &http.Client{Jar: Jar, Timeout: time.Duration(40 * time.Second)}
}

func Login(username, password string) error {
	var resp 	*http.Response
	var doNotCache	string

	doNotCache = strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)

	// Get RSA Key
	resp, err = Client.PostForm("https://steamcommunity.com/login/getrsakey/", map[string][]string{
		"donotcache": {doNotCache},
		"username": {username},
	})
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}

	err = json.Unmarshal(content, &decoded)
	if err != nil {
		return err
	}

	if decoded["success"] != true {
		return errors.New("Failed to retrieve RSA key.")
	}

	// Set encryption variables
	var privateKey *rsa.PrivateKey
	var publicKey rsa.PublicKey
	var plain_text, encrypted []byte

	plain_text = []byte(password)

	// Generate Private Key
	if privateKey, err = rsa.GenerateKey(rand.Reader, 1024); err != nil {
		return err
	}

	privateKey.Precompute()

	if err = privateKey.Validate(); err != nil {
		return err
	}

	modulus, success := new(big.Int).SetString(decoded["publickey_mod"].(string), 16 /* = base 16 */)
	if !success {
		return errors.New("Unable to set modulus.")
	}


	exponent, success := new(big.Int).SetString(decoded["publickey_exp"].(string), 16 /* = base 16 */)
	if !success {
		return errors.New("Unable to set modulus.")
	}

	publicKey.N = modulus
	publicKey.E = int(exponent.Int64())

	encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, &publicKey, plain_text)
	if err != nil {
		return err
	}

	doNotCache = strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)

	resp, err = Client.PostForm("https://steamcommunity.com/login/dologin/", map[string][]string{
		"donotcache":	{doNotCache},
		"username": 	{username},
		"password": 	{base64.StdEncoding.EncodeToString(encrypted[0:len(encrypted)])},
		"rsatimestamp":	{decoded["timestamp"].(string)},
	})
	if err != nil {
		return err
	}

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var transfer map[string]interface{}

	err = json.Unmarshal(content, &transfer)
	if err != nil {
		return err
	}

	if transfer["success"] != true || transfer["login_complete"] != true {
		return errors.New(transfer["message"].(string))
	}

	resp, err = Client.PostForm("https://store.steampowered.com/login/transfer", map[string][]string{
		"steamid":		{transfer["transfer_parameters"].(map[string]interface{})["steamid"].(string)},
		"token": 		{transfer["transfer_parameters"].(map[string]interface{})["token"].(string)},
		"auth": 		{transfer["transfer_parameters"].(map[string]interface{})["auth"].(string)},
		"token_secure":		{transfer["transfer_parameters"].(map[string]interface{})["token_secure"].(string)},
		"remember_login":	{"true"},
	})
	if err != nil {
		return err
	}

	resp, err = Client.PostForm("https://help.steampowered.com/login/transfer", map[string][]string{
		"steamid":		{transfer["transfer_parameters"].(map[string]interface{})["steamid"].(string)},
		"token": 		{transfer["transfer_parameters"].(map[string]interface{})["token"].(string)},
		"auth": 		{transfer["transfer_parameters"].(map[string]interface{})["auth"].(string)},
		"token_secure":		{transfer["transfer_parameters"].(map[string]interface{})["token_secure"].(string)},
		"remember_login":	{"true"},
	})
	if err != nil {
		return err
	}

	Steamid = transfer["transfer_parameters"].(map[string]interface{})["steamid"].(string)

	return nil
}

func Logout() error {
	sessionid, err := getSessionid()
	if err != nil {
		return err
	}

	Client.PostForm("https://steamcommunity.com/login/logout/", url.Values{
		"sessionid":	[]string{sessionid},
	})

	return nil
}

func Message(recipient int64, message string) error {
	umqid, err := getUmqid()
	if err != nil {
		return err
	}

	accessToken, err := getAccessToken()
	if err != nil {
		return err
	}

	resp, err := Client.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Message/v0001/", url.Values{
		"steamid_dst":		[]string{strconv.FormatInt(recipient, 10)},
		"text":			[]string{message},
		"umqid":		[]string{umqid},
		"_":			[]string{strconv.FormatInt(makeTimestamp(), 10)},
		"type":			[]string{"saytext"},
		"jsonp":		[]string{"1"},
		"access_token":		[]string{accessToken},
	})
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}

	err = json.Unmarshal(content, &decoded)
	if err != nil {
		return err
	}

	if decoded["error"].(string) != "OK" {
		return errors.New(decoded["error"].(string))
	}

	return nil
}

func Broadcast(message string) error {
	resp, err := Client.Get("http://api.steampowered.com/ISteamUser/GetFriendList/v0001/?" + url.Values{
		"key":		[]string{apiKey},
		"steamid":	[]string{Steamid},
		"relationship": []string{"friend"},
	}.Encode())
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var friends map[string]interface{}

	err = json.Unmarshal(content, &friends)
	if err != nil {
		return err
	}

	for _, val := range friends["friendslist"].(map[string]interface{})["friends"].([]interface{}) {
		err = Message(val.(map[string]interface{})["steamid"].(int64), message)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO Complete InviteToGroup function
func InviteToGroup(recipients []int64, groupid string) error {
	sessionid, err := getSessionid()
	if err != nil {
		return err
	}

	var inviteeList string

	for count, val := range recipients {
		if count == 0 {
			inviteeList += "["
		}
		inviteeList += `"` + strconv.FormatInt(val, 10) + `"`
		if count == len(recipients) - 1 {
			inviteeList += "]"
		} else {
			inviteeList += ","
		}
	}

	resp, err := Client.PostForm("http://steamcommunity.com/actions/GroupInvite", url.Values{
		"json":		[]string{"1"},
		"type":		[]string{"groupInvite"},
		"group":	[]string{groupid},
		"sessionID":	[]string{sessionid},
		"invitee_list":	[]string{inviteeList},
	})
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}

	err = json.Unmarshal(content, &decoded)
	if err != nil {
		return err
	}

	if decoded["results"].(string) != "OK" {
		return errors.New(decoded["results"].(string))
	}

	return nil
}

func ListenAndServer(callback func(user int64, message string)) error {
	umqid, err := getUmqid()
	if err != nil {
		return err
	}

	accessToken, err := getAccessToken()
	if err != nil {
		return err
	}

	resp, err := Client.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001/?" + url.Values{
		"jsonp":		[]string{"jQuery1111001475823658514086_1460550648276"},
		"ui_mode":		[]string{"web"},
		"access_token":		[]string{accessToken},
		"_":			[]string{strconv.FormatInt(makeTimestamp(), 10)},
	}.Encode())
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response map[string]interface{}

	r, _ := regexp.Compile(`\/\*\*\/\w+\(`)
	begging := r.FindString(string(content))
	extracted := string(content)[len(begging):len(string(content))-1]

	err = json.Unmarshal([]byte(extracted), &response)
	if err != nil {
		return err
	}

	var msg int64
	var pollid int64

	if val, ok := response["message"].(float64); ok {
		msg = int64(val)
	}

	resp.Body.Close()

	for {
		resp, err := Client.Get("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Poll/v0001/?" + url.Values{
			"jsonp":		[]string{"jQuery1111001475823658514086_1460550648276"},
			"umqid":		[]string{umqid},
			"message":		[]string{strconv.FormatInt(msg, 10)},
			"pollid":		[]string{strconv.FormatInt(pollid, 10)},
			"sectimeout":		[]string{"20"},
			"secidletime":		[]string{"20"},
			"use_accountids":	[]string{"1"},
			"access_token":		[]string{accessToken},
			"_":			[]string{strconv.FormatInt(makeTimestamp(), 10)},
		}.Encode())
		if err != nil {
			return err
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var response map[string]interface{}

		r, _ := regexp.Compile(`\/\*\*\/\w+\(`)
		begging := r.FindString(string(content))
		extracted := string(content)[len(begging):len(string(content))-1]

		err = json.Unmarshal([]byte(extracted), &response)
		if err != nil {
			return err
		}

		for i, v := range response {
			if i == "messages" {
				for _, messages := range v.([]interface {}) {
					if messages.(map[string]interface{})["type"].(string) == "saytext" && len(messages.(map[string]interface{})["text"].(string)) > 0 {
						user := 76500000000000000 + (int64(messages.(map[string]interface{})["accountid_from"].(float64)) + int64(61197960265728))
						message := messages.(map[string]interface{})["text"].(string)
						fmt.Println(message)
						go callback(user, message)
					}
				}

			}
		}

		if val, ok := response["messagelast"].(float64); ok {
			msg = int64(val)
		}
		if val, ok := response["pollid"].(float64); ok {
			pollid = int64(val) + 1
		}

		resp.Body.Close()
	}
}

func GetCookie(cookie string) string {
	url, _ := url.Parse("https://steamcommunity.com")

	for _, v := range Client.Jar.Cookies(url) {
		if v.Name == cookie {
			return v.Value
		}
	}

	return ""
}

func getAccessToken() (string, error) {
	resp, err := Client.Get("https://steamcommunity.com/chat")
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	token := regexp.MustCompile(`CWebAPI\s*\(\s*(?:[^,]+,){2}\s*"([0-9a-f]{32})"\s*\)`).FindSubmatch(content)
	if token == nil {
		return "", errors.New("No token available.")
	}

	return string(token[1]), nil
}

func getUmqid() (string, error) {
	accessToken, err := getAccessToken()

	resp, err := Client.PostForm("https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v0001", url.Values{
		"jsonp":	[]string{"1"},
		"ui_mode":	[]string{"web"},
		"access_token":	[]string{accessToken},
		"_":		[]string{strconv.FormatInt(makeTimestamp(), 10)},
	})
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var decoded map[string]interface{}

	err = json.Unmarshal(content, &decoded)
	if err != nil {
		return "", err
	}

	if len(decoded["umqid"].(string)) <= 0 {
		return "", errors.New("Unable to retreive umqid.")
	}

	return decoded["umqid"].(string), nil
}

func getSessionid() (string, error) {
	resp, err := Client.Get("https://steamcommunity.com/")
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
		return "", errors.New("No sessionid available.")
	}

	return string(sessionid[1]), nil
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
