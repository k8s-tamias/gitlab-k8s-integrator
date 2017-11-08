package usecases

import (
	"io/ioutil"
	"k8s.io/client-go/pkg/api"
	_ "k8s.io/client-go/pkg/api/install"
	_ "k8s.io/client-go/pkg/apis/extensions/install"
	"log"
	"os"
	"regexp"
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

	regExp := regexp.MustCompile(`.*(\.yml|\.yaml)`)

	acceptedK8sTypes := regexp.MustCompile(`(Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount)`)

	for _, f := range files {
		isYaml := regExp.MatchString(f.Name())

		if !f.IsDir() && isYaml {
			fileR, err := ioutil.ReadFile(f.Name())
			if err != nil {
				log.Printf("An error occurred while reading file %s from directory %s. Err: %s", f.Name(), customDir, err)
				return
			}
			//decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(fileR), 100)
			decode := api.Codecs.UniversalDeserializer().Decode
			obj, groupVersionKind, err := decode(fileR, nil, nil)
			if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
				log.Println("The custom-roles ")
			}
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
	if envDir != "" {
		dir = envDir
	}
	return dir
}
