package gitlabclient

import (
	"encoding/json"
	"fmt"
	"github.com/peterhellberg/link"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type GitlabGroup struct {
	FullPath string `json:"full_path"`
}

type GitlabProject struct {
	PathWithNameSpace string `json:"path_with_namespace"`
}

type GitlabUser struct {
	Username string `json:"username"`
}

type GitlabContent struct {
	Groups   []GitlabGroup
	Projects []GitlabProject
	Users    []GitlabUser
}

func GetFullGitlabContent() (GitlabContent, error) {
	groupUrl := getGitlabBaseUrl() + "groups"
	foundGroups, err := GetAllGroups(make([]GitlabGroup, 0), groupUrl)
	if check(err) {
		log.Fatal(err.Error())
	}
	projectUrl := getGitlabBaseUrl() + "projects"
	foundProjects, err := GetAllProjects(make([]GitlabProject, 0), projectUrl)
	if check(err) {
		log.Fatal(err.Error())
	}
	userUrl := getGitlabBaseUrl() + "users"
	foundUsers, err := GetAllUsers(make([]GitlabUser, 0), userUrl)
	if check(err) {
		log.Fatal(err.Error())
	}
	return GitlabContent{Groups: foundGroups, Projects: foundProjects, Users: foundUsers}, nil
}

func getGitlabBaseUrl() string {
	return fmt.Sprintf("https://%s/api/%s/", os.Getenv("GITLAB_HOSTNAME"), os.Getenv("GITLAB_API_VERSION"))
}

func performGitlabHTTPRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if check(err) {
		log.Fatal("Fatal Error while creating new HTTP Request! Err:" + err.Error())
	}

	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))
	req.Header.Add("SUDO", os.Getenv("GITLAB_ADMIN_USER"))
	result, err := http.DefaultClient.Do(req)
	return result, err

}

func GetAllGroups(gitlabGroups []GitlabGroup, url string) ([]GitlabGroup, error) {
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
	}
	if result.StatusCode == 401 {
		return nil, errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now.")
	}
	content, err := ioutil.ReadAll(result.Body)

	groups := make([]GitlabGroup, 0)

	json.Unmarshal(content, &groups)

	gitlabGroups = append(groups, gitlabGroups...)

	group := link.ParseHeader(result.Header)
	next := group["next"]
	if next != nil {
		finalGroups, err := GetAllGroups(gitlabGroups, next.URI)
		if err != nil {
			return nil, err
		}
		gitlabGroups = finalGroups
	}
	return gitlabGroups, nil
}

func GetAllProjects(gitlabProjects []GitlabProject, url string) ([]GitlabProject, error) {
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
	}
	if result.StatusCode == 401 {
		return nil, errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now.")
	}
	content, err := ioutil.ReadAll(result.Body)

	projects := make([]GitlabProject, 0)

	json.Unmarshal(content, &projects)

	gitlabProjects = append(projects, gitlabProjects...)

	group := link.ParseHeader(result.Header)
	next := group["next"]
	if next != nil {
		finalProjects, err := GetAllProjects(gitlabProjects, next.URI)
		if err != nil {
			return nil, err
		}
		gitlabProjects = finalProjects
	}
	return gitlabProjects, nil
}

func GetAllUsers(gitlabUsers []GitlabUser, url string) ([]GitlabUser, error) {
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
	}
	if result.StatusCode == 401 {
		return nil, errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now.")
	}
	content, err := ioutil.ReadAll(result.Body)

	Users := make([]GitlabUser, 0)

	json.Unmarshal(content, &Users)

	gitlabUsers = append(Users, gitlabUsers...)

	group := link.ParseHeader(result.Header)
	next := group["next"]
	if next != nil {
		finalUsers, err := GetAllUsers(gitlabUsers, next.URI)
		if err != nil {
			return nil, err
		}
		gitlabUsers = finalUsers
	}
	return gitlabUsers, nil
}

func check(err error) bool {
	if err != nil {
		log.Println("Error : ", err.Error())
		return true
	}
	return false
}