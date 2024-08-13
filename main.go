package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/zmb3/spotify/v2"
)

// TODO Use session variable to user data
var userId = os.Getenv("USER_ID")

const redirectURI = "http://localhost:8080/callback"

const DB_FILE = "db.json"

var playerHtml = `
<br/>
<a href="/player/play">Play</a><br/>
<a href="/player/pause">Pause</a><br/>
<a href="/player/next">Next track</a><br/>
<a href="/player/previous">Previous Track</a><br/>
<a href="/player/shuffle">Shuffle</a><br/>

`

var (
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadCurrentlyPlaying,
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserModifyPlaybackState),
	)
	state = "testing"
)

func main() {
	db := newDatabase(DB_FILE)
	go db.run()

	hub := newHub(db)
	go hub.run()

	http.HandleFunc("/sign-in/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if _, ok := hub.clients[userId]; ok {
			log.Printf("%d already signed in.", userId)
			http.Redirect(w, r, "/player/", http.StatusSeeOther)
			return
		}

		userData, ok := db.userTable[userId]

		if !ok {
			// Sign up
			url := auth.AuthURL(state)
			fmt.Fprintln(w, "Please log in to Spotify by visiting the following page in your browser:", url)
			return
		}

		fmt.Println("Cached token exists.")
		newToken, err := auth.RefreshToken(ctx, userData.Token)
		if err != nil {
			log.Fatal(err)
		}
		client := &UserClient{
			id:             userId,
			token:          newToken,
			spotifyClient:  spotify.New(auth.Client(ctx, newToken)),
			signInComplete: make(chan bool),
		}
		hub.register <- client
		// <-client.signInComplete
		http.Redirect(w, r, "/player/", http.StatusSeeOther)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatal(err)
		}
		if st := r.FormValue("state"); st != state {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", st, state)
		}

		spotifyClient := spotify.New(auth.Client(ctx, token))

		// use the token to get an authenticated client
		client := &UserClient{
			id:             userId,
			token:          token,
			spotifyClient:  spotifyClient,
			signInComplete: make(chan bool),
		}
		fmt.Printf("token: %s\n", token)

		hub.register <- client
		// <-client.signInComplete
		http.Redirect(w, r, "/player/", http.StatusSeeOther)
	})

	http.HandleFunc("/change-user/", func(w http.ResponseWriter, r *http.Request) {
		userId = html.EscapeString(r.URL.Query().Get("id"))
		fmt.Fprintf(w, "%s", userId)
	})

	http.HandleFunc("/player/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		action := strings.TrimPrefix(r.URL.Path, "/player/")
		fmt.Println("Got request for:", action)
		client, ok := hub.clients[userId]
		if !ok {
			log.Print("Sign in required.")
			http.Redirect(w, r, "/sign-in/", http.StatusSeeOther)
			return
		}

		spotifyClient := client.spotifyClient
		playerState, err := spotifyClient.PlayerState(ctx)
		if err != nil {
			log.Print(err)
			return
		}

		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
		switch action {
		case "play":
			err = spotifyClient.Play(ctx)
		case "pause":
			err = spotifyClient.Pause(ctx)
		case "next":
			err = spotifyClient.Next(ctx)
		case "previous":
			err = spotifyClient.Previous(ctx)
		case "shuffle":
			playerState.ShuffleState = !playerState.ShuffleState
			err = spotifyClient.Shuffle(ctx, playerState.ShuffleState)
		}
		if err != nil {
			log.Print(err)
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "%s%s\n%s", "Last action: ", action, playerHtml)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
