package usecases

import (
	"io/ioutil"
	"k8s.io/client-go/pkg/api"
	_ "k8s.io/client-go/pkg/api/install"
	_ "k8s.io/client-go/pkg/apis/extensions/install"
	_ "k8s.io/client-go/pkg/apis/rbac/install"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/rbac/v1beta1"
	"log"
	"os"
	"regexp"
	"k8s.io/apimachinery/pkg/runtime"
	"fmt"
	"strings"
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



	for _, f := range files {
		isYaml := regExp.MatchString(f.Name())

		if !f.IsDir() && isYaml {
			fileR, err := ioutil.ReadFile(f.Name())
			if err != nil {
				log.Printf("An error occurred while reading file %s from directory %s. Err: %s", f.Name(), customDir, err)
				return
			}


			objects := ParseK8sYaml(fileR)
			for _, o := range objects {
				switch o := o.(type) {
				// TODO: create custom objects which are compatible to GitlabContent stuff in order to be used by sync usecase
				case *v1beta1.Role:
				case *v1beta1.RoleBinding:
				case *v1beta1.ClusterRole:
				case *v1beta1.ClusterRoleBinding:
				case *v1.ServiceAccount:
				}
			}


		}
	}
}

func ParseK8sYaml(fileR []byte) []runtime.Object {

	acceptedK8sTypes := regexp.MustCompile(`(Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object,0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := api.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(f), nil, nil)

		if err != nil {
			log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.Printf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			retVal = append(retVal, obj)
		}

	}
	return retVal
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
