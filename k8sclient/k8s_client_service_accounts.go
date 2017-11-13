package k8sclient

import (
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/pkg/errors"
)

type ServiceAccountInfo struct {
	Name 		string
	Namespace	string
	Token		string
}

func CreateServiceAccountAndSecret(username, namespace string) (ServiceAccountInfo, error) {
	client := getK8sClient()

	sa := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name:username, Namespace: namespace}}

	sa, err := client.ServiceAccounts(namespace).Create(sa)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return ServiceAccountInfo{}, err
	}
	if len(sa.Secrets) < 1 {
		return ServiceAccountInfo{}, errors.New("ServiceAccount was created, but Secrets were empty!")
	}
	secretName := sa.Secrets[0].Name
	saSecret, err := client.Secrets(namespace).Get(secretName, metav1.GetOptions{})
	token := saSecret.Data["token"]
	tokenAsString := string(token[:])

	sAI := ServiceAccountInfo{Namespace: namespace, Name: username, Token: tokenAsString}
	return sAI, nil
}
