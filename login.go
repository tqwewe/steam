package steam

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Login logs into steam using the specified username and password and returns a type Account.
func Login(username, password string) (*Account, error) {
	acc := Account{
		Username: username,
		Password: password,
	}
	cookieJar, _ := cookiejar.New(nil)
	acc.HttpClient = &http.Client{Jar: cookieJar, Timeout: time.Duration(120 * time.Second)}

	resp, err := acc.HttpClient.PostForm("https://steamcommunity.com/login/getrsakey", url.Values{
		"donotcache": {strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)},
		"username":   {acc.Username},
	})
	if err != nil {
		return &acc, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &acc, err
	}

	var rsakeyResult struct {
		Success       bool
		Publickey_mod string
		Publickey_exp string
		Timestamp     string
		Token_gid     string
	}
	if err := json.Unmarshal(content, &rsakeyResult); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return &acc, jsonUnmarshallErrorCheck(content)
		}
		return &acc, err
	}

	if rsakeyResult.Success != true {
		return &acc, errors.New("failed to retrieve RSA key")
	}

	encryptedPassword := encryptPassword(acc.Password, rsakeyResult.Publickey_mod, rsakeyResult.Publickey_exp)
	if encryptedPassword == "" {
		return &acc, errors.New("unable to encrypt password")
	}

	resp, err = acc.HttpClient.PostForm("https://steamcommunity.com/login/dologin", url.Values{
		"donotcache":   {strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)},
		"username":     {acc.Username},
		"password":     {encryptedPassword},
		"rsatimestamp": {rsakeyResult.Timestamp},
		"captchagid":   {"-1"},
		"captcha_text": {""},
	})
	if err != nil {
		return &acc, err
	}
	defer resp.Body.Close()

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return &acc, err
	}

	var loginResult struct {
		Success             bool
		Requires_twofactor  bool
		Login_complete      bool
		Transfer_urls       []string
		Transfer_parameters struct {
			SteamId        string
			Token          string
			Auth           string
			Remember_login bool
			Token_secure   string
		}
		Message string
	}
	if err = json.Unmarshal(content, &loginResult); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return &acc, jsonUnmarshallErrorCheck(content)
		}
		return &acc, err
	}

	SteamId, err := strconv.Atoi(loginResult.Transfer_parameters.SteamId)
	if err == nil {
		acc.SteamID = SteamID64(SteamId)
	}

	if loginResult.Success != true || loginResult.Login_complete != true {
		return &acc, errors.New("failed to login: " + loginResult.Message)
	}

	for _, transferUrl := range loginResult.Transfer_urls {
		resp, err = acc.HttpClient.PostForm(transferUrl, url.Values{
			"steamid":        {loginResult.Transfer_parameters.SteamId},
			"token":          {loginResult.Transfer_parameters.Token},
			"auth":           {loginResult.Transfer_parameters.Auth},
			"token_secure":   {loginResult.Transfer_parameters.Token_secure},
			"remember_login": {"true"},
		})
		if err != nil {
			return &acc, err
		}
		resp.Body.Close()
	}

	return &acc, nil
}

// Logout logs out of Steam for a specified Account clearning all existing cookies.
func (acc *Account) Logout() {
	sessionID, _ := acc.getSessionId()
	acc.HttpClient.PostForm("https://steamcommunity.com/login/logout/", url.Values{
		"sessionid": {sessionID},
	})
	cookieJar, _ := cookiejar.New(nil)
	acc.HttpClient.Jar = cookieJar
}

// IsLoggedIn returns a bool based on weather an Account is logged in or not.
func (acc *Account) IsLoggedIn() bool {
	resp, err := acc.HttpClient.Get("http://steamcommunity.com/")
	if err != nil {
		return false
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return !strings.Contains(string(content), "https://steamcommunity.com/login/home")
}

// Relogin logs into Steam again from a previous type Account updating the session.
func (acc *Account) Relogin() error {
	resp, err := acc.HttpClient.PostForm("https://steamcommunity.com/login/getrsakey", url.Values{
		"donotcache": {strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)},
		"username":   {acc.Username},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rsakeyResult struct {
		Success       bool
		Publickey_mod string
		Publickey_exp string
		Timestamp     string
		Token_gid     string
	}
	if err := json.Unmarshal(content, &rsakeyResult); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	if rsakeyResult.Success != true {
		return errors.New("failed to retrieve RSA key")
	}

	encryptedPassword := encryptPassword(acc.Password, rsakeyResult.Publickey_mod, rsakeyResult.Publickey_exp)
	if encryptedPassword == "" {
		return errors.New("unable to encrypt password")
	}

	resp, err = acc.HttpClient.PostForm("https://steamcommunity.com/login/dologin", url.Values{
		"donotcache":   {strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)},
		"username":     {acc.Username},
		"password":     {encryptedPassword},
		"rsatimestamp": {rsakeyResult.Timestamp},
		"captchagid":   {"-1"},
		"captcha_text": {""},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var loginResult struct {
		Success             bool
		Requires_twofactor  bool
		Login_complete      bool
		Transfer_urls       []string
		Transfer_parameters struct {
			SteamId        string
			Token          string
			Auth           string
			Remember_login bool
			Token_secure   string
		}
		Message string
	}
	if err = json.Unmarshal(content, &loginResult); err != nil {
		if err.Error() == "invalid character '<' looking for beginning of value" {
			return jsonUnmarshallErrorCheck(content)
		}
		return err
	}

	SteamId, err := strconv.Atoi(loginResult.Transfer_parameters.SteamId)
	if err == nil {
		acc.SteamID = SteamID64(SteamId)
	}

	if loginResult.Success != true || loginResult.Login_complete != true {
		return errors.New("failed to login: " + loginResult.Message)
	}

	for _, transferUrl := range loginResult.Transfer_urls {
		resp, err = acc.HttpClient.PostForm(transferUrl, url.Values{
			"steamid":        {loginResult.Transfer_parameters.SteamId},
			"token":          {loginResult.Transfer_parameters.Token},
			"auth":           {loginResult.Transfer_parameters.Auth},
			"token_secure":   {loginResult.Transfer_parameters.Token_secure},
			"remember_login": {"true"},
		})
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return nil
}
