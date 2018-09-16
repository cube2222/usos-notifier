package credentials

type Config struct {
	ListenPortHttp int `default:"8080" split_words:"true"`
	ListenPortGrpc int `default:"8081" split_words:"true"`

	ProjectName                  string `default:"usos-notifier" split_words:"true"`
	AdditionalAuthenticatedData  string `default:"something" split_word:"true"`
	EncryptionKeyID              string `default:"projects/usos-notifier/locations/global/keyRings/credentials/cryptoKeys/credentials" split_word:"true"`
	CredentialsReceivedTopic     string `default:"credentials-credentials_received" split_words:"true"`
	NotificationsTopic           string `default:"notifications" split_words:"true"`
	UserCreatedSubscription      string `default:"credentials-notifier-user_created" split_words:"true"`
	GoogleApplicationCredentials string `default:"/var/secrets/google/serviceaccount.json" split_words:"true"`
}
