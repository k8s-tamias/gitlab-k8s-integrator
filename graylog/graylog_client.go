package graylog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Streams struct {
	Total      int      `json:"total"`
	StreamList []Stream `json:"streams"`
}

type Stream struct {
	Id                             string `json:"id"`
	Title                          string `json:"title"`
	Description                    string `json:"description"`
	Rules                          []Rule `json:"rules"`
	ContentPack                    string `json:"content_pack"`
	MatchingType                   string `json:"matching_type"`
	RemoveMatchesFromDefaultStream bool   `json:"remove_matches_from_default_stream"`
	IndexSetId                     string `json:"index_set_id"`
}

type StreamId struct {
	StreamId string `json:"stream_id"`
}

type StreamCreate struct {
	Title                          string `json:"title"`
	Description                    string `json:"description"`
	RemoveMatchesFromDefaultStream bool   `json:"remove_matches_from_default_stream"`
	IndexSetId                     string `json:"index_set_id"`
}

type Rules struct {
	Total 		int `json:"total"`
	StreamRules []Rule `json:"stream_rules"`
}

type Rule struct {
	Type        int    `json:"type"`
	Value       string `json:"value"`
	Field       string `json:"field"`
	Inverted    bool   `json:"inverted"`
	Description string `json:"description"`
}



type IndexSets struct {
	Total     int        `json:"total"`
	IndexSets []IndexSet `json:"index_sets"`
}

type IndexSet struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type UserUpdate struct {
	Roles []string `json:"roles"`
}

type User struct {
	Id       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	ReadOnly    bool     `json:"read_only"`
}

func CreateStream(namespaceName string) (bool, string) {
	if !isGrayLogActive() {
		return false, ""
	}


	if cond, id := isStreamAlreadyCreated(namespaceName); cond == true {
		return true, id
	}

	client := http.DefaultClient
	indexSetId := getIndexSetId()
	requestObject := StreamCreate{
		Title:                          namespaceName,
		Description:                    fmt.Sprintf("Logs for namespace %s", namespaceName),
		IndexSetId:                     indexSetId,
		RemoveMatchesFromDefaultStream: true,
	}

	body, err := json.Marshal(requestObject)

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, getGraylogBaseUrl()+"/api/streams", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream creation. Error was: %s", err.Error()))
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	switch resp.StatusCode {
	case 201:
		var stream StreamId

		err = json.Unmarshal(content, &stream)
		if err != nil {
			log.Fatal(err.Error())
		}

		createRuleForNamespace(namespaceName, stream.StreamId)

		createRoleForStreamReaders(namespaceName, stream.StreamId)
		// start stream
		startStream(stream.StreamId)
		return true, stream.StreamId
	case 403:
		log.Println("Graylog communication failed due to permission denied for user.")
		return false, ""
	default:
		log.Println(fmt.Sprintf("Graylog returned a not-OK status code when creating a stream. Code was: %d , message was: %s", resp.StatusCode, content))
		return false, ""
	}
}

func DeleteStream(namespaceName string) {
	if !isGrayLogActive() {
		return
	}

	stream, err := getStreamByNamespaceName(namespaceName)
	if err != nil {
		log.Println(fmt.Sprintf("An error occured while fetching information about the stream to be deleted. Error was: %s", err.Error()))
	}

	streamId := stream.Id

	client := http.DefaultClient

	req, err := http.NewRequest(http.MethodDelete, getGraylogBaseUrl()+fmt.Sprintf("/api/streams/%s", streamId), nil)
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream Start. Error was: %s", err.Error()))
	}

	switch resp.StatusCode {
	case 204:
		// stream deleted successfully, so delete role for it and reload local Stream cache
		deleteRoleForStreamReaders(namespaceName)
		reloadStreams()
	case 404:
		log.Println(fmt.Sprintf("Error while deleting stream: Stream %s could not be found", streamId))
	case 400:
		log.Println(fmt.Sprintf("Error while deleting stream: Stream %s was invalid", streamId))
	}
}

func GrantPermissionForStream(namespaceName, username string) bool {
	success := false
	if !isGrayLogActive() {
		return success
	}

	client := http.DefaultClient

	role, err := getRoleForNamespace(namespaceName)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of Role for Namespace. Error was: %s", err.Error()))
		return success
	}

	user, err := getGraylogUser(username)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of User. Error was: %s", err.Error()))
		return success
	}

	if contained, _ := contains(user.Roles, role.Name); contained == false {
		updatedRoles := append(user.Roles, role.Name)
		userup := UserUpdate{Roles: updatedRoles}

		body, err := json.Marshal(userup)

		if err != nil {
			log.Fatal(err.Error())
		}

		req, err := http.NewRequest(http.MethodPut, getGraylogBaseUrl()+"/api/users/"+username, bytes.NewBuffer(body))
		if err != nil {
			log.Fatal(err.Error())
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.SetBasicAuth(getGraylogSessionToken(), "session")

		resp, err := client.Do(req)
		if err != nil {
			log.Println(fmt.Sprintf("Error occured while calling Graylog for PermissionGrant on Stream. Error was: %s", err.Error()))
		}

		switch resp.StatusCode {
		case 204:
			success = true
		case 400:
			log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
			success = false

		case 404:
			success = false
		}
	} else {
		success = true
	}

	return success
}

func TakePermissionForStream(namespaceName, username string) bool {
	success := false
	if !isGrayLogActive() {
		return success
	}

	client := http.DefaultClient

	user, err := getGraylogUser(username)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of User. Error was: %s", err.Error()))
		return success
	}

	if contained, index := contains(user.Roles, getRoleNameForNamespace(namespaceName)); contained == true {
		// remove role from Roles slice
		updatedRoles := append(user.Roles[:index], user.Roles[index+1:]...)
		userup := UserUpdate{Roles: updatedRoles}

		body, err := json.Marshal(userup)

		if err != nil {
			log.Fatal(err.Error())
		}

		req, err := http.NewRequest(http.MethodPut, getGraylogBaseUrl()+"/api/users/"+username, bytes.NewBuffer(body))
		if err != nil {
			log.Fatal(err.Error())
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.SetBasicAuth(getGraylogSessionToken(), "session")

		resp, err := client.Do(req)
		if err != nil {
			log.Println(fmt.Sprintf("Error occured while calling Graylog for PermissionGrant on Stream. Error was: %s", err.Error()))
		}

		switch resp.StatusCode {
		case 204:
			success = true
		case 400:
			log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
			success = false
		}
	} else {
		success = true
	}
	return success
}

