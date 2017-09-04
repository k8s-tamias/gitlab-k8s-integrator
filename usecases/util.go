package usecases

import "log"

func check(err error) bool {
	if err != nil {
		log.Println("Error : ", err.Error())
		return true
	}
	return false
}
