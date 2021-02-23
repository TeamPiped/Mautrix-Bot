package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

var homeserver = getEnv("homeserver", "matrix.org")
var username = getEnv("username", "piped")
var password = getEnv("password", "")

var h2client = &http.Client{}

var client mautrix.Client

// user agent to use
var ua = "Mozilla/5.0 (Windows NT 10.0; rv:78.0) Gecko/20100101"

func main() {
	flag.Parse()
	if username == "" || password == "" || homeserver == "" {
		fmt.Println("Please use environment variables to set the username, password and homeserver.")
		os.Exit(1)
	}

	fmt.Println("Logging into", homeserver, "as", username)
	mautrixClient, err := mautrix.NewClient(homeserver, "", "")

	if err != nil {
		panic(err)
	}

	client = *mautrixClient

	_, err = client.Login(&mautrix.ReqLogin{
		Type:             "m.login.password",
		Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: username},
		Password:         password,
		StoreCredentials: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Login successful")

	oei := &mautrix.OldEventIgnorer{UserID: client.UserID}
	syncer := client.Syncer.(mautrix.ExtensibleSyncer)
	oei.Register(syncer)

	syncer.OnEventType(event.EventMessage, func(_ mautrix.EventSource, evt *event.Event) {

		if evt.Sender == client.UserID {
			return
		}

		go handleEvent(evt)

	})

	err = client.Sync()
	if err != nil {
		panic(err)
	}
}

func handleEvent(evt *event.Event) {
	content := strings.ToLower(evt.Content.AsMessage().Body)

	searchPrefix := "!search"

	if strings.HasPrefix(content, searchPrefix) {

		index := len(searchPrefix) + 1

		if len(content) > index {

			q := content[index:]

			request, err := http.NewRequest("GET", "https://pipedapi.kavin.rocks/search?q="+url.QueryEscape(q), nil)

			request.Header.Set("User-Agent", ua)

			if err != nil {
				log.Panic(err)
			}

			resp, err := h2client.Do(request)

			if err != nil {
				log.Panic(err)
			}

			bArray, err := io.ReadAll(resp.Body)

			var result SearchResult

			err = json.Unmarshal(bArray, &result)

			var errored bool

			if err != nil {
				errored = true
			}

			var message string

			if len(result.Items) == 0 {
				message = "No results found."
			} else if !errored {
				item := result.Items[0]
				message = "[" + item.Name + "]" + "(" + "https://piped.kavin.rocks" + item.URL + ")" + "\n" + "Thumbnail: [here](" + item.Thumbnail + ")"
				if item.Views > 0 {
					message += "\nViews: `" + fmt.Sprint(item.Views) + "`"
				}
				if item.Duration > 0 {
					message += "\nDuration: `" + fmt.Sprint(item.Duration) + " seconds`"
				}
			} else {
				message = "```\n" + string(bArray) + "\n```"
			}

			client.SendMessageEvent(evt.RoomID, event.EventMessage, format.RenderMarkdown(message, true, false))

		} else {
			client.SendMessageEvent(evt.RoomID, event.EventMessage, &event.MessageEventContent{
				MsgType: event.MsgText,
				Body:    "No argument given!",
			})
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// SearchResult is a JSONObject
type SearchResult struct {
	Nextpage string `json:"nextpage"`
	ID       string `json:"id"`
	Items    []struct {
		Name      string `json:"name"`
		Thumbnail string `json:"thumbnail"`
		URL       string `json:"url"`
		Views     int    `json:"views,omitempty"`
		Duration  int    `json:"duration,omitempty"`
	} `json:"items"`
}
