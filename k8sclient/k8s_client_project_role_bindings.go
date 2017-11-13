package k8sclient

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"k8s.io/client-go/pkg/apis/rbac/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

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
	if k8serrors.IsNotFound(err) {
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