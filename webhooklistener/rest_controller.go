/*
	Copyright 2017 by Christian Hüning (christianhuening@googlemail.com).

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package webhooklistener

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/k8s-tamias/gitlab-k8s-integrator/usecases"
)

func Listen(quit chan int) {
	router := http.NewServeMux()
	router.HandleFunc("/healthz", handleHealthz)
	if enableSyncEndpoint := os.Getenv("ENABLE_SYNC_ENDPOINT"); enableSyncEndpoint == "true" {
		log.Println("WARNING: Sync Endpoint enabled")
		router.HandleFunc("/sync", handleSync)
	}
	router.HandleFunc("/hook", handleGitlabWebhook)

	log.Fatal(http.ListenAndServe(":8080", router))
	quit <- 0
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		go usecases.PerformGlK8sSync()
		w.WriteHeader(202)
	}
}

// handleGitlabWebhook listens for the following events from the
// Gitlab System Webhooks Events: https://docs.gitlab.com/ce/system_hooks/system_hooks.html
func handleGitlabWebhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "POST":
		if r.Header.Get("X-Gitlab-Event") != "System Hook" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Printf("Received bad request from Gitlab. Problem was: X-Gitlab-Event Header was set to %s", r.Header.Get("X-Gitlab-Event"))
			return
		}
		// if GITLAB_SECRET_TOKEN env is set and is unequal to provided token, deny request
		if getGitlabSecretToken() != "" && r.Header.Get("X-Gitlab-Token") != getGitlabSecretToken() {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Printf("Received bad request from Gitlab. Problem was: X-Gitlab-Token (%s) didn't match stored secret token %s", r.Header.Get("X-Gitlab-Token"), getGitlabSecretToken())
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			HandleError(err, w, "Could not read body!", http.StatusBadRequest)
		}
		go usecases.HandleGitlabEvent(body)
		w.WriteHeader(http.StatusOK)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type ErrorMessage struct {
	Msg string
}

func HandleError(err error, w http.ResponseWriter, msg string, statusCode int) {
	log.Println("Error occurred! Err was: " + err.Error())
	w.WriteHeader(statusCode)
	if msg != "" {
		answer, _ := json.Marshal(ErrorMessage{msg + err.Error()})
		w.Write(answer)
	}
	return
}

func getGitlabSecretToken() string {
	return os.Getenv("GITLAB_SECRET_TOKEN")
}
