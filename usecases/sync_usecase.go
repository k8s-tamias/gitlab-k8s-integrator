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

package usecases

import (
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/gitlabclient"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/k8sclient"
	"log"
	"os"
	"time"
	"sync"
	"fmt"
)

/*
What to fetch from k8s api
- Get all namespaces with gitlab-origin field (ns without that field won't be gitlab created)
- Get all rolebindings of these namespaces

What to get from gitlab
- get all groups
- get all projects
- get all users (private namespace)

Algo:
1. Delete all namespaces which are not in the gitlab Set
2. Iterate all gitlab namespaces
   if namespace is present in k8s set:
	2.1 Iterate all rolebindings
	2.2 Compare to rolebindings from k8s set by using the gitlab-origin field as key and
		2.2.1 Delete every rolebinding not present in the gitlab set
		2.2.1 Create every rolebinding not present in the k8s set
   else:
	2.1 Create namespace
		2.1.1 If namespace is present by name, but does not have a gitlab-origin label attached
		AND is not(!) labeled with 'gitlab-ignored' it get's labeled with its origin name.
		Otherwise the naming collision is solved by suffixing the name with a counter
	2.2 Create all rolebindings

 done

*/

// TODO : Cache Webhooks while Sync is running and execute them later!

func PerformGlK8sSync() {
	log.Println("Starting new Synchronization run!")
	log.Println("Getting Gitlab Contents...")
	gitlabContent, err := gitlabclient.GetFullGitlabContent()
	if check(err) {
		return
	}

	// 1. delete all Namespaces which are not in the gitlab set
	log.Println("Getting K8s Contents...")
	gitlabNamespacesInK8s := k8sclient.GetAllGitlabOriginNamesFromNamespacesWithOriginLabel()

	log.Println("Deleting all namespaces which are no longer in the gitlab namespace...")
	for _, originalName := range gitlabNamespacesInK8s {
		delete := true

		for _, user := range gitlabContent.Users {
			if originalName == user.Username {
				delete = false
				break
			}
		}

		if delete {
			for _, project := range gitlabContent.Projects {
				if originalName == project.PathWithNameSpace {
					delete = false
					break
				}
			}
		}

		if delete {
			for _, group := range gitlabContent.Groups {
				if originalName == group.FullPath {
					delete = false
					break
				}
			}
		}

		if delete {
			k8sclient.DeleteNamespace(originalName)
		}
	}

	log.Println("Reading custom-rolebindings if any...")

	cRaB := ReadAndApplyCustomRolesAndBindings()

	var syncDoneWg sync.WaitGroup

	log.Println("Syncing Gitlab Users...")
	go syncUsers(gitlabContent, cRaB, syncDoneWg)

	log.Println("Syncing Gitlab Groups...")
	go syncGroups(gitlabContent, cRaB, syncDoneWg)

	log.Println("Syncing Gitlab Projects...")
	go syncProjects(gitlabContent, cRaB, syncDoneWg)

	syncDoneWg.Wait()
	log.Println("Finished Synchronization run.")
}

// TODO also delete ServiceAccounts and associated Rolebindings in Namespaces when users get deleted from namespaces

func syncUsers(gitlabContent *gitlabclient.GitlabContent, cRaB CustomRolesAndBindings, syncDoneWg sync.WaitGroup){
	defer syncDoneWg.Done()
	for _, user := range gitlabContent.Users {
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(user.Username)
		if actualNamespace != "" {
			// create or get ServiceAccount
			sAI, err := k8sclient.CreateServiceAccountAndSecret(user.Username, actualNamespace)
			if err != nil {
				log.Fatalln(fmt.Sprintf("A fatal error occurred while creating a ServiceAccount. Err was: %s"), err)
			}

			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)
			roleName := k8sclient.GetGroupRoleName("Master")
			expectedGitlabRolebindingName := k8sclient.ConstructRoleBindingName(user.Username, roleName, actualNamespace)

			// 2.1 Iterate all roleBindings
			for rb := range k8sRoleBindings {
				if rb != expectedGitlabRolebindingName && !cRaB.RoleBindings[rb] {
					k8sclient.DeleteGroupRoleBindingByName(rb, actualNamespace)
				}
			}
			// make sure the project's role binding is present
			if !k8sRoleBindings[expectedGitlabRolebindingName] {
				k8sclient.CreateGroupRoleBinding(user.Username, user.Username, "Master")
			}

			// finally check if namespace has CEPHSecretUser
			k8sclient.DeployCEPHSecretUser(actualNamespace)

		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(user.Username)
			k8sclient.CreateGroupRoleBinding(user.Username, user.Username, "Master")
		}
	}
}

func syncGroups(gitlabContent *gitlabclient.GitlabContent, cRaB CustomRolesAndBindings, syncDoneWg sync.WaitGroup){
	defer syncDoneWg.Done()
	// same same for Groups
	for _, group := range gitlabContent.Groups {
		if debugSync() {
			log.Println("Syncing: " + group.FullPath)
		}
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(group.FullPath)
		if debugSync() {
			log.Println("ActualNamespace: " + actualNamespace)
		}
		if actualNamespace != "" {
			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)
			if debugSync() {
				log.Printf("Found %d rolebindings \n", len(k8sRoleBindings))
			}

			// get expectedRoleBindings by retrieved Members
			expectedRoleBindings := map[string]bool{}
			for _, member := range group.Members {
				if debugSync() {
					log.Println("Processing member " + member.Name)
				}
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				roleName := k8sclient.GetGroupRoleName(accessLevel)
				rbName := k8sclient.ConstructRoleBindingName(member.Username, roleName, actualNamespace)
				expectedRoleBindings[rbName] = true

				if debugSync() {
					log.Printf("AccessLevel: %s, roleName: %s, rbName: %s", accessLevel, roleName, rbName)
				}

				// make sure the groups's expected rolebindings are present
				if !k8sRoleBindings[rbName] {
					if debugSync() {
						log.Println("Creating RoleBinding " + rbName)
					}
					k8sclient.CreateGroupRoleBinding(member.Username, group.FullPath, accessLevel)
				}
			}

			// 2.1 Iterate all roleBindings and delete those which are not anymore present in gitlab or in custom roles
			for rb := range k8sRoleBindings {
				if !expectedRoleBindings[rb] && !cRaB.RoleBindings[rb]{
					if debugSync() {
						log.Println("Deleting RoleBinding " + rb)
					}
					k8sclient.DeleteGroupRoleBindingByName(rb, actualNamespace)
				}
			}
			// finally check if namespace has CEPHSecretUser
			k8sclient.DeployCEPHSecretUser(actualNamespace)

		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(group.FullPath)
			if debugSync() {
				log.Println("Creating Namespace for " + group.FullPath)
			}
			for _, member := range group.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				k8sclient.CreateGroupRoleBinding(member.Username, group.FullPath, accessLevel)
			}
		}
	}
}

func syncProjects(gitlabContent *gitlabclient.GitlabContent, cRaB CustomRolesAndBindings, syncDoneWg sync.WaitGroup) {
	defer syncDoneWg.Done()
	for _, project := range gitlabContent.Projects {
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(project.PathWithNameSpace)
		if actualNamespace != "" {
			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)

			// get expectedRoleBindings by retrieved Members
			expectedRoleBindings := map[string]bool{}
			for _, member := range project.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				roleName := k8sclient.GetProjectRoleName(accessLevel)
				rbName := k8sclient.ConstructRoleBindingName(member.Username, roleName, actualNamespace)
				expectedRoleBindings[rbName] = true

				// make sure the project's expected rolebindings are present
				if !k8sRoleBindings[rbName] {
					k8sclient.CreateProjectRoleBinding(member.Username, project.PathWithNameSpace, accessLevel)
				}
			}

			// 2.1 Iterate all roleBindings and delete those which are not anymore present in gitlab
			for rb := range k8sRoleBindings {
				if !expectedRoleBindings[rb] && !cRaB.RoleBindings[rb] {
					k8sclient.DeleteProjectRoleBindingByName(rb, actualNamespace)
				}
			}

			// finally check if namespace has CEPHSecretUser
			k8sclient.DeployCEPHSecretUser(actualNamespace)
		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(project.PathWithNameSpace)
			for _, member := range project.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				k8sclient.CreateProjectRoleBinding(member.Username, project.PathWithNameSpace, accessLevel)
			}
		}
	}
}

func StartRecurringSyncTimer() {
	log.Println("Starting Sync Timer...")
	ticker := time.NewTicker(time.Hour * 3)
	go func() {
		for range ticker.C {
			go PerformGlK8sSync()
		}
	}()
}

func debugSync() bool {
	return os.Getenv("ENABLE_GITLAB_SYNC_DEBUG") == "true"
}