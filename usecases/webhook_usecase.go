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

package usecases

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/k8s-tamias/gitlab-k8s-integrator/gitlabclient"
	"github.com/k8s-tamias/gitlab-k8s-integrator/k8sclient"
)

type GitlabEvent struct {
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
	EventName                string    `json:"event_name"`
	Name                     string    `json:"name"`
	OwnerEmail               string    `json:"owner_email"`
	OwnerName                string    `json:"owner_name"`
	Path                     string    `json:"path"`
	PathWithNameSpace        string    `json:"path_with_namespace"`
	ProjectPathWithNameSpace string    `json:"project_path_with_namespace"`
	ProjectId                int       `json:"project_id"`
	ProjectVisibility        string    `json:"project_visibility"`
	OldPathWithNamespace     string    `json:"old_path_with_namespace"`
	ProjectAccess            string    `json:"access_level"`
	GroupAccess              string    `json:"group_access"`
	UserEmail                string    `json:"user_email"`
	UserName                 string    `json:"user_name"`
	UserUsername             string    `json:"user_username"`
	UserCreatedUserName      string    `json:"username"`
	UserId                   int       `json:"user_id"`
	GroupId                  int       `json:"group_id"`
	GroupName                string    `json:"group_name"`
	GroupPath                string    `json:"group_path"`
}

func HandleGitlabEvent(body []byte) {

	if os.Getenv("ENABLE_GITLAB_HOOKS_DEBUG") == "true" {
		rawMsg := string(body[:])
		log.Println(fmt.Sprintf("DEBUG: Raw Hook Contents Received= %s", rawMsg))
	}

	var event GitlabEvent
	err := json.Unmarshal(body, &event)
	if check(err) {
		return
	}

	switch event.EventName {

	// project operations

	case "project_create":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Creating Namespace for %s", event.PathWithNameSpace))
		createdNs := k8sclient.CreateNamespace(event.PathWithNameSpace)
		sai, _, err := k8sclient.CreateServiceAccountAndRoleBinding(event.PathWithNameSpace)
		if err != nil {
			log.Printf("Creation of ServiceAccount and RoleBinding failed for project %s", event.Name)
		} else {
			gitlabclient.SetupK8sIntegrationForGitlabProject(strconv.Itoa(event.ProjectId), createdNs, sai.Token)
		}
	case "project_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Deleting Namespace for %s", event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.PathWithNameSpace)
	case "project_rename":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Rename: Deleting %s and Recreating Namespace for %s", event.OldPathWithNamespace, event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		createdNs := k8sclient.CreateNamespace(event.PathWithNameSpace)
		sai, _, err := k8sclient.CreateServiceAccountAndRoleBinding(event.PathWithNameSpace)
		if err != nil {
			log.Printf("Creation of ServiceAccount and RoleBinding failed for project %s", event.Name)
		} else {
			gitlabclient.SetupK8sIntegrationForGitlabProject(strconv.Itoa(event.ProjectId), createdNs, sai.Token)
		}
	case "project_transfer":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Transfer: Deleting %s and Recreating Namespace for %s", event.OldPathWithNamespace, event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		createdNs := k8sclient.CreateNamespace(event.PathWithNameSpace)
		sai, _, err := k8sclient.CreateServiceAccountAndRoleBinding(event.PathWithNameSpace)
		if err != nil {
			log.Printf("Creation of ServiceAccount and RoleBinding failed for project %s", event.Name)
		} else {
			gitlabclient.SetupK8sIntegrationForGitlabProject(strconv.Itoa(event.ProjectId), createdNs, sai.Token)
		}
		// project member operations

	case "user_add_to_team":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create RoleBinding for %s in %s as %s", event.UserUsername, event.ProjectPathWithNameSpace, event.ProjectAccess))
		k8sclient.CreateProjectRoleBinding(event.UserUsername, event.ProjectPathWithNameSpace, event.ProjectAccess)
		k8sclient.GetActualNameSpaceNameByGitlabName(event.ProjectPathWithNameSpace)
	case "user_remove_from_team":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete RoleBinding for %s in %s as %s", event.UserUsername, event.ProjectPathWithNameSpace, event.ProjectAccess))
		k8sclient.DeleteProjectRoleBinding(event.UserUsername, event.ProjectPathWithNameSpace, event.ProjectAccess)
		k8sclient.GetActualNameSpaceNameByGitlabName(event.ProjectPathWithNameSpace)

		// group operations
	case "group_create":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Creating Namespace for %s", event.Path))
		k8sclient.CreateNamespace(event.Path)

	case "group_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Deleting Namespace for %s", event.Path))
		k8sclient.DeleteNamespace(event.Path)

		// group member operations

	case "user_add_to_group":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create RoleBinding for %s in %s as %s", event.UserUsername, event.GroupPath, event.GroupAccess))
		k8sclient.CreateGroupRoleBinding(event.UserUsername, event.GroupPath, event.GroupAccess)
		k8sclient.GetActualNameSpaceNameByGitlabName(event.GroupPath)

	case "user_remove_from_group":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete RoleBinding for %s in %s as %s", event.UserUsername, event.GroupPath, event.GroupAccess))
		k8sclient.DeleteGroupRoleBinding(event.UserUsername, event.GroupPath, event.GroupAccess)
		k8sclient.GetActualNameSpaceNameByGitlabName(event.GroupPath)

	case "user_create":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create Namespace and RoleBinding for %s in %s as %s", event.UserCreatedUserName, event.UserCreatedUserName, event.ProjectAccess))
		k8sclient.CreateNamespace(event.UserCreatedUserName)
		k8sclient.CreateGroupRoleBinding(event.UserCreatedUserName, event.UserCreatedUserName, "Master")
		k8sclient.GetActualNameSpaceNameByGitlabName(event.UserCreatedUserName)

	case "user_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete Namespace for %s", event.UserCreatedUserName))
		k8sclient.DeleteNamespace(event.UserCreatedUserName)
		k8sclient.DeleteGroupRoleBinding(event.UserCreatedUserName, event.UserCreatedUserName, "Master")
		k8sclient.GetActualNameSpaceNameByGitlabName(event.UserCreatedUserName)

	default:
		log.Println(fmt.Sprintf("HOOK RECEIVED: Unknown Hook Type. Type was: %s", event.EventName))
	}
}
