package k8sclient

import (
	"log"

	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateProjectRoleBinding(username, path, accessLevel string) {
	ns := GetActualNameSpaceNameByGitlabName(path)
	if ns == "" {
		CreateNamespace(path)
		ns = GetActualNameSpaceNameByGitlabName(path)
	}
	rolename := GetProjectRoleName(accessLevel)

	rB := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ConstructRoleBindingName(username, rolename, ns), Namespace: ns},
		Subjects: []rbacv1.Subject{{Name: username, Kind: "User"}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: rolename, APIGroup: "rbac.authorization.k8s.io"}}

	_, err := getK8sClient().RbacV1().RoleBindings(ns).Create(&rB)
	if k8serrors.IsNotFound(err) {
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1().RoleBindings(ns).Create(&rB)
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
	ns, errGetNs := getK8sClient().CoreV1().Namespaces().Get(actualNamespace, metav1.GetOptions{})
	if check(errGetNs) {
		log.Fatal("Error while retrieving namespace. Error: " + errGetNs.Error())
	}
	if ns.Labels["gitlab-ignored"] == "" {
		err := getK8sClient().RbacV1().RoleBindings(actualNamespace).Delete(roleBindingName, &metav1.DeleteOptions{})
		if check(err) {
			log.Println("WARNING: Communication with K8s Server threw error, while deleting RoleBinding. Err: " + err.Error())
		}
	}
}
