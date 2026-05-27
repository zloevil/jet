package jet

import (
	"github.com/nyaruka/phonenumbers"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// IsEmailValid checks email format
func IsEmailValid(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	match, err := regexp.MatchString("^(((\\\\.)|[^\\s[:cntrl:]\\(\\)<>@,;:'\\\\\\\"\\.\\[\\]]|')+|(\"(\\\\\"|[^\"])*\"))(\\.(((\\\\.)|[^\\s[:cntrl:]\\(\\)<>@,;:'\\\\\\\"\\.\\[\\]]|')+|(\"(\\\\\"|[^\"])*\")))*@[a-zA-Z0-9\\p{L}](?:[a-zA-Z0-9\\p{L}-]{0,61}[a-zA-Z0-9\\p{L}])?(?:\\.[a-zA-Z0-9\\p{L}](?:[a-zA-Z0-9\\p{L}-]{0,61}[a-zA-Z0-9\\p{L}])?)*$", email)
	return match && err == nil
}

func IsUrlValid(url string) bool {
	match, err := regexp.MatchString("^(https:\\/\\/)?(http:\\/\\/)?(www\\.)?(([-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6})|(localhost:[0-9]{1,4}))\\b([-a-zA-Z0-9()@:%_\\+.~#?&\\/=]*)$", url)
	return match && err == nil
}

// IsIpV4Valid checks ip v4 format
func IsIpV4Valid(ip string) bool {
	match, err := regexp.MatchString("^((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4}$", ip)
	return match && err == nil
}

// IsIpV6Valid checks ip v6 format
func IsIpV6Valid(ip string) bool {
	match, err := regexp.MatchString("^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$", ip)
	return match && err == nil
}

// IsPhoneValid checks phone format (with country code without special characters)
func IsPhoneValid(phone string) bool {
	match, err := regexp.MatchString("^\\d{2,14}$", phone)
	return match && err == nil
}

// IsPhoneWithCountryCodeValid checks phone with country code
func IsPhoneWithCountryCodeValid(code, phone string) bool {

	phoneLen := len(phone)

	if code == "" || phoneLen == 0 {
		return false
	}

	// parse phone number
	phoneUint, err := strconv.ParseUint(phone, 10, 64)
	if err != nil || phoneUint == 0 {
		return false
	}

	// parse code
	codeUint, err := strconv.ParseUint(code, 10, 32)
	if err != nil {
		return false
	}

	phoneRq := &phonenumbers.PhoneNumber{
		CountryCode:    UInt64ToInt32Ptr(&codeUint),
		NationalNumber: &phoneUint,
	}

	// special handling of TON numbers
	region := phonenumbers.GetRegionCodeForNumber(phoneRq)
	if region == "001" {
		return phoneLen <= 15
	}

	return phonenumbers.IsValidNumber(phoneRq)
}

// IsRussianPhoneValid checks Russian phone format (with country code without special characters)
func IsRussianPhoneValid(phone string) bool {
	match, err := regexp.MatchString("^(7|8)\\d{10}$", phone)
	return match && err == nil
}

func IsTelegramUsernameValid(username string) bool {
	if strings.Contains(username, "__") {
		return false
	}
	match, err := regexp.MatchString(`^@[A-Za-z]\w{2,29}[\dA-Za-z]$`, username)
	return match && err == nil
}

func IsTelegramChannelValid(channel string) bool {
	if channel == "" {
		return false
	}
	ok, _ := regexp.MatchString("^(https?:\\/\\/)?(www[.])?(telegram|t)\\.me\\/([a-zA-Z0-9_-]*)\\/?$", channel)
	return ok
}

// IsCoordinateValid checks if coordinate valid
func IsCoordinateValid(c string) bool {
	ok, _ := regexp.MatchString(`^-?[0-9]{1,2}\.[0-9]{5,7}$`, c)
	return ok
}

// ExtractUrlExtension extract extension from url
func ExtractUrlExtension(s string) (string, error) {
	if !IsUrlValid(s) {
		return "", nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	pos := strings.LastIndex(u.Path, ".")
	if pos == -1 {
		return "", nil //couldn't find a period to indicate a file extension
	}
	return u.Path[pos+1 : len(u.Path)], nil
}
