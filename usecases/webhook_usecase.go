package usecases

import (
	"encoding/json"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/k8sclient"
	"log"
	"time"
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
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	case "project_destroy":
		k8sclient.DeleteNamespace(event.PathWithNameSpace)

	case "project_rename":
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	case "project_transferred":
		k8sclient.DeleteNamespace(event.OldPathWithNamespace)
		k8sclient.CreateNamespace(event.PathWithNameSpace)

	// project member operations

	case "user_add_to_team":
		k8sclient.CreateProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	case "user_remove_from_team":
		k8sclient.DeleteProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	// group operations

	case "group_create":
		k8sclient.CreateNamespace(event.Path)

	case "group_destroy":
		k8sclient.DeleteNamespace(event.PathWithNameSpace)

	// group member operations

	case "user_add_to_group":
		k8sclient.CreateGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	case "user_remove_from_group":
		k8sclient.DeleteGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	}
}
