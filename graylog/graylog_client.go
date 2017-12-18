package graylog

import (
	"bytes"
	"encoding/json"
	"errors"
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
	StreamId		string `json:"stream_id"`
}

type StreamCreate struct {
	Title                          string `json:"title"`
	Description                    string `json:"description"`
	RemoveMatchesFromDefaultStream bool   `json:"remove_matches_from_default_stream"`
	IndexSetId                     string `json:"index_set_id"`
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
	roles []string
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

func CreateStream(namespaceName string) bool {
	if !isGrayLogActive() {
		return false
	}

	if isStreamAlreadyCreated(namespaceName) {
		return true
	}

	client := http.DefaultClient
	indexSetId := getIndexSetId()
	requestObject := StreamCreate{
		Title:                          namespaceName,
		Description:                    fmt.Sprintf("Logs for namespace %s", namespaceName),
		IndexSetId:						indexSetId,
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

		createRoleforStreamReaders(namespaceName, stream.StreamId)
		// start stream
		startStream(stream.StreamId)
		return true
	case 403:
		log.Println("Graylog communication failed due to permission denied for user.")
		return false
	default:
		log.Println(fmt.Sprintf("Graylog returned a not-OK status code when creating a stream. Code was: %d , message was: %s", resp.StatusCode,content))
		return false
	}
}

func createRoleforStreamReaders(namespaceName, streamId string) {
	if !isGrayLogActive() || roleIsAlreadyPresent(namespaceName) {
		return
	}

	client := http.DefaultClient

	newRole := Role{
		Name:        getRoleNameForNamespace(namespaceName),
		Description: fmt.Sprintf("Role to allow users to read from stream %s", namespaceName),
		Permissions: []string{fmt.Sprintf("streams:read:%s", streamId)},
		ReadOnly:    false,
	}

	body, err := json.Marshal(newRole)

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, getGraylogBaseUrl()+"/api/roles", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for RoleCreation. Error was: %s", err.Error()))
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	switch resp.StatusCode {
	case 200:

	case 403:
		log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
	default:
		log.Println(fmt.Sprintf("Graylog returned a not-OK status code when creating a role for a stream. Code was: %d , message was: %s", resp.StatusCode, content))
	}

}

func startStream(streamId string) {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodPost, getGraylogBaseUrl()+fmt.Sprintf("/api/streams/%s/resume", streamId), nil)
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream Start. Error was: %s", err.Error()))
	}

	switch resp.StatusCode {
	case 404:
		log.Println(fmt.Sprintf("Error while starting stream: Stream %s could not be found", streamId))
	case 400:
		log.Println(fmt.Sprintf("Error while starting stream: Stream %s was missing or invalid", streamId))
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
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream Start. Error was: %s", err.Error()))
	}

	switch resp.StatusCode {
	case 404:
		log.Println(fmt.Sprintf("Error while deleting stream: Stream %s could not be found", streamId))
	case 400:
		log.Println(fmt.Sprintf("Error while deleting stream: Stream %s was invalid", streamId))
	}
}

var grayLogStreams Streams
const streamNotPresentMsg = "Stream not present in Graylog!"

func getStreamByNamespaceName(namespaceName string) (*Stream, error) {
	if contained, index := containsStream(grayLogStreams.StreamList, namespaceName); contained == true {
		return &grayLogStreams.StreamList[index], nil
	} else {
		client := http.DefaultClient
		req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/api/streams", nil)
		if err != nil {
			log.Fatal(err.Error())
		}

		req.Header.Add("Content-Type", "application/json")
		req.SetBasicAuth(getGraylogSessionToken(), "session")

		resp, err := client.Do(req)
		if err != nil {
			log.Println(fmt.Sprintf("Error occured while calling Graylog for Stream Start. Error was: %s", err.Error()))
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err.Error())
		}

		err = json.Unmarshal(content, &grayLogStreams)
		if err != nil {
			log.Fatal(err.Error())
		}

		if contained, index := containsStream(grayLogStreams.StreamList, namespaceName); contained == true {
			return &grayLogStreams.StreamList[index], nil
		}
	}
	return nil, errors.New(streamNotPresentMsg)
}

func isStreamAlreadyCreated(namespaceName string) bool {
	result := false
	s, err := getStreamByNamespaceName(namespaceName)
	if err != nil {
		if err.Error() != streamNotPresentMsg {
			log.Fatal(fmt.Sprintf("An error occured while communication with Graylog to determine whether a Stream is already present. Error was: %s", err.Error()))
		}
	}
	if s != nil && err == nil { result = true }
	return result
}

func containsStream(s []Stream, title string) (bool, int) {
	for i, a := range s {
		if a.Title == title {
			return true, i
		}
	}
	return false, 0
}

func GrantPermissionForStream(namespaceName, username string) {
	if !isGrayLogActive() {
		return
	}

	client := http.DefaultClient

	role, err := getRoleForNamespace(namespaceName)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of Role for Namespace. Error was: %s", err.Error()))
		return
	}

	user, err := getGraylogUser(username)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of User. Error was: %s", err.Error()))
		return
	}

	if contained, _ := contains(user.Roles, role.Name); contained == false {
		updatedRoles := append(user.Roles, role.Name)
		userup := UserUpdate{roles: updatedRoles}

		body, err := json.Marshal(userup)

		if err != nil {
			log.Fatal(err.Error())
		}

		req, err := http.NewRequest(http.MethodPut, getGraylogBaseUrl()+"/users/"+username, bytes.NewBuffer(body))
		if err != nil {
			log.Fatal(err.Error())
		}

		req.Header.Add("Content-Type", "application/json")
		req.SetBasicAuth(getGraylogSessionToken(), "session")

		resp, err := client.Do(req)
		if err != nil {
			log.Println(fmt.Sprintf("Error occured while calling Graylog for PermissionGrant on Stream. Error was: %s", err.Error()))
		}

		switch resp.StatusCode {
		case 200:
		case 400:
			log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
		}
	}
}

func TakePermissionForStream(namespaceName, username string) {
	if !isGrayLogActive() {
		return
	}

	client := http.DefaultClient

	user, err := getGraylogUser(username)
	if err != nil {
		log.Println(fmt.Sprintf("Error occured while calling Graylog for retrieval of User. Error was: %s", err.Error()))
		return
	}

	if contained, index := contains(user.Roles, getRoleNameForNamespace(namespaceName)); contained == true {
		// remove role from roles slice
		updatedRoles := append(user.Roles[:index], user.Roles[index+1:]...)
		userup := UserUpdate{roles: updatedRoles}

		body, err := json.Marshal(userup)

		if err != nil {
			log.Fatal(err.Error())
		}

		req, err := http.NewRequest(http.MethodPut, getGraylogBaseUrl()+"/users/"+username, bytes.NewBuffer(body))
		if err != nil {
			log.Fatal(err.Error())
		}

		req.Header.Add("Content-Type", "application/json")
		req.SetBasicAuth(getGraylogSessionToken(), "session")

		resp, err := client.Do(req)
		if err != nil {
			log.Println(fmt.Sprintf("Error occured while calling Graylog for PermissionGrant on Stream. Error was: %s", err.Error()))
		}

		switch resp.StatusCode {
		case 200:
		case 400:
			log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
		}
	}
}

func roleIsAlreadyPresent(namespaceName string) bool {
	res := false

	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl() + "/api/roles/" + getRoleNameForNamespace(namespaceName), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
	switch resp.StatusCode {
	case 200:
		res = true
	case 404:
		res = false
	default:
		log.Fatal(fmt.Sprintf("Query for Role failed with error. Statuscode was %s, message was: %s", resp.StatusCode, content))
	}
	return res
}

func getRoleForNamespace(namespaceName string) (*Role, error) {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/api/roles/"+getRoleNameForNamespace(namespaceName), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	roleResp, err := client.Do(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	switch roleResp.StatusCode {
	case 200:
		roleBody, err := ioutil.ReadAll(roleResp.Body)
		if err != nil {
			log.Fatal(err.Error())
		}
		var role Role
		err = json.Unmarshal(roleBody, &role)
		if err != nil {
			log.Fatal(err.Error())
		}
		return &role, nil

	case 404:
		errMsg := fmt.Sprintf("Error role for namespace %s is not present. Please re-sync!", namespaceName)
		log.Println(errMsg)
		return nil, errors.New(errMsg)

	case 403:
		errMsg := fmt.Sprintf("Error permission was denied to fetch role for namespace %s. Please check with an admin!", namespaceName)
		log.Println(errMsg)
		return nil, errors.New(errMsg)

	default:
		log.Println(fmt.Sprintf("An unknown returncode was received when fetching Role for Namespace %s", namespaceName))
		return nil, nil
	}
}

func getRoleNameForNamespace(namespaceName string) string {
	return namespaceName + "_readers"
}

func contains(s []string, e string) (bool, int) {
	for i, a := range s {
		if a == e {
			return true, i
		}
	}
	return false, 0
}

func getGraylogUser(username string) (*User, error) {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/users/"+username, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(getGraylogSessionToken(), "session")

	roleResp, err := client.Do(req)
	if err != nil {
		log.Fatal(err.Error())
	}

	switch roleResp.StatusCode {
	case 200:
		roleBody, err := ioutil.ReadAll(roleResp.Body)
		if err != nil {
			log.Fatal(err.Error())
		}
		var user User
		err = json.Unmarshal(roleBody, &user)
		if err != nil {
			log.Fatal(err.Error())
		}
		return &user, nil

	case 404:
		errMsg := fmt.Sprintf("Error user %s is not present. Please re-sync!", username)
		log.Println(errMsg)
		return nil, errors.New(errMsg)

	case 403:
		errMsg := fmt.Sprintf("Error permission was denied to fetch user %s. Please check with an admin!", username)
		log.Println(errMsg)
		return nil, errors.New(errMsg)

	default:
		log.Println(fmt.Sprintf("An unknown returncode was received when fetching User %s", username))
		return nil, nil
	}
}
