package k8sclient

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"github.com/pkg/errors"
	"log"
	rbacv1 "k8s.io/api/rbac/v1"
	"time"
	"os"
)

type ServiceAccountInfo struct {
	Name 		string
	Namespace	string
	Token		string
}

// CreateServiceAccountAndRoleBinding creates a ServiceAccount and a RoleBinding for it to use it.
// If either of the two already exists, it will instead return their information to the caller
// returns (InfoAboutServiceAccount, RoleBindingName, error)
func CreateServiceAccountAndRoleBinding(fullProjectPath string) (ServiceAccountInfo, string, error) {
	name := getServiceAccountName()
	namespace := GetActualNameSpaceNameByGitlabName(fullProjectPath)

	client := getK8sClient()

	sa := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}

	serviceAccount, err := client.ServiceAccounts(namespace).Create(sa)

	if k8serrors.IsAlreadyExists(err) {
		// ServiceAccount already exists, so retrieve and use it
		serviceAccount, err = client.ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return ServiceAccountInfo{},"", err
		}
	} else if err != nil {
		return ServiceAccountInfo{}, "", err
	}

	// try to retrieve ServiceAccount once as the newly created one won't have the secret set
	serviceAccount, err = client.ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	// The secret in the ServiceAccount is not created and linked immediately, so we have to wait for it
	// to not wait indefinitely we use a timeout
	timeout := time.After(30 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	// Keep trying until we're timed out or got a result or got an error
	for k8serrors.IsNotFound(err) || len(serviceAccount.Secrets) < 1 {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return ServiceAccountInfo{}, "", errors.New("ServiceAccount was created, but Secrets were empty!")

		case <-tick:
			serviceAccount, err = client.ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
				return ServiceAccountInfo{},"", err
			}
		}
	}

	secretName := serviceAccount.Secrets[0].Name
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

	rB := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns},
		Subjects: []rbacv1.Subject{{Name: saName, Kind: "ServiceAccount", Namespace: ns}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: rolename, APIGroup: "rbac.authorization.k8s.io"}}

	_, err := getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	if err != nil && k8serrors.IsNotFound(err) {
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1beta1().RoleBindings(ns).Create(&rB)
	}
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		log.Fatal("Communication with K8s Server threw error, while creating ServiceAccount RoleBinding. Err: " + err.Error())
	}
	return rB.Name
}

func getServiceAccountName() string {
	name := os.Getenv("GITLAB_SERVICEACCOUNT_NAME")
	if name == "" {
		name = "gitlab-serviceaccount"
	} else if errs := validation.IsDNS1123Label(name) ; len(errs) != 0 {
		log.Fatalf("The provided value for GITLAB_SERVICEACCOUNT_NAME is not a DNS-1123 compliant name!")
	}
	return name
}