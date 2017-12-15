package graylog

import (
	"net/http"
	"encoding/json"
	"log"
	"bytes"
	"fmt"
)

type Stream struct {
	Title 								string `json:"Title"`
	Description 						string `json:"Description"`
	Rules 								[]Rule `json:"rules"`
	ContentPack 						string `json:"content_pack"`
	MatchingType 						string `json:"matching_type"`
	RemoveMatchesFromDefaultStream 		bool   `json:"remove_matches_from_default_stream"`
	IndexSetId 							string `json:"index_set_id"`
}

type Rule struct {
	Type 		int `json:"type"`
	Value 		string 	`json:"value"`
	Field 		string	`json:"field"`
	Inverted 	bool `json:"inverted"`
	Description string 	`json:"description"`
}

func createStream(namespaceName string) {
	client := &http.Client{}
	requestObject := Stream{
		Title: namespaceName,
		Description: fmt.Sprintf("Logs for namespace %s", namespaceName),
	}

	body, err := json.Marshal(requestObject)

	if err != nil {
		log.Fatal(err.Error())
	}
	req, err := http.NewRequest("POST", "https://icc-logging.informatik.haw-hamburg.de/api/streams",  bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream creation. Error was: %s", err.Error()))
	}

	switch resp.StatusCode{
	case 200:
	case 403:
		log.Println("Graylog communication failed due to permission denied for user.")
	}

}

func deleteStream(namespaceName string) {

}

func grantPermissionToStream(namespaceName, username string) {

}

func takePermissionToStream(namespaceName, username string) {

}
