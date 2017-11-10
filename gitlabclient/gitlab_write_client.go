package gitlabclient

import (
	"net/http"
	"log"
	"os"
	"fmt"
)

func WriteTokenToGitlab(projectId, namespace, token string) {
	url := fmt.Sprintf("%sprojects/%s/services/kubernetes",getGitlabBaseUrl(),projectId)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	k8sUrl := os.Getenv("K8S_API_URL")

	q := req.URL.Query()
	q.Add("token",token)
	q.Add("namespace", namespace)
	if k8sUrl != "" {
		q.Add("api_url", k8sUrl)
	}

	req.URL.RawQuery = q.Encode()

	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println(fmt.Sprintf("Could not set up Kubernetes Integration for project %s . Err was: %s ", projectId, err))
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(fmt.Sprintf("Setting up Kubernetes Integration for project %s failed with errorCode %d", projectId, resp.StatusCode))
	} else {
		log.Println(fmt.Sprintf("Setting up Kubernetes Integration for project %s was succesful!", projectId))
	}
}


