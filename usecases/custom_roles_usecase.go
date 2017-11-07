package usecases

import (
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"log"
	"io/ioutil"
	"regexp"
	"bytes"
)

func ReadCustomRolesAndBindings() {
	customDir := getCustomRoleDir()
	customRolesPresent, err := fileExists(customDir)
	if err != nil {
		log.Printf("An error occurred while trying to read custom roles from directory %s. Err: %s", customDir, err)
		return
	}
	if !customRolesPresent {
		log.Println("No custom-roles directory present, skipping step...")
		return
	}

	files, err := ioutil.ReadDir(customDir)
	if err != nil {
		log.Printf("An error occurred while trying to read custom roles from directory %s. Err: %s", customDir, err)
		return
	}
	for _, f := range files {
		isYaml, err := regexp.MatchString(".yaml", f.Name()) // TODO also allow yml
		if err != nil {
			log.Printf("An error occurred while checking suffix of file %s from directory %s. Err: %s", f.Name(), customDir, err)
			return
		}

		if !f.IsDir() && isYaml {
			fileR, err := ioutil.ReadFile(f.Name())
			if err != nil {
				log.Printf("An error occurred while reading file %s from directory %s. Err: %s", f.Name(), customDir, err)
				return
			}
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(fileR), 100)

		}
	}
}

func fileExists(filename string) (bool, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func getCustomRoleDir() string {
	dir := "/etc/custom-roles"
	envDir := os.Getenv("CUSTOM_ROLE_DIR")
	if  envDir != "" {
		dir = envDir
	}
	return dir
}