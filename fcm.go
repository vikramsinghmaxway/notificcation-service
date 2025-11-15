package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

func InitFCM() error {
	ctx := context.Background()

	// Yeh code SIRF function ke andar hona chahiye
	opt := option.WithCredentialsFile("serviceAccountKey.json")

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return err
	}

	fcmClient = client

	log.Println("FCM initialized successfully")
	return nil
}

func SendFCM(title, body, token string) error {
	ctx := context.Background()

	msg := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
	}

	_, err := fcmClient.Send(ctx, msg)
	if err != nil {
		return err
	}

	log.Println("Notification sent:", title, body)
	return nil
}
