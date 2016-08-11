package steam

import (
	"io/ioutil"
	"regexp"
)

// checkCaptcha returns an error if it is unable
// to retrieve the bytes returned from Steam's
// login page.
//
// An empty string with no error will be returned
// if there is no captcha, else the captcha id
// will be returned.
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
		return "", nil
	}

	return string(captchaSlice[1]), nil
}