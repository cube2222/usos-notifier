package service

import (
	"context"
	"log"
	"os"
	"testing"
)

/*
func setupDefaultService() *Service {
	cli, err := google.DefaultClient(context.Background(), cloudkms.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	kms, err := cloudkms.New(cli)
	if err != nil {
		log.Fatal(err)
	}
	ds, err := datastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatal(err)
	}

	return &Service{
		ds:  ds,
		kms: kms,
	}
}

func TestService_GetSession(t *testing.T) {
	service := setupDefaultService()

	sess, err := service.GetSession(context.Background(), &credentials.GetSessionRequest{
		Userid: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("%+v", *sess)
}
*/
func TestService_login(t *testing.T) {
	sess, err := login(context.Background(), os.Getenv("usos_user"), os.Getenv("usos_pass"))
	if err != nil {
		t.Fatal(err)
	}

	log.Println(sess)
}
