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
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/usecases"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func Listen(quit chan int) {
	router := http.NewServeMux()
	router.HandleFunc("/healthz", handleHealthz)
	router.HandleFunc("/", handleGitlabWebhook)
	if enableSyncEndpoint := os.Getenv("ENABLE_SYNC_ENDPOINT"); enableSyncEndpoint == "true" {
		log.Println("WARNING: Sync Endpoint enabled")
		router.HandleFunc("/sync", handleSync)
	}

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
