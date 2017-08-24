package webhooklistener

import (
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/gitlabk8s"
)

func Listen(quit chan int) {
	router := http.NewServeMux()
	router.HandleFunc("/healthz", handleHealthz)
	router.HandleFunc("/", handleGitlabWebhook)

	log.Fatal(http.ListenAndServe(":8080", router))
	quit <- 0
}

// handleGitlabWebhook listens for the following events from the
// Gitlab System Webhooks Events: https://docs.gitlab.com/ce/system_hooks/system_hooks.html
func handleGitlabWebhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "POST":
		if r.Header.Get("X-Gitlab-Event") != "System Hook" {
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			HandleError(err, w, "Could not read body!", http.StatusBadRequest)
		}
		go gitlabk8s.HandleGitlabEvent(body)
		w.WriteHeader(http.StatusOK)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request){
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
