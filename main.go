package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kmei3560/libdns/totaluptime"
)

const (
	REMOTEURL = "https://api.totaluptime.com/CloudDNS/Domain/All"
)

var provider totaluptime.Provider

func main() {
	client := http.Client{}

	// configure http basic auth
	auth := USERNAME + ":" + PASSWORD
	basicAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	// configure http request
	req, err := http.NewRequest(http.MethodGet, REMOTEURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// body, err := httputil.DumpResponse(resp, true)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	fmt.Println(string(body))
}
