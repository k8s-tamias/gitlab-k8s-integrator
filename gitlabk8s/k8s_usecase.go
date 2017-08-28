package gitlabk8s

import (
	"encoding/json"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/rbac/v1beta1"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
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
		createNamespace(event.PathWithNameSpace)

	case "project_destroy":
		deleteNamespace(event.PathWithNameSpace)

	case "project_rename":
		deleteNamespace(event.OldPathWithNamespace)
		createNamespace(event.PathWithNameSpace)

	case "project_transferred":
		deleteNamespace(event.OldPathWithNamespace)
		createNamespace(event.PathWithNameSpace)

	// project member operations

	case "user_add_to_team":
		createProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	case "user_remove_from_team":
		deleteProjectRoleBinding(event.UserUsername, event.PathWithNameSpace, event.ProjectAccess)

	// group operations

	case "group_create":
		createNamespace(event.Path)

	case "group_destroy":
		deleteNamespace(event.PathWithNameSpace)

	// group member operations

	case "user_add_to_group":
		createGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	case "user_remove_from_group":
		deleteGroupRoleBinding(event.UserUsername, event.PathWithNameSpace, event.GroupAccess)

	}
}

func createNamespace(name string) {
	nsName, err := getK8sCompatibleNamespaceName(name)
	if check(err) {
		log.Fatal(err)
	}

	_, err = getK8sClient().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": name}}})

	// if error is due to namespace name collision, retry with suffixed number
	i := 0
	for k8serrors.IsAlreadyExists(err) {
		i++
		nsName = nsName + "-" + strconv.Itoa(i)
		_, err = getK8sClient().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": name}}})
	}
	check(err)
}

func deleteNamespace(path string) {
	k8sclient := getK8sClient()
	correctNs := getActualNameSpaceName(path)
	if correctNs != "" {
		err := k8sclient.Namespaces().Delete(correctNs, &metav1.DeleteOptions{})
		check(err)
	}
}

// getActualNameSpaceName looks for the original name from gitlab in the gitlab-origin labels of namespaces
// and returns the given namespace in the K8s cluster
func getActualNameSpaceName(gitlabOriginName string) string {
	correctName := ""

	k8sclient := getK8sClient()
	namespaces, err := k8sclient.CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: "gitlab-origin=" + gitlabOriginName})
	if check(err) {
		log.Fatal("Error while retrieving namespaces: " + err.Error())
	}
	if len(namespaces.Items) > 1 {
		log.Println("WARNING: Found mutliple namespaces with gitlab-origin= " + gitlabOriginName + ". This is potentially very bad, consult a cloud admin!")
	} else if len(namespaces.Items) < 1 {
		log.Println("INFO: No namespace has been found with gitlab-origin= " + gitlabOriginName + ". Check if namespace still exsists in K8s Cluster!")
	} else {
		correctName = namespaces.Items[0].Name
	}
	return correctName
}

func createProjectRoleBinding(username, path, accessLevel string) {
	ns, err := getK8sCompatibleNamespaceName(path)

	if check(err) {
		log.Fatal(err)
	}

	rolename := getProjectRoleName(accessLevel)

	rB := v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: getRoleBindingName(username, rolename, getActualNameSpaceName(path)), Namespace: ns},
		Subjects: []v1beta1.Subject{{Name: username, Kind: "User", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  v1beta1.RoleRef{Kind: "ClusterRole", Name: getProjectRoleName(accessLevel)}}

	getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
}

func deleteProjectRoleBinding(username, path, accessLevel string) {
	ns, err := getK8sCompatibleNamespaceName(path)
	if check(err) {
		log.Fatal(err)
	}

	rolename := getProjectRoleName(accessLevel)

	if rolename != "" {
		getK8sClient().RbacV1beta1().RoleBindings(ns).Delete(getRoleBindingName(username, rolename, getActualNameSpaceName(path)), &metav1.DeleteOptions{})
	}
}

func createGroupRoleBinding(username, path, accessLevel string) {
	ns, err := getK8sCompatibleNamespaceName(path)

	if check(err) {
		log.Fatal(err)
	}

	rolename := getGroupRoleName(accessLevel)

	rB := v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: getRoleBindingName(username, rolename, getActualNameSpaceName(path)), Namespace: ns},
		Subjects: []v1beta1.Subject{{Name: username, Kind: "User", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  v1beta1.RoleRef{Kind: "ClusterRole", Name: getGroupRoleName(accessLevel)}}

	getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
}

func deleteGroupRoleBinding(username, path, accessLevel string) {
	ns, err := getK8sCompatibleNamespaceName(path)
	if check(err) {
		log.Fatal(err)
	}

	rolename := getGroupRoleName(accessLevel)

	if rolename != "" {
		getK8sClient().RbacV1beta1().RoleBindings(ns).Delete(getRoleBindingName(username, rolename, getActualNameSpaceName(path)), &metav1.DeleteOptions{})
	}
}

func getRoleBindingName(username, rolename, ns string) string {
	return username + "-" + rolename + "-" + ns
}

func getK8sCompatibleNamespaceName(givenName string) (string, error) {
	nsName := strings.ToLower(givenName)

	replacer := strings.NewReplacer(" ", "",
		"ü", "ue",
		"ö", "oe",
		"ä", "ae",
		"ß", "ss",
		"_", "-",
		".", "-",
		"/", "-")

	nsName = replacer.Replace(nsName)
	// regex for checking k8s namespace name
	regex, err := regexp.Compile("[a-z0-9]([-a-z0-9]*[a-z0-9])?")

	if check(err) {
		return "", err
	}

	if !regex.MatchString(nsName) {
		return "", errors.New("Created Namespace name did not adhere to rules")
	}

	return nsName, nil
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

func getProjectRoleName(accessLevel string) string {
	rname := ""
	switch accessLevel {
	case "Master":
		rname := os.Getenv("PROJECT_MASTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-master"
		}
	case "Reporter":
		rname := os.Getenv("PROJECT_REPORTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-reporter"
		}
	case "Developer":
		rname := os.Getenv("PROJECT_DEVELOPER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-developer"
		}

	default:
		rname := os.Getenv("PROJECT_DEFAULT_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-guest"
		}
	}
	return rname
}

func getGroupRoleName(accessLevel string) string {
	rname := ""
	switch accessLevel {
	case "Master":
		rname := os.Getenv("GROUP_MASTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-master"
		}
	case "Reporter":
		rname := os.Getenv("GROUP_REPORTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-reporter"
		}
	case "Developer":
		rname := os.Getenv("GROUP_DEVELOPER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-developer"
		}

	default:
		rname := os.Getenv("GROUP_DEFAULT_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-guest"
		}
	}
	return rname
}
