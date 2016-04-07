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
)

jar, err := cookiejar.New(nil)
if err != nil {
	return err
}

const Client := &http.Client{Jar: jar}

func Login(username, password string) error {
	var err 	error
	var resp 	*http.Response
	var doNotCache	string

	doNotCache = strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)

	// Get RSA Key
	resp, err = Client.PostForm("https://steamcommunity.com/login/getrsakey/", map[string][]string{
		"donotcache": {doNotCache},
		"username": {username},
	})
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

	modulus, bol := new(big.Int).SetString(decoded["publickey_mod"].(string), 16 /* = base 16 */)
	if !bol {
		return errors.New("Unable to set modulus.")
	}


	exponent, bol := new(big.Int).SetString(decoded["publickey_exp"].(string), 16 /* = base 16 */)
	if !bol {
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

	return nil
}
