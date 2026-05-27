package google

type Config struct {
	ConfigurationPath string  `mapstructure:"configuration_path"`
	JsonConfiguration string  `mapstructure:"json_configuration"`
	ClientTimeout     int     `mapstructure:"client_timeout"`
	ReCaptchaSecretV2 string  `mapstructure:"recaptcha_secret_v2"`
	ReCaptchaSecretV3 string  `mapstructure:"recaptcha_secret_v3"`
	ReCaptchaScore    float64 `mapstructure:"recaptcha_score"`
}
