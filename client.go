package main

import (
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type UserClient struct {
	id             string
	token          *oauth2.Token
	spotifyClient  *spotify.Client
	signInComplete chan bool
}
