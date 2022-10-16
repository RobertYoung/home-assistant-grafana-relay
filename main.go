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
var TOKEN string

// Home-assistant api uri
var URL string

// Device in home assisant to send the notification to
var DEVICE string

// Port to listen for webhooks from grafana
var PORT string

// Host/ip to listen for webhooks from grafana
var HOST string

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

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/notify/%s", URL, DEVICE), bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+TOKEN)

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
			EnvVars:  []string{"PORT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "host",
			Usage:    "host to listen on",
			Aliases:  []string{"ho"},
			Value:    "localhost",
			Required: false,
			EnvVars:  []string{"HOST"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "token",
			Usage:    "token for home assistant to authenticate to the api",
			Required: false,
			EnvVars:  []string{"TOKEN"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "url",
			Usage:    "url for home assistant",
			Required: false,
			EnvVars:  []string{"URL"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "device",
			Usage:    "device in home assistant to send the notification to",
			Required: false,
			EnvVars:  []string{"DEVICE"},
		}),
	}
	app := &cli.App{
		Name:                   "home-assistant-grafana-relay",
		Usage:                  "listens for grafana notifications and relays to home assistant",
		UseShortOptionHandling: true,
		Flags:                  flags,
		Before:                 altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("env-file")),
		Action: func(cCtx *cli.Context) error {
			HOST = cCtx.String("host")
			PORT = cCtx.String("port")
			URL = cCtx.String("url")
			TOKEN = cCtx.String("token")
			DEVICE = cCtx.String("device")

			log.Println("Listening for webooks on: " + HOST + ":" + PORT)
			log.Println("Using home-assistant at: " + URL)
			http.HandleFunc("/", receiveHook)
			log.Fatal(http.ListenAndServe(HOST+":"+PORT, nil))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
