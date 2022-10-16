package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"log"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// Long-lived access token from home-assistant
var HAAS_TOKEN string

// Home-assistant api uri
var HAAS_URL string

// Device in home assisant to send the notification to
var HAAS_NOTIFY_DEVICE string

// Port to listen for webhooks from grafana
var LISTEN_PORT string

// Host/ip to listen for webhooks from grafana
var LISTEN_HOST string

type GrafanaJson struct {
	Title       string   `json:"title"`
	RuleID      int64    `json:"ruleId"`
	RuleName    string   `json:"ruleName"`
	State       string   `json:"state"`
	RuleURL     string   `json:"ruleUrl"`
	ImageURL    string   `json:"imageUrl"`
	Message     string   `json:"message"`
	OrgID       int      `json:"orgId"`
	DashboardID int      `json:"dashboardId"`
	PanelID     int      `json:"panelId"`
	Tags        struct{} `json:"tags"`
	EvalMatches []struct {
		Value  int         `json:"value"`
		Metric string      `json:"metric"`
		Tags   interface{} `json:"tags"`
	} `json:"evalMatches"`
}

func receiveHook(rw http.ResponseWriter, req *http.Request) {

	// Parse webhook data
	var j GrafanaJson
	body, _ := io.ReadAll(req.Body)
	json.Unmarshal(body, &j)

	// send notification
	log.Println(string(body))
	notify(j)
}

func notify(hookData GrafanaJson) {

	// Encode the data
	postBody, err := json.Marshal(map[string]interface{}{
		"message": hookData.Message,
		"title":   hookData.Title,
		"data": map[string]string{
			"image": hookData.ImageURL,
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/notify/%s", HAAS_URL, HAAS_NOTIFY_DEVICE), bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+HAAS_TOKEN)

	// Don't care about the response
	_, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "env-file"},
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "port",
			Usage:    "port to listen on",
			Aliases:  []string{"p"},
			Value:    "80",
			Required: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "host",
			Usage:    "host to listen on",
			Aliases:  []string{"ho"},
			Value:    "localhost",
			Required: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "haas-token",
			Usage:    "token for home assistant to authenticate to the api",
			Aliases:  []string{"ht"},
			Required: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "haas-url",
			Usage:    "url for home assistant",
			Aliases:  []string{"url"},
			Required: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "haas-notify-device",
			Usage:    "device in home assistant to send the notification to",
			Aliases:  []string{"hnd"},
			Required: false,
		}),
	}
	app := &cli.App{
		Name:                   "home-assistant-grafana-relay",
		Usage:                  "listens for grafana notifications and relays to home assistant",
		UseShortOptionHandling: true,
		Flags:                  flags,
		Before:                 altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("env-file")),
		Action: func(cCtx *cli.Context) error {
			LISTEN_HOST = cCtx.String("host")
			LISTEN_PORT = cCtx.String("port")
			HAAS_URL = cCtx.String("haas-url")
			HAAS_TOKEN = cCtx.String("haas-token")
			HAAS_NOTIFY_DEVICE = cCtx.String("haas-notify-device")

			log.Println("Listening for webooks on: " + LISTEN_HOST + ":" + LISTEN_PORT)
			log.Println("Using home-assistant at: " + HAAS_URL)
			http.HandleFunc("/", receiveHook)
			log.Fatal(http.ListenAndServe(LISTEN_HOST+":"+LISTEN_PORT, nil))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
