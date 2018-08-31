package notifier

type Config struct {
	CommandsTopic                string `default:"notifier-commands"`
	DevelopmentMode              bool   `default:"false"`
	FacebookDomain               string `default:"graph.facebook.com"`
	GeneralPerHourRateLimit      int    `default:"1000"`
	GoogleApplicationCredentials string `default:"/var/secrets/google/serviceaccount.json"`
	ListenPortHttp               int    `default:"8080"`
	MessengerApiKey              string `required:"true"`
	MessengerVerifyKey           string `required:"true"`
	NotificationsTopic           string `default:"notifier-notifications"`
	ProjectName                  string `default:"usos-notifier"`
	UserPerHourRateLimit         int    `default:"100"`
}
