package webhooklistener

import (
	"net/http"
	"log"
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

	case "GET":


	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}