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

var grayLogStreams Streams

const streamNotPresentMsg = "Stream not present in Graylog!"

/*
	AUTHN & AUTHZ RELATED
*/

func createRoleForStreamReaders(namespaceName, streamId string) {
	if !isGrayLogActive() || isRoleAlreadyPresent(namespaceName) {
		log.Println(fmt.Sprintf("Readers role for names %s is already present, skipping.", namespaceName))
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
	req.Header.Add("Accept", "application/json")
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
	case 201:

	case 403:
		log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
	default:
		log.Println(fmt.Sprintf("Graylog returned a not-OK status code when creating a role for a stream. Code was: %d , message was: %s", resp.StatusCode, content))
	}

}

func deleteRoleForStreamReaders(namespaceName string) {
	if !isGrayLogActive() || !isRoleAlreadyPresent(namespaceName) {
		log.Println(fmt.Sprintf("Readers role for names %s is already deleted, skipping.", namespaceName))
		return
	}

	client := http.DefaultClient

	if err != nil {
		log.Fatal(err.Error())
	}

	req, err := http.NewRequest(http.MethodDelete, getGraylogBaseUrl()+"/api/roles/"+getRoleNameForNamespace(namespaceName), nil)
	if err != nil {
		log.Fatal(err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
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
	case 204:
		// success
	case 403:
		log.Println("Graylog communication for PermissionGrant on Stream failed due to permission denied for user.")
	default:
		log.Println(fmt.Sprintf("Graylog returned a not-OK status code when creating a role for a stream. Code was: %d , message was: %s", resp.StatusCode, content))
	}

}

func isRoleAlreadyPresent(namespaceName string) bool {
	res := false

	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/api/roles/"+getRoleNameForNamespace(namespaceName), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
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
	req.Header.Add("Accept", "application/json")
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

func getGraylogUser(username string) (*User, error) {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/api/users/"+username, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
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

		// TODO: CREATE USER !!!

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

/*
	STREAM RELATED
*/

func getStreamByNamespaceName(namespaceName string) (*Stream, error) {
	if contained, index := containsStream(grayLogStreams.StreamList, namespaceName); contained == true {
		return &grayLogStreams.StreamList[index], nil
	} else {

		reloadStreams()

		if contained, index := containsStream(grayLogStreams.StreamList, namespaceName); contained == true {
			return &grayLogStreams.StreamList[index], nil
		}
	}
	return nil, errors.New(streamNotPresentMsg)
}

func reloadStreams() {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, getGraylogBaseUrl()+"/api/streams", nil)
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

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal(content, &grayLogStreams)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func isStreamAlreadyCreated(namespaceName string) (bool, string) {
	result := false
	id := ""
	s, err := getStreamByNamespaceName(namespaceName)
	if err != nil {
		if err.Error() != streamNotPresentMsg {
			log.Fatal(fmt.Sprintf("An error occured while communication with Graylog to determine whether a Stream is already present. Error was: %s", err.Error()))
		}
	}
	if err == nil && s.Id != "" {
		result = true
		id = s.Id
	}
	return result, id
}

func startStream(streamId string) {
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodPost, getGraylogBaseUrl()+fmt.Sprintf("/api/streams/%s/resume", streamId), nil)
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
	case 404:
		log.Println(fmt.Sprintf("Error while starting stream: Stream %s could not be found", streamId))
	case 400:
		log.Println(fmt.Sprintf("Error while starting stream: Stream %s was missing or invalid", streamId))
	}
}