package gitlabclient

import "log"

type GitlabGroup struct {
}

type GitlabSubGroup struct {
}

type GitlabProject struct {
}

type GitlabPersonalRepo struct {
}

type GitlabContent struct {
	Groups []GitlabGroup
}

func GetFullGitlabContent() (GitlabContent, error) {
	groups, err := getAllGroups()
	if check(err) {
		log.Println("Error while retrieving Gitlab Contents! Err:" + err.Error())
		return GitlabContent{}, err
	}

	return GitlabContent{Groups: groups}, nil
}

func getAllGroups() ([]GitlabGroup, error) {
	return []GitlabGroup{}, nil
}

func check(err error) bool {
	if err != nil {
		log.Println("Error : ", err.Error())
		return true
	}
	return false
}
