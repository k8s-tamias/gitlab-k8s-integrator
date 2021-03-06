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
package gitlabclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/peterhellberg/link"
	"github.com/pkg/errors"
)

func GetFullGitlabContent() (*GitlabContent, error) {
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
	return &GitlabContent{Groups: foundGroups, Projects: foundProjects, Users: foundUsers}, nil
}

func GetAllGroups(gitlabGroups []GitlabGroup, url string) ([]GitlabGroup, error) {
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
		return nil, err
	}
	if result.StatusCode == 401 {
		return nil, errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now")
	}
	content, err := ioutil.ReadAll(result.Body)

	groups := make([]GitlabGroup, 0)

	err = json.Unmarshal(content, &groups)
	if check(err) {
		return nil, err
	}

	for i := range groups {
		err := groups[i].getMembers()
		check(err)
	}

	// DEEP APPEND here
	gitlabGroups = append(gitlabGroups, groups...)

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

	err = json.Unmarshal(content, &projects)

	if check(err) {
		return nil, err
	}

	for i := range projects {
		err := projects[i].getMembers()
		check(err)
	}

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

	err = json.Unmarshal(content, &Users)
	if check(err) {
		return nil, err
	}

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

/*
From: https://docs.gitlab.com/ee/api/members.html
10 => Guest access
20 => Reporter access
30 => Developer access
40 => Maintainer access
50 => Owner access # Only valid for groups
*/
func TranslateIntAccessLevels(lvl int) string {
	level := "default"
	switch lvl {
	case 20:
		level = "Reporter"
	case 30:
		level = "Developer"
	case 40:
		level = "Master"
	case 50:
		level = "Master" // owner has same rights in k8s
	}
	return level
}

func (g *GitlabGroup) getMembers() error {
	url := getGitlabBaseUrl() + "groups/" + strconv.Itoa(g.Id) + "/members"
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
		return err
	}
	if result.StatusCode == 401 {
		return errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now.")
	}
	if result.StatusCode == 404 {
		return errors.New("The requested URL was invalid! Stopping now.")
	}

	content, err := ioutil.ReadAll(result.Body)

	members := make([]Member, 0)
	err = json.Unmarshal(content, &members)

	if check(err) {
		return err
	}

	g.Members = members
	if len(g.Members) == 0 {
		log.Println(fmt.Sprintf("WARNING: No Group Members were found for group %s . StatusCode of Request was: %d . This is a potential bug in Gitlab, will continue to sync anyway", g.FullPath, result.StatusCode))
	}
	return nil
}

func (p *GitlabProject) getMembers() error {
	url := getGitlabBaseUrl() + "projects/" + strconv.Itoa(p.Id) + "/members"
	result, err := performGitlabHTTPRequest(url)

	if check(err) {
		log.Println("Error occured while calling Gitlab! Cancelling Sync! Err:" + err.Error())
		return err
	}
	if result.StatusCode == 401 {
		return errors.New("GITLAB_PRIVATE_TOKEN was not set or wrong. Stopping now.")
	}
	if result.StatusCode == 404 {
		return errors.New("The requested URL was invalid! Stopping now. Url was: " + url)
	}

	content, err := ioutil.ReadAll(result.Body)

	members := make([]Member, 0)
	err = json.Unmarshal(content, &members)
	if check(err) {
		return err
	}

	p.Members = members

	// aggregate with members from parent group(s)
	if p.Namespace.Kind == "group" {
		glGroup := GitlabGroup{FullPath: p.Namespace.FullPath, Id: p.Namespace.Id}
		err := glGroup.getMembers()
		if check(err) {
			return err
		}
		// merge with project members
		for _, gm := range glGroup.Members {
			if !contains(p.Members, gm) {
				p.Members = append(p.Members, gm)
			}
		}
	}

	if len(p.Members) == 0 {
		log.Println(fmt.Sprintf("WARNING: No Project Members were found for project %s . StatusCode of Request was: %d . This is a potential bug in Gitlab, will continue to sync anyway", p.PathWithNameSpace, result.StatusCode))
	}

	return nil
}

func performGitlabHTTPRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if check(err) {
		log.Fatal("Fatal Error while creating new HTTP Request! Err:" + err.Error())
	}

	req.Header.Add("PRIVATE-TOKEN", os.Getenv("GITLAB_PRIVATE_TOKEN"))
	result, err := http.DefaultClient.Do(req)
	return result, err

}
