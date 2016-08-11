package steam

import (
	"io/ioutil"
	"regexp"
)

// checkCaptcha returns the current captcha for an Account.
//
// If there is no captcha or if an error occurs then an empty string will be returned.
func (acc *Account) checkCaptcha() string {
	resp, err := acc.HttpClient.Get("https://steamcommunity.com/login/home")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	captchaSlice := regexp.MustCompile(`\bgidCaptcha:\B\s"(\w+)"`).FindSubmatch(content)
	if len(captchaSlice) < 2 {
		return ""
	}

	return string(captchaSlice[1])
}