package graylog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var currentSessionToken sessionToken

type sessionToken struct {
	ValidUntil string `json:"valid_until"`
	SessionId  string `json:"session_id"`
}

func getIndexSetId() string {
	if !isGrayLogActive() {
		return ""
	}

	client := http.DefaultClient

	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl() + "/api/system/indices/index_sets", nil)
	if err != nil {
		log.Fatal(err)
	}


	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error occured while querying graylog for index_sets. Error was %s", err.Error()))
	}

	if resp.StatusCode != 200 {
		log.Fatal(fmt.Sprintf("A StatusCode != 200 was received from Graylog while querying for index_sets. Code was: %d", resp.StatusCode))
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error occured while parsing result from querying graylog for index_sets. Error was %s", err.Error()))
	}

	var iSet IndexSets
	err = json.Unmarshal(content, &iSet)
	if err != nil {
		log.Fatal(fmt.Sprintf("An error occured while unmarshalling Graylog Response for index-set-id into local datastructure. Error was %s", err.Error()))
	}

	if iSet.Total <= 0 {
		log.Fatal(fmt.Sprintf("The received IndexSets from Graylog are empty. This is a error on the graylog side, please report to icc@informatik.haw-hamburg.de!"))
	}

	if iSet.IndexSets[0].Id == "" {
		log.Fatal(fmt.Sprintf("The received Default IndexSet's Id from Graylog is empty. This is a error on the graylog side, please report to icc@informatik.haw-hamburg.de!"))
	}

	return iSet.IndexSets[0].Id
}

func getGraylogBaseUrl() string {
	url := ""
	if os.Getenv("GRAYLOG_BASE_URL") != "" {
		url = os.Getenv("GRAYLOG_BASE_URL")
	}
	return url
}

func getGraylogUserName() string {
	user := "admin"
	if os.Getenv("GRAYLOG_ADMIN_USER") != "" {
		user = os.Getenv("GRAYLOG_ADMIN_USER")
	}
	return user
}

func getGraylogPassword() string {
	pw := ""
	if os.Getenv("GRAYLOG_ADMIN_PASSWORD") != "" {
		pw = os.Getenv("GRAYLOG_ADMIN_PASSWORD")
	}
	return pw
}

func isGrayLogActive() bool {
	return os.Getenv("GRAYLOG_BASE_URL") != ""
}

func getGraylogSessionToken() string {
	if currentSessionToken.SessionId != "" && isStillValid(currentSessionToken.ValidUntil) {
		return currentSessionToken.SessionId
	}

	body := []byte(fmt.Sprintf(`{"username":"%s", "password":"%s", "host":""}`, getGraylogUserName(), getGraylogPassword()))
	resp, err := http.Post(getGraylogBaseUrl()+"/api/system/sessions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(fmt.Sprintf("Error occured while fetching session token from Graylog. Error was %s", err.Error()))
	}

	if resp.StatusCode != 200 {
		log.Println(fmt.Sprintf("Reponse code was not 200 while receiving session token from Graylog. Code was: %d", resp.StatusCode))
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error occured while reading result from fetching session token from Graylog. Error was %s", err.Error()))
	}

	err = json.Unmarshal(content, &currentSessionToken)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error occured while unmarshalling result from fetching session token from Graylog. Error was %s", err.Error()))
	}

	if currentSessionToken.SessionId == "" {
		log.Fatal("Received SessionId was empty, this is a bug in Graylog, please contact icc@informatik.haw-hamburg.de")
	}

	return currentSessionToken.SessionId
}

func isStillValid(validUntil string) bool {
	str := strings.Replace(validUntil, "+0000", "Z", 1)
	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error while Parsing timestamp %s. Error was: %s",validUntil,err))
	}
	return t.After(time.Now().Add(time.Minute * 5))
}
