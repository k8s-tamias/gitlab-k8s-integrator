package gitlabk8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"time"
)

type GitlabEvent struct {
	CreatedAt				time.Time 	`json:"created_at"`
	UpdatedAt				time.Time 	`json:"updated_at"`
	EventName   			string    	`json:"event_name"`
	Name					string	  	`json:"name"`
	OwnerEmail				string	  	`json:"owner_email"`
	OwnerName				string	  	`json:"owner_name"`
	Path					string		`json:"path"`
	PathWithNameSpace 		string		`json:"path_with_namespace"`
	ProjectId				int			`json:"project_id"`
	ProjectVisibility 		string		`json:"project_visibility"`
	OldPathWithNamespace 	string 		`json:"old_path_with_namespace"`
	ProjectAccess			string		`json:"project_access"`
	GroupAccess				string		`json:"group_access"`
	UserEmail				string		`json:"user_email"`
	UserName				string		`json:"user_name"`
	UserUsername			string		`json:"user_username"`
	UserId					int			`json:"user_id"`
	GroupId					int			`json:"group_id"`
	GroupName				string		`json:"group_name"`
	GroupPath				string		`json:"group_path"`
}


func getK8sClient() *kubernetes.Clientset {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if check(err) {
		log.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if check(err) {
		log.Fatal(err)
	}
	return clientset
}

func check(err error) bool {
	if err != nil {
		log.Println("Error : ", err.Error())
		return true
	}
	return false
}