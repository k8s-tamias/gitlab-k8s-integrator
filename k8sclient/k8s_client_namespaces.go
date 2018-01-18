package k8sclient

import (
	"fmt"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"os"
	"strconv"
)

func DeleteNamespace(originalName string) string {

	client := getK8sClient()
	correctNs := GetActualNameSpaceNameByGitlabName(originalName)
	if correctNs == "kube-system" {
		return correctNs
	}
	if correctNs != "" {
		err := client.CoreV1().Namespaces().Delete(correctNs, &metav1.DeleteOptions{})
		if check(err) {
			log.Fatal("Deletion of Namespace failed with error: " + err.Error())
		}
	}
	return correctNs
}

func CreateNamespace(name string) string{
	if name == "kube-system" {
		return name
	}
	// check if that namespace has already been created by either CreateProjectRoleBinding or CreateGroupRoleBinding
	// this has been implemented due to the asynchronous manner in which the webhook calls might be received
	// GetActualNameSpaceNameByGitlabName checks for the origin label field, so it only finds the namespace if it's
	// the correct one
	if actualNs := GetActualNameSpaceNameByGitlabName(name); actualNs != "" {
		return actualNs
	}

	nsName, err := GitlabNameToK8sNamespace(name)
	if check(err) {
		log.Fatal(err)
	}

	labelName, err := GitlabNameToK8sLabel(name)
	if check(err) {
		log.Fatal("Error while transforming gitlab name to k8s label: " + err.Error())
	}
	client := getK8sClient()
	_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": labelName}}})

	// if the already present namespace does not have "gitlab-ignored" label, we will update it with a  gitlab-origin label
	if k8serrors.IsAlreadyExists(err) {
		ns, errGetNs := getK8sClient().CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
		if check(errGetNs) {
			log.Fatal("Error while retrieving namespace. Error: " + errGetNs.Error())
		}
		if ns.Labels["gitlab-ignored"] == "" {
			// add label to already present namespace
			patchContent := fmt.Sprintf(`{"metadata":{"labels":{"gitlab-origin":"%s"}}}`, labelName)
			patchByteArray := []byte(patchContent)
			_, errPatch := client.CoreV1().Namespaces().Patch(nsName, types.MergePatchType, patchByteArray)
			if check(errPatch) {
				log.Fatal("Error while Updating namespace. Error: " + errPatch.Error())
			}
		}
	} else {
		if err != nil {
			log.Println(fmt.Sprintf("Namespace creation caused an error, which was not IsAlreadyExists. Error was: %s", err))
		}
		// if error is due to namespace name collision, retry with suffixed number
		i := 0
		for k8serrors.IsAlreadyExists(err) {
			// it has gitlab-ignored label, so create new namespace with suffix counter
			i++
			nsName = nsName + "-" + strconv.Itoa(i)
			_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{"gitlab-origin": labelName}}})
		}
	}
	log.Println(fmt.Sprintf("Succesfully created Namespace %s for Gitlab Ressource %s", nsName, name))
	// deploy CEPH Secret User if specified via ENV var
	DeployCEPHSecretUser(nsName)

	// deploy GPU SA and RoleBinding
	DeployGPUServiceAccountAndRoleBinding(nsName)

	check(err)

	return nsName
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

func DeployGPUServiceAccountAndRoleBinding(namespace string){
	clusterRoleName := os.Getenv("GPU_PSP_CLUSTER_ROLE_NAME")
	if clusterRoleName == "" {
		return
	}

	client := getK8sClient()

	sa := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "gpu-serviceaccount", Namespace: namespace}}

	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil {
		log.Fatalln("Error creating GPU ServiceAccount. Error: " + err.Error())
	}

	rB := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "gpu-psp-binding", Namespace: namespace},
		Subjects: []rbacv1.Subject{{Name: serviceAccount.Name, Kind: "ServiceAccount", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: clusterRoleName, APIGroup: "rbac.authorization.k8s.io"}}

	_, err = client.RbacV1().RoleBindings(namespace).Create(&rB)
	if err != nil {
		log.Fatalln("Error creating GPU ServiceAccount. Error: " + err.Error())
	}
}