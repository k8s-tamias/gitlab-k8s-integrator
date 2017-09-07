package usecases

import (
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/gitlabclient"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/k8sclient"
	"log"
	"time"
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
	gitlabContent, err := gitlabclient.GetFullGitlabContent()
	if check(err) {
		return
	}

	// 1. delete all Namespaces which are not in the gitlab set
	log.Println("Getting Gitlab Contents...")
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

		if delete == true {
			for _, project := range gitlabContent.Projects {
				if originalName == project.PathWithNameSpace {
					delete = false
					break
				}
			}
		}

		if delete == true {
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

	log.Println("Syncing Gitlab Users...")
	// 2. iterate all gitlab "namespaces"
	for _, user := range gitlabContent.Users {
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(user.Username)
		if actualNamespace != "" {
			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)

			expectedGitlabRolebindingName := k8sclient.ConstructRoleBindingName(user.Username, k8sclient.GetGroupRoleName("Master"), actualNamespace)
			// 2.1 Iterate all roleBindings
			for rb := range k8sRoleBindings {
				if rb != expectedGitlabRolebindingName {
					k8sclient.DeleteGroupRoleBindingByName(rb, actualNamespace)
				}
			}
			// make sure the project's role binding is present
			if !k8sRoleBindings[expectedGitlabRolebindingName] {
				k8sclient.CreateGroupRoleBinding(user.Username, user.Username, "Master")
			}

		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(user.Username)
			k8sclient.CreateGroupRoleBinding(user.Username, user.Username, "Master")
		}
	}

	log.Println("Syncing Gitlab Groups...")
	// same same for Groups
	for _, group := range gitlabContent.Groups {
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(group.FullPath)
		if actualNamespace != "" {
			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)

			// get expectedRoleBindings by retrieved Members
			expectedRoleBindings := map[string]bool{}
			for _, member := range group.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				roleName := k8sclient.GetGroupRoleName(accessLevel)
				rbName := k8sclient.ConstructRoleBindingName(member.Username, roleName, actualNamespace)
				expectedRoleBindings[rbName] = true

				// make sure the project's expected rolebindings are present
				if !k8sRoleBindings[rbName] {
					k8sclient.CreateGroupRoleBinding(member.Username, group.FullPath, accessLevel)
				}
			}

			// 2.1 Iterate all roleBindings and delete those which are not anymore present in gitlab
			for rb := range k8sRoleBindings {
				if !expectedRoleBindings[rb] {
					k8sclient.DeleteGroupRoleBindingByName(rb, actualNamespace)
				}
			}

		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(group.FullPath)
			for _, member := range group.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				k8sclient.CreateGroupRoleBinding(member.Username, group.FullPath, accessLevel)
			}
		}
	}

	log.Println("Syncing Gitlab Projects...")
	// same same for Projects
	for _, project := range gitlabContent.Projects {
		actualNamespace := k8sclient.GetActualNameSpaceNameByGitlabName(project.PathWithNameSpace)
		if actualNamespace != "" {
			// namespace is present, check rolebindings
			k8sRoleBindings := k8sclient.GetRoleBindingsByNamespace(actualNamespace)

			// get expectedRoleBindings by retrieved Members
			expectedRoleBindings := map[string]bool{}
			for _, member := range project.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				roleName := k8sclient.GetGroupRoleName(accessLevel)
				rbName := k8sclient.ConstructRoleBindingName(member.Username, roleName, actualNamespace)
				expectedRoleBindings[rbName] = true

				// make sure the project's expected rolebindings are present
				if !k8sRoleBindings[rbName] {
					k8sclient.CreateProjectRoleBinding(member.Username, project.PathWithNameSpace, accessLevel)
				}
			}

			// 2.1 Iterate all roleBindings and delete those which are not anymore present in gitlab
			for rb := range k8sRoleBindings {
				if !expectedRoleBindings[rb] {
					k8sclient.DeleteProjectRoleBindingByName(rb, actualNamespace)
				}
			}

		} else {
			// create Namespace & RoleBinding
			k8sclient.CreateNamespace(project.PathWithNameSpace)
			for _, member := range project.Members {
				accessLevel := gitlabclient.TranslateIntAccessLevels(member.AccessLevel)
				k8sclient.CreateProjectRoleBinding(member.Username, project.PathWithNameSpace, accessLevel)
			}
		}
	}
	log.Println("Finished Synchronization run.")

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
