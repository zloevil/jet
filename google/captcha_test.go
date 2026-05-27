//go:build integration

package google

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type captchaTestSuite struct {
	jet.Suite
}

func (s *captchaTestSuite) SetupSuite() {
	s.Suite.Init(nil)
}

func TestCaptchaSuite(t *testing.T) {
	suite.Run(t, new(captchaTestSuite))
}

const (
	dummyV2Captcha = "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"
	dummyV2Key     = "6LeIxAcTAAAAAGG-vFI1TnRWxMZNFuojJ4WifJWe"
)

func (s *captchaTestSuite) Test_WhenPassedWithDummyKey() {
	cpt := NewCaptcha(&Config{ReCaptchaSecretV2: dummyV2Key}, s.L())
	r, err := cpt.Verify(s.Ctx, &CaptchaRequest{Captcha: dummyV2Captcha, ClientIP: "0.0.0.0", Version: "v2"})
	s.NoError(err)
	s.True(r)
}

func (s *captchaTestSuite) Test_WhenInvalid() {

	test := func(key, cap, ver string) {
		cpt := NewCaptcha(&Config{ReCaptchaSecretV2: key}, s.L())
		r, _ := cpt.Verify(s.Ctx, &CaptchaRequest{Captcha: cap, ClientIP: "0.0.0.0", Version: ver})
		s.False(r)
	}

	test("invalid", dummyV2Captcha, "v2")
	test(dummyV2Key, dummyV2Captcha, "invalid")
	test("", "", "")
}
