package gitlabclient

import (
	"os"
	"fmt"
	"log"
)

type GitlabGroup struct {
	Id       int
	FullPath string `json:"full_path"`
	Members  []Member
}

type GitlabProject struct {
	Id                int
	PathWithNameSpace string `json:"path_with_namespace"`
	Members           []Member
	Links             Links     `json:"_links"`
	Namespace         Namespace `json:"namespace"`
	Path			  string    `json:"path"`
}

type Namespace struct {
	Id       int
	Name     string
	Path     string
	Kind     string
	FullPath string
}

type Links struct {
	Members string
}

type GitlabUser struct {
	Username string `json:"username"`
}

type Member struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	State       string `json:"state"`
	AccessLevel int    `json:"access_level"`
}

type GitlabContent struct {
	Groups   []GitlabGroup
	Projects []GitlabProject
	Users    []GitlabUser
}


func contains(s []Member, e Member) bool {
	for _, a := range s {
		if a.Id == e.Id {
			return true
		}
	}
	return false
}

func check(err error) bool {
	if err != nil {
		log.Println("Error : ", err.Error())
		return true
	}
	return false
}


func getGitlabBaseUrl() string {
	apiVersion := os.Getenv("GITLAB_API_VERSION")
	if apiVersion == "" {
		apiVersion = "v4"
	}
	hostName := os.Getenv("GITLAB_HOSTNAME")
	if hostName == "" {
		log.Fatal("The GITLAB_HOSTNAME ENV has not been set!")
	}
	return fmt.Sprintf("https://%s/api/%s/", hostName, apiVersion)
}