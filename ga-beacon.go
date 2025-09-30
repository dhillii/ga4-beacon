package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/delay"
)

// Config structure for GA4 settings
type Config struct {
	MeasurementID string `json:"measurement_id"`
	APISecret     string `json:"api_secret"`
}

var config Config

var (
	pixel        = mustReadFile("static/pixel.gif")
	badge        = mustReadFile("static/badge.svg")
	badgeGif     = mustReadFile("static/badge.gif")
	badgeFlat    = mustReadFile("static/badge-flat.svg")
	badgeFlatGif = mustReadFile("static/badge-flat.gif")
	pageTemplate = template.Must(template.New("page").ParseFiles("page.html"))
)

// GA4 Event structure
type GA4Event struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

// GA4 Payload structure
type GA4Payload struct {
	ClientID string     `json:"client_id"`
	Events   []GA4Event `json:"events"`
}

func loadConfig() error {
	configFile := "config.json"
	if envConfig := os.Getenv("CONFIG_FILE"); envConfig != "" {
		configFile = envConfig
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", configFile, err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	if config.MeasurementID == "" || config.APISecret == "" {
		return fmt.Errorf("measurement_id and api_secret are required in config file")
	}

	log.Printf("Loaded config: Measurement ID = %s", config.MeasurementID)
	return nil
}

func main() {
	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func mustReadFile(path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func generateUUID(cid *string) error {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}

	b[8] = (b[8] | 0x80) & 0xBF // what's the purpose ?
	b[6] = (b[6] | 0x40) & 0x4F // what's the purpose ?
	*cid = hex.EncodeToString(b)
	return nil
}

func generateSessionID() string {
	now := time.Now().Unix()
	return strconv.FormatInt(now, 10)
}

var delayHit = delay.Func("collect", logHit)

func sendToGA(c context.Context, ua string, ip string, cid string, payload GA4Payload) error {
	client := &http.Client{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON: %s", err.Error())
		return err
	}

	// Build URL with config values
	beaconURL := fmt.Sprintf("https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s", 
		config.MeasurementID, config.APISecret)

	req, _ := http.NewRequest("POST", beaconURL, bytes.NewBuffer(jsonPayload))
	req.Header.Add("User-Agent", ua)
	req.Header.Add("Content-Type", "application/json")

	if resp, err := client.Do(req); err != nil {
		log.Printf("GA collector POST error: %s", err.Error())
		return err
	} else {
		log.Printf("GA collector status: %v, cid: %v, ip: %s", resp.Status, cid, ip)
		log.Printf("Reported payload: %v", string(jsonPayload))
	}
	return nil
}

func logHit(c context.Context, params []string, query url.Values, ua string, ip string, cid string) error {
	
	// Create GA4 payload matching the Apps Script structure
	event := GA4Event{
		Name: "page_view",
		Params: map[string]interface{}{
			"session_id":          generateSessionID(),
			"user_agent":          ua,
			"ip_address":          ip,
			"timestamp":           time.Now().Format(time.RFC3339),
		},
	}

	// Add any additional query parameters as custom parameters
	for key, values := range query {
		if len(values) > 0 && !isReservedParam(key) {
			event.Params["custom_"+key] = values[0]
		}
	}

	payload := GA4Payload{
		ClientID: cid,
		Events:   []GA4Event{event},
	}

	return sendToGA(c, ua, ip, cid, payload)
}

// Helper function to check if a parameter is reserved
func isReservedParam(param string) bool {
	reserved := []string{"referer", "pixel", "gif", "flat", "flat-gif", "useReferer"}
	for _, r := range reserved {
		if param == r {
			return true
		}
	}
	return false
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := r.Context()
	params := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)
	query, _ := url.ParseQuery(r.URL.RawQuery)
	refOrg := r.Header.Get("Referer")

	// Add referer to query for tracking
	if refOrg != "" {
		query.Set("referer", refOrg)
	}

	// / -> redirect
	if len(params[0]) == 0 {
		http.Redirect(w, r, "https://github.com/igrigorik/ga-beacon", http.StatusFound)
		return
	}

	// activate referrer path if ?useReferer is used and if referer exists
	if _, ok := query["useReferer"]; ok {
		if len(refOrg) != 0 {
			referer := strings.Replace(strings.Replace(refOrg, "http://", "", 1), "https://", "", 1)
			if len(referer) != 0 {
				// if the useReferer is present and the referer information exists
				//  the path is ignored and the beacon referer information is used instead.
				params = strings.SplitN(strings.Trim(r.URL.Path, "/")+"/"+referer, "/", 2)
			}
		}
	}

	// /account -> account template
	if len(params) == 1 {
		templateParams := struct {
			Account string
			Referer string
		}{
			Account: params[0],
			Referer: refOrg,
		}
		if err := pageTemplate.ExecuteTemplate(w, "page.html", templateParams); err != nil {
			http.Error(w, "could not show account page", 500)
			log.Printf("Cannot execute template: %v", err)
		}
		return
	}

	// /account/page -> GIF + log pageview to GA collector
	var cid string
	if cookie, err := r.Cookie("cid"); err != nil {
		if err := generateUUID(&cid); err != nil {
			log.Printf("Failed to generate client UUID: %v", err)
		} else {
			log.Printf("Generated new client UUID: %v", cid)
			http.SetCookie(w, &http.Cookie{Name: "cid", Value: cid, Path: fmt.Sprint("/", params[0])})
		}
	} else {
		cid = cookie.Value
		log.Printf("Existing CID found: %v", cid)
	}

	if len(cid) != 0 {
		var cacheUntil = time.Now().Format(http.TimeFormat)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, private")
		w.Header().Set("Expires", cacheUntil)
		w.Header().Set("CID", cid)

		logHit(c, params, query, r.Header.Get("User-Agent"), r.RemoteAddr, cid)
		// delayHit.Call(c, params, r.Header.Get("User-Agent"), cid)
	}

	// Write out GIF pixel or badge, based on presence of "pixel" param.
	if _, ok := query["pixel"]; ok {
		w.Header().Set("Content-Type", "image/gif")
		w.Write(pixel)
	} else if _, ok := query["gif"]; ok {
		w.Header().Set("Content-Type", "image/gif")
		w.Write(badgeGif)
	} else if _, ok := query["flat"]; ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(badgeFlat)
	} else if _, ok := query["flat-gif"]; ok {
		w.Header().Set("Content-Type", "image/gif")
		w.Write(badgeFlatGif)
	} else {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(badge)
	}
}