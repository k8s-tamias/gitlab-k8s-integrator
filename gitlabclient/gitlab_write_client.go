package gitlabclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func SetupK8sIntegrationForGitlabProject(projectId, namespace, token string) {
	k8sUrl := os.Getenv("EXTERNAL_K8S_API_URL")
	if k8sUrl == "" {
		// abort if K8S_API_URL was not set
		log.Println("K8S_API_URL was not set, skipping setup of K8s integration in Gitlab...")
		return
	}

	url := fmt.Sprintf("%sprojects/%s/services/kubernetes", getGitlabBaseUrl(), projectId)

	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	q := req.URL.Query()
	q.Add("token", token)
	q.Add("namespace", namespace)
	q.Add("api_url", k8sUrl)

	caPem := os.Getenv("K8S_CA_PEM")
	if caPem != "" {
		q.Add("ca_pem", caPem)
	}

	req.URL.RawQuery = q.Encode()

	req.Header.Add("Private-Token", os.Getenv("GITLAB_PRIVATE_TOKEN"))
	req.Header.Add("Sudo", "root")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println(fmt.Sprintf("Could not set up Kubernetes Integration for project %s . Err was: %s ", projectId, err.Error()))
	}

	if resp.StatusCode != http.StatusOK {
		msg := ""
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			msg = string(body[:])

		}
		log.Println(fmt.Sprintf("Setting up Kubernetes Integration for project %s failed with errorCode %d and message %s", projectId, resp.StatusCode, msg))
	}

	setupEnvironment(projectId)
}

type ErrorMessage struct {
	Message Msg
}

type Msg struct {
	Name []string
	Slug []string
}

func setupEnvironment(projectId string) {
	envName := os.Getenv("GITLAB_ENVIRONMENT_NAME")
	if envName == "" {
		// abort if GITLAB_ENVIRONMENT_NAME was not set
		log.Println("GITLAB_ENVIRONMENT_NAME was not set, skipping creation of environment in Gitlab...")
		return
	}

	url := fmt.Sprintf("%sprojects/%s/environments", getGitlabBaseUrl(), projectId)
	values := map[string]string{"id": projectId, "name": envName}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		log.Fatalln(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return

	case http.StatusBadRequest:
		var msg ErrorMessage
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(body, &msg)
		if len(msg.Message.Name) > 0 && msg.Message.Name[0] == "has already been taken" {
			return
		}
		log.Println(fmt.Sprintf("Creation of environment failed with http error %d, projectID was: %s", resp.StatusCode, projectId))
	default:
		log.Println(fmt.Sprintf("Creation of environment failed with http error %d, projectID was: %s", resp.StatusCode, projectId))
	}
}
