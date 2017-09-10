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
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/k8sclient"
	"time"
	"log"
	"fmt"
)

type GitlabEvent struct {
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	EventName            string    `json:"event_name"`
	Name                 string    `json:"name"`
	OwnerEmail           string    `json:"owner_email"`
	OwnerName            string    `json:"owner_name"`
	Path                 string    `json:"path"`
	PathWithNameSpace    string    `json:"path_with_namespace"`
	ProjectId            int       `json:"project_id"`
	ProjectVisibility    string    `json:"project_visibility"`
	OldPathWithNamespace string    `json:"old_path_with_namespace"`
	ProjectAccess        string    `json:"project_access"`
	GroupAccess          string    `json:"group_access"`
	UserEmail            string    `json:"user_email"`
	UserName             string    `json:"user_name"`
	UserUsername         string    `json:"user_username"`
	UserCreatedUserName  string    `json:"username"`
	UserId               int       `json:"user_id"`
	GroupId              int       `json:"group_id"`
	GroupName            string    `json:"group_name"`
	GroupPath            string    `json:"group_path"`
}

func HandleGitlabEvent(body []byte) {

	var event GitlabEvent
	err := json.Unmarshal(body, &event)
	if check(err) {
		return
	}

	switch event.EventName {

	// project operations

	case "project_create":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Creating Namespace for %s", event.PathWithNameSpace))
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	case "project_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Deleting Namespace for %s", event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.PathWithNameSpace)

	case "project_rename":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Rename: Deleting %s and Recreating Namespace for %s", event.OldPathWithNamespace, event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	case "project_transfer":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Transfer: Deleting %s and Recreating Namespace for %s", event.OldPathWithNamespace, event.PathWithNameSpace))
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	// project member operations

	case "user_add_to_team":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create RoleBinding for %s in %s as %s", event.UserUsername, event.PathWithNameSpace, event.ProjectAccess))
		k8sclient.CreateProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	case "user_remove_from_team":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete RoleBinding for %s in %s as %s", event.UserUsername, event.PathWithNameSpace, event.ProjectAccess))
		k8sclient.DeleteProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	// group operations

	case "group_create":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Creating Namespace for %s", event.Path))
		k8sclient.CreateNamespace(event.Path)

	case "group_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Deleting Namespace for %s", event.Path))
		k8sclient.DeleteNamespace(event.Path)

	// group member operations

	case "user_add_to_group":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create RoleBinding for %s in %s as %s", event.UserUsername, event.PathWithNameSpace, event.ProjectAccess))
		k8sclient.CreateGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	case "user_remove_from_group":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete RoleBinding for %s in %s as %s", event.UserUsername, event.PathWithNameSpace, event.ProjectAccess))
		k8sclient.DeleteGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	case "user_created":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Create Namespace and RoleBinding for %s in %s as %s", event.UserCreatedUserName, event.UserCreatedUserName, event.ProjectAccess))
		k8sclient.CreateNamespace(event.UserCreatedUserName)
		k8sclient.CreateGroupRoleBinding(event.UserCreatedUserName, event.UserCreatedUserName, "Master")

	case "user_destroy":
		log.Println(fmt.Sprintf("HOOK RECEIVED: Delete Namespace for %s", event.UserCreatedUserName))
		k8sclient.DeleteNamespace(event.UserCreatedUserName)
	}
}
