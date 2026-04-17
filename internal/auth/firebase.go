package auth

import (
	"context"
	"errors"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseIdentity struct {
	UID       string
	Email     string
	FullName  string
	AvatarURL string
}

type FirebaseVerifier interface {
	VerifyIDToken(ctx context.Context, token string) (FirebaseIdentity, error)
}

type AdminFirebaseVerifier struct {
	client interface {
		VerifyIDToken(context.Context, string) (*firebaseauth.Token, error)
	}
}

func NewFirebaseVerifier(ctx context.Context, projectID, credentialsFile string) (FirebaseVerifier, error) {
	if projectID == "" && credentialsFile == "" {
		return nil, errors.New("firebase project or credentials file is required")
	}
	opts := []option.ClientOption{}
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return &AdminFirebaseVerifier{client: client}, nil
}

func (v *AdminFirebaseVerifier) VerifyIDToken(ctx context.Context, token string) (FirebaseIdentity, error) {
	verified, err := v.client.VerifyIDToken(ctx, token)
	if err != nil {
		return FirebaseIdentity{}, err
	}
	email, _ := verified.Claims["email"].(string)
	name, _ := verified.Claims["name"].(string)
	picture, _ := verified.Claims["picture"].(string)
	return FirebaseIdentity{UID: verified.UID, Email: email, FullName: name, AvatarURL: picture}, nil
}
