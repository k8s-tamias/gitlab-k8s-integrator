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
- get all subgroups
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

func PerformGlK8sSync() {
	gitlabContent, err := gitlabclient.GetFullGitlabContent()
	if check(err) {
		return
	}

	// 1. delete all Namespaces which are not in the gitlab set
	namespaces := k8sclient.GetAllNamespacesByOriginLabel()

	for _, namespace := range namespaces {
		delete := true

		for _, user := range gitlabContent.Users {
			if namespace == user.Username {
				delete = false
				break
			}
		}

		if delete == true {
			for _, project := range gitlabContent.Projects {
				if namespace == project.PathWithNameSpace {
					delete = false
					break
				}
			}
		}

		if delete == true {
			for _, group := range gitlabContent.Groups {
				if namespace == group.FullPath {
					delete = false
					break
				}
			}
		}

		if delete {
			k8sclient.DeleteNamespace(namespace)
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
