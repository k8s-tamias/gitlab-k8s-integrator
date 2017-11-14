package k8sclient

import (
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/pkg/errors"
	"log"
	"k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

type ServiceAccountInfo struct {
	Name 		string
	Namespace	string
	Token		string
}

// CreateServiceAccountAndRoleBinding creates a ServiceAccount and a matching secret to use it.
func CreateServiceAccountAndRoleBinding(name, fullProjectPath string) (ServiceAccountInfo, string, error) {
	namespace := GetActualNameSpaceNameByGitlabName(fullProjectPath)

	client := getK8sClient()

	sa := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}

	sa, err := client.ServiceAccounts(namespace).Create(sa)

	if k8serrors.IsAlreadyExists(err) {
		// ServiceAccount already exists, so retrieve and use it
		sa, err = client.ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return ServiceAccountInfo{},"", err
		}
	} else if err != nil {
		return ServiceAccountInfo{}, "", err
	}

	if len(sa.Secrets) < 1 {
		return ServiceAccountInfo{}, "", errors.New("ServiceAccount was created, but Secrets were empty!")
	}
	secretName := sa.Secrets[0].Name
	saSecret, err := client.Secrets(namespace).Get(secretName, metav1.GetOptions{})
	token := saSecret.Data["token"]
	if len(token) <= 0 {
		return ServiceAccountInfo{}, "", errors.New("The token field in the Secret's data was empty!")
	}

	tokenAsString := string(token[:])

	sAI := ServiceAccountInfo{Namespace: namespace, Name: name, Token: tokenAsString}
	rbName := createServiceAccountRoleBinding(name, fullProjectPath)
	return sAI, rbName, nil
}

func createServiceAccountRoleBinding(saName, path string) string {
	ns := GetActualNameSpaceNameByGitlabName(path)
	if ns == "" {
		CreateNamespace(path)
		ns = GetActualNameSpaceNameByGitlabName(path)
	}
	// ServiceAccounts are always bound to Master roles
	rolename := GetProjectRoleName("Master")

	rB := v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns},
		Subjects: []v1beta1.Subject{{Name: saName, Kind: "ServiceAccount", Namespace: ns}},
		RoleRef:  v1beta1.RoleRef{Kind: "ClusterRole", Name: rolename, APIGroup: "rbac.authorization.k8s.io"}}

	rb, err := getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	if k8serrors.IsNotFound(err) {
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	}
	if check(err) {
		log.Fatal("Communication with K8s Server threw error, while creating ServiceAccount RoleBinding. Err: " + err.Error())
	}
	return rb.Name
}