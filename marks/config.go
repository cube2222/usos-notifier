package marks

type Config struct {
	ProjectName                     string `default:"usos-notifier" split_words:"true"`
	CredentialsAddress              string `default:"credentials:8081" split_words:"true"`
	CredentialsReceivedSubscription string `default:"marks-credentials-credentials_received" split_words:"true"`
	NotificationsTopic              string `default:"notifications" split_words:"true"`
	CommandsSubscription            string `default:"marks-notifier-commands" split_words:"true"`
	GoogleApplicationCredentials    string `default:"/var/secrets/google/serviceaccount.json" split_words:"true"`
}
