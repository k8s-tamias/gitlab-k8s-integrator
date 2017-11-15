package gitlabclient

import (
	"net/http"
	"log"
	"os"
	"fmt"
)

func SetupK8sIntegrationForGitlabProject(projectId, namespace, token string) {
	k8sUrl := os.Getenv("K8S_API_URL")
	if k8sUrl == "" {
		// abort if K8S_API_URL was not set
		log.Println("K8S_API_URL was not set, skipping setup of K8s integration in Gitlab...")
		return
	}

	url := fmt.Sprintf("%sprojects/%s/services/kubernetes",getGitlabBaseUrl(),projectId)


	if isK8sIntegrationSetup(url) {
		return
	}


	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	q := req.URL.Query()
	q.Add("token",token)
	q.Add("namespace", namespace)
	q.Add("api_url", k8sUrl)

	caPem := os.Getenv("K8S_CA_PEM")
	if caPem != "" {
		q.Add("ca_pem", caPem)
	}

	req.URL.RawQuery = q.Encode()

	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println(fmt.Sprintf("Could not set up Kubernetes Integration for project %s . Err was: %s ", projectId, err))
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(fmt.Sprintf("Setting up Kubernetes Integration for project %s failed with errorCode %d", projectId, resp.StatusCode))
	}
}
func isK8sIntegrationSetup(url string) bool {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))
	return false
	//resp, err := http.DefaultClient.Do(req)
	// TODO: Check if kubernetes integration is already setup!
}
