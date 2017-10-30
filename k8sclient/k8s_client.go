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

package k8sclient

import (
	"fmt"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/rbac/v1beta1"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func CreateNamespace(name string) {
	// check if that namespace has already been created by either CreateProjectRoleBinding or CreateGroupRoleBinding
	// this has been implemented due to the asynchronous manner in which the webhook calls might be received
	// GetActualNameSpaceNameByGitlabName checks for the origin label field, so it only finds the namespace if it's
	// the correct one
	if actualNs := GetActualNameSpaceNameByGitlabName(name); actualNs != "" { return }


	nsName, err := GitlabNameToK8sNamespace(name)
	if check(err) {
		log.Fatal(err)
	}

	labelName, err := GitlabNameToK8sLabel(name)
	if check(err) {
		log.Fatal("Error while transforming gitlab name to k8s label: " + err.Error())
	}
	client := getK8sClient()
	_, err = client.Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": labelName}}})

	// if the already present namespace does not have "gitlab-ignored" label, we will update it with a  gitlab-origin label
	if k8serrors.IsAlreadyExists(err) {
		ns, errGetNs := getK8sClient().Namespaces().Get(nsName, metav1.GetOptions{})
		if check(errGetNs) {
			log.Fatal("Error while retrieving namespace. Error: " + errGetNs.Error())
		}
		if ns.Labels["gitlab-ignored"] == "" {
			// add label to already present namespace
			patchContent := fmt.Sprintf(`{"metadata":{"labels":{"gitlab-origin":"%s"}}}`, labelName)
			patchByteArray := []byte(patchContent)
			_, errPatch := client.Namespaces().Patch(nsName, types.MergePatchType, patchByteArray)
			if check(errPatch) {
				log.Fatal("Error while Updating namespace. Error: " + errPatch.Error())
			}
		}
	} else {
		// if error is due to namespace name collision, retry with suffixed number
		i := 0
		for k8serrors.IsAlreadyExists(err) {
			// it has gitlab-ignored label, so create new namespace with suffix counter
			i++
			nsName = nsName + "-" + strconv.Itoa(i)
			_, err = client.Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": labelName}}})
		}
	}
	log.Println(fmt.Sprintf("Succesfully created Namespace %s for Gitlab Ressource %s",nsName, name))
	// finally deploy CEPH Secret User if specified via ENV var
	DeployCEPHSecretUser(nsName)

	check(err)
}

func DeployCEPHSecretUser(namespace string) {
	if userKey := os.Getenv("CEPH_USER_KEY"); userKey != "" {
		client := getK8sClient()
		_, err := client.CoreV1().Secrets(namespace).Create(&v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ceph-secret-user",
				Namespace: namespace,
			},
			Data: map[string][]byte{"key": []byte(userKey)},
			Type: "kubernetes.io/rbd",
		})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Fatalln("Error creating CEPH Secret User. Error was: " + err.Error())
		}
	}
}

func DeleteNamespace(originalName string) {
	client := getK8sClient()
	correctNs := GetActualNameSpaceNameByGitlabName(originalName)
	if correctNs != "" {
		err := client.Namespaces().Delete(correctNs, &metav1.DeleteOptions{})
		if check(err) {
			log.Fatal("Deletion of Namespace failed with error: " + err.Error())
		}
	}
}

func CreateProjectRoleBinding(username, path, accessLevel string) {
	ns := GetActualNameSpaceNameByGitlabName(path)
	if ns == "" {
		CreateNamespace(path)
		ns = GetActualNameSpaceNameByGitlabName(path)
	}
	rolename := GetProjectRoleName(accessLevel)

	rB := v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ConstructRoleBindingName(username, rolename, ns), Namespace: ns},
		Subjects: []v1beta1.Subject{{Name: username, Kind: "User", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  v1beta1.RoleRef{Kind: "ClusterRole", Name: rolename, APIGroup: "rbac.authorization.k8s.io"}}

	_, err := getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	if k8serrors.IsNotFound(err){
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	}
	if check(err) {
		log.Fatal("Communication with K8s Server threw error, while creating RoleBinding. Err: " + err.Error())
	}
}

func DeleteProjectRoleBinding(username, path, accessLevel string) {
	ns, err := GitlabNameToK8sNamespace(path)
	if check(err) {
		log.Fatal(err)
	}

	rolename := GetProjectRoleName(accessLevel)

	if rolename != "" {
		roleBindingName := ConstructRoleBindingName(username, rolename, GetActualNameSpaceNameByGitlabName(path))
		DeleteProjectRoleBindingByName(roleBindingName, ns)
	}
}

func DeleteProjectRoleBindingByName(roleBindingName, actualNamespace string) {
	err := getK8sClient().RbacV1beta1().RoleBindings(actualNamespace).Delete(roleBindingName, &metav1.DeleteOptions{})
	if check(err) {
		log.Println("WARNING: Communication with K8s Server threw error, while deleting RoleBinding. Err: " + err.Error())
	}
}

func CreateGroupRoleBinding(username, path, accessLevel string) {
	ns := GetActualNameSpaceNameByGitlabName(path)
	if ns == "" {
		CreateNamespace(path)
		ns = GetActualNameSpaceNameByGitlabName(path)
	}
	rolename := GetGroupRoleName(accessLevel)

	rB := v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ConstructRoleBindingName(username, rolename, ns), Namespace: ns},
		Subjects: []v1beta1.Subject{{Name: username, Kind: "User", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  v1beta1.RoleRef{Kind: "ClusterRole", Name: GetGroupRoleName(accessLevel), APIGroup: "rbac.authorization.k8s.io"}}

	_, err := getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	if k8serrors.IsNotFound(err){
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	}
	if check(err) {
		log.Fatal("Communication with K8s Server threw error, while creating RoleBinding. Err: " + err.Error())
	}
	log.Println(fmt.Sprintf("INFO: Created GroupRoleBinding for user %s as %s in namespace %s", username, rolename, ns))
}

func DeleteGroupRoleBinding(username, path, accessLevel string) {

	ns, err := GitlabNameToK8sNamespace(path)
	if check(err) {
		log.Fatal(err)
	}

	rolename := GetGroupRoleName(accessLevel)

	if rolename != "" {
		roleBindingName := ConstructRoleBindingName(username, rolename, GetActualNameSpaceNameByGitlabName(path))
		DeleteGroupRoleBindingByName(roleBindingName, ns)
	}
}

func DeleteGroupRoleBindingByName(roleBindingName, actualNamespace string) {
	err := getK8sClient().RbacV1beta1().RoleBindings(actualNamespace).Delete(roleBindingName, &metav1.DeleteOptions{})
	if check(err) {
		log.Println("WARNING: Communication with K8s Server threw error, while deleting RoleBinding. Err: " + err.Error())
	}
}

// Utils

func GetAllGitlabOriginNamesFromNamespacesWithOriginLabel() []string {
	nsList, err := getK8sClient().CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: "gitlab-origin"})
	if check(err) {
		log.Fatal(err)
	}
	vsf := make([]string, 0)
	for _, v := range nsList.Items {
		if labelName := v.Labels["gitlab-origin"]; labelName != "" {
			gitlabName, err := K8sLabelToGitlabName(labelName)
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

func K8sLabelToGitlabName(givenName string) (string, error) {
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
