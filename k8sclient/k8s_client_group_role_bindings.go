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
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func CreateGroupRoleBinding(username, path, accessLevel string) {
	ns := GetActualNameSpaceNameByGitlabName(path)
	if ns == "" {
		CreateNamespace(path)
		ns = GetActualNameSpaceNameByGitlabName(path)
	}
	rolename := GetGroupRoleName(accessLevel)

	rB := rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ConstructRoleBindingName(username, rolename, ns), Namespace: ns},
		Subjects: []rbacv1.Subject{{Name: username, Kind: "User", APIGroup: "rbac.authorization.k8s.io"}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: GetGroupRoleName(accessLevel), APIGroup: "rbac.authorization.k8s.io"}}

	_, err := getK8sClient().RbacV1().RoleBindings(ns).Create(&rB)
	if k8serrors.IsNotFound(err) {
		CreateNamespace(path)
		_, err = getK8sClient().RbacV1().RoleBindings(ns).Create(&rB)
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
