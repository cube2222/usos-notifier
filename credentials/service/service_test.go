package service

import (
	"context"
	"log"
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/cube2222/usos-notifier/credentials"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
)

func setupDefaultService() *Service {
	cli, err := google.DefaultClient(context.Background(), cloudkms.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	kms, err := cloudkms.New(cli)
	if err != nil {
		log.Fatal(err)
	}
	ds, err := datastore.NewClient(context.Background(), "usos-notifier", option.WithCredentialsFile("C:/Development/Projects/Go/src/github.com/cube2222/usos-notifier/usos-notifier-9a2e44d7f26b.json"))
	if err != nil {
		log.Fatal(err)
	}

	return &Service{
		ds:  ds,
		kms: kms,
	}
}

func TestService_handleSignup(t *testing.T) {
	service := setupDefaultService()

	err := service.handleSignup(context.Background(), "user", "password", "uuid")
	if err != nil {
		log.Fatal(err)
	}
}

func TestService_GetSession(t *testing.T) {
	service := setupDefaultService()

	sess, err := service.GetSession(context.Background(), &credentials.GetSessionRequest{
		Userid: "bb4907ba-f8f2-45aa-b899-b16f7f4348ba",
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("%+v", *sess)
}

/*func TestService_login(t *testing.T) {
	service := setupDefaultService()

	sess, err := service.login(os.Getenv("usos_user"), os.Getenv("usos_pass"))
	if err != nil {
		t.Fatal(err)
	}

	log.Println(sess)
}*/

