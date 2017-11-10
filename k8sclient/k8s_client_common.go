package k8sclient

import (
	"fmt"
	"log"
	"os"
	"strings"
	"regexp"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"github.com/pkg/errors"
)

// Utils

func GetAllGitlabOriginNamesFromNamespacesWithOriginLabel() []string {
	nsList, err := getK8sClient().CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: "gitlab-origin"})
	if check(err) {
		log.Fatal(err)
	}
	vsf := make([]string, 0)
	for _, v := range nsList.Items {
		if labelName := v.Labels["gitlab-origin"]; labelName != "" {
			gitlabName, err := k8sLabelToGitlabName(labelName)
			if check(err) {
				log.Fatal("Error while transforming labelName back to Gitlab Name. Err: " + err.Error())
			}
			vsf = append(vsf, gitlabName)
		}
	}
	return vsf
}

// GetActualNameSpaceNameByGitlabName looks for the original name from gitlab in the gitlab-origin labels of namespaces
// and returns the given namespace name in the K8s cluster or an empty string if namespace has not been found
func GetActualNameSpaceNameByGitlabName(gitlabOriginName string) string {
	correctName := ""

	client := getK8sClient()

	k8sName, err := GitlabNameToK8sLabel(gitlabOriginName)
	if check(err) {
		log.Fatal("Error while transforming gitlab name to k8s label: " + err.Error())
	}

	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: "gitlab-origin=" + k8sName})
	if check(err) {
		log.Fatal("Error while retrieving namespaces: " + err.Error())
	}
	if len(namespaces.Items) > 1 {
		log.Println("WARNING: Found mutliple namespaces with gitlab-origin= " + k8sName + ". This is potentially very bad, consult a cloud admin!")
	} else if len(namespaces.Items) < 1 {
		log.Println("INFO: No namespace has been found with gitlab-origin=" + k8sName + ".")
	} else {
		correctName = namespaces.Items[0].Name
	}
	return correctName
}

/// GetRoleBindingsByNamespace retrieves the rolebindings present in K8s for the provided namespace
/// the namespace parameter is assumed to be the real namespace name in k8s!
func GetRoleBindingsByNamespace(namespace string) map[string]bool {
	rbs, err := getK8sClient().RbacV1beta1().RoleBindings(namespace).List(metav1.ListOptions{})
	if check(err) {
		log.Fatal(fmt.Sprintf("Error while retrieving rolebindings for namespace %s. Error: %s", namespace, err))
	}
	res := map[string]bool{}

	for _, rb := range rbs.Items {
		res[rb.Name] = true
	}

	return res
}

func ConstructRoleBindingName(username, rolename, ns string) string {
	return username + "-" + rolename + "-" + ns
}

func GetProjectRoleName(accessLevel string) string {
	var rname string
	switch accessLevel {
	case "Master":
		rname = os.Getenv("PROJECT_MASTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-master"
		}
	case "Reporter":
		rname = os.Getenv("PROJECT_REPORTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-reporter"
		}
	case "Developer":
		rname = os.Getenv("PROJECT_DEVELOPER_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-developer"
		}

	default:
		rname = os.Getenv("PROJECT_DEFAULT_ROLENAME")
		if rname == "" {
			rname = "gitlab-project-guest"
		}
	}
	return rname
}

func GetGroupRoleName(accessLevel string) string {
	var rname string
	switch accessLevel {
	case "Master":
		rname = os.Getenv("GROUP_MASTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-master"
		}
	case "Reporter":
		rname = os.Getenv("GROUP_REPORTER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-reporter"
		}
	case "Developer":
		rname = os.Getenv("GROUP_DEVELOPER_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-developer"
		}

	default:
		rname = os.Getenv("GROUP_DEFAULT_ROLENAME")
		if rname == "" {
			rname = "gitlab-group-guest"
		}
	}
	return rname
}

// Internal Functions

func GitlabNameToK8sNamespace(givenName string) (string, error) {
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

func GitlabNameToK8sLabel(givenName string) (string, error) {
	/*
		Rules:
		1) “.” -> “.”
		2) “-” -> “-”
		3) “_” -> “__”
		4) “/” -> “_”
	*/
	replacer := strings.NewReplacer("_", "__",
		"/", "_")

	labelName := replacer.Replace(givenName)
	// regex for checking k8s namespace name
	regex, err := regexp.Compile("(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?")

	if check(err) {
		return "", err
	}

	if !regex.MatchString(labelName) {
		return "", errors.New("Created Namespace name did not adhere to rules")
	}

	return labelName, nil
}

func k8sLabelToGitlabName(givenName string) (string, error) {
	/*
		Rules:
		1) “.” <- “.”
		2) “-” <- “-”
		3) “_” <- “__”
		4) “/” <- “_”
	*/
	// Path can contain only letters, digits, '_', '-' and '.'. Cannot start with '-' or end in '.', '.git' or '.atom'.
	replacer := strings.NewReplacer("__", "_",
		"_", "/")

	labelName := replacer.Replace(givenName)
	// regex for checking gitlab namespace name
	regex, err := regexp.Compile("(?:[a-zA-Z0-9_.][a-zA-Z0-9_.]*[a-zA-Z0-9_-]|[a-zA-Z0-9_])")

	if check(err) {
		return "", err
	}

	if !regex.MatchString(labelName) {
		return "", errors.New("Created Gitlab Label name did not adhere to rules")
	}

	return labelName, nil
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
		return true
	}
	return false
}
