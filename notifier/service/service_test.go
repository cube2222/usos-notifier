package service

import (
	"context"
	"log"
	"testing"

)

func TestService_createUser(t *testing.T) {
	s, err := NewService()
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.createUser(context.Background(), "myid")
	if err != nil {
		t.Fatal(err)
	}
}

func TestService_get(t *testing.T) {
	s, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	mID, err := s.getMessengerID(ctx, "8926ea8a-acf0-4d88-bc1e-9908d94e8bb1")
	if err != nil {
		t.Fatal(err)
	}
	uID, err := s.getUserID(ctx, "myid2")
	if err != nil {
		t.Fatal(err)
	}

	log.Println(mID)
	log.Println(uID)
}