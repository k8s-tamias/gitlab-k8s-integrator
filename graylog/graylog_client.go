package graylog

import (
	"net/http"
	"encoding/json"
	"log"
	"bytes"
	"fmt"
	"io/ioutil"
)

type Stream struct {
	Id									string `json:"id"`
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

type IndexSets struct {
	Total		int			`json:"total"`
	IndexSets	[]IndexSet  `json:"index_sets"`
}

type IndexSet struct {
	Id 		string	`json:"id"`
	Title 	string	`json:"title"`
}

type UserUpdate struct {
	roles	[]string
}

type Role struct {
	Name 			string		`json:"name"`
	Description 	string		`json:"description"`
	Permissions 	[]string	`json:"permissions"`
	ReadOnly		bool		`json:"read_only"`
}

func CreateStream(namespaceName string) {
	if !isGrayLogActive() { return }

	client := &http.DefaultClient
	requestObject := Stream{
		Title: namespaceName,
		Description: fmt.Sprintf("Logs for namespace %s", namespaceName),
		RemoveMatchesFromDefaultStream: true,
	}

	body, err := json.Marshal(requestObject)

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest("POST", getGraylogBaseUrl()+"/api/streams",  bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream creation. Error was: %s", err.Error()))
	}

	switch resp.StatusCode{
	case 200:
		var stream Stream
		content, err := ioutil.ReadAll(resp.Body)
		err := json.Unmarshal(content, stream)
		createRoleforStreamReaders(namespaceName, stream.Id)
	case 403:
		log.Println("Graylog communication failed due to permission denied for user.")
	}



}

func createRoleforStreamReaders(namespaceName, streamId string){
	if !isGrayLogActive() { return }

	client := &http.DefaultClient

	// TODO: Check if role is already present

	newRole := Role{
		Name: namespaceName+"_readers",
		Description: fmt.Sprintf("Role to allow users to read from stream %s", namespaceName),
		Permissions: []string{fmt.Sprintf("streams:read:%s", streamId)},
		ReadOnly: false,
	}

	body, err := json.Marshal(newRole)

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, getGraylogBaseUrl()+"/roles", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for RoleCreation. Error was: %s", err.Error()))
	}

	switch resp.StatusCode{
	case 200:
	case 403:
		log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
	}

}

func DeleteStream(namespaceName string) {

}

func GrantPermissionForStream(namespaceName, username string) {
	if !isGrayLogActive() { return }

	client := &http.DefaultClient

	/*
	TODO: Get current user roles
	TODO: Fetch Role to access stream for this namespace
	TODO: Merge Roles
	*/

	var currentRoles []string
	append(currentRoles, )
	userup := UserUpdate{roles: []string{roleName}}

	body, err := json.Marshal(userup)

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest(http.MethodPut, getGraylogBaseUrl()+"/user/"+username, bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for PermissionGrant on Stream. Error was: %s", err.Error()))
	}

	switch resp.StatusCode{
	case 200:
	case 400:
		log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
	}
}

func TakePermissionForStream(namespaceName, username string) {

}


