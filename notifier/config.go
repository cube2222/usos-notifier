package notifier

type Config struct {
	DevelopmentMode         bool `default:"false" split_words:"true"`
	ListenPortHttp          int  `default:"8080" split_words:"true"`
	GeneralPerHourRateLimit int  `default:"1000" split_words:"true"`
	UserPerHourRateLimit    int  `default:"100" split_words:"true"`

	ProjectName                  string `default:"usos-notifier" split_words:"true"`
	CommandsTopic                string `default:"notifier-commands" split_words:"true"`
	NotificationsSubscription    string `default:"notifier-notifications" split_words:"true"`
	UserCreatedTopic             string `default:"notifier-user_created" split_words:"true"`
	GoogleApplicationCredentials string `default:"/var/secrets/google/serviceaccount.json" split_words:"true"`

	FacebookDomain       string `default:"graph.facebook.com" split_words:"true"`
	MessengerApiKey      string `required:"true" split_words:"true"`
	MessengerVerifyToken string `required:"true" split_words:"true"`
}
