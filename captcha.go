package steam

import (
	"io/ioutil"
	"regexp"
)

// checkCaptcha returns the current captcha for an Account.
//
// If there is no captcha required then a -1 string will be returned.
func (acc *Account) checkCaptcha() (string, error) {
	resp, err := acc.HttpClient.Get("https://steamcommunity.com/login/home")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	captchaSlice := regexp.MustCompile(`\bgidCaptcha:\B\s"(\w+)"`).FindSubmatch(content)
	if len(captchaSlice) < 2 {
		return "", err
	}

	return string(captchaSlice[1]), nil
}
