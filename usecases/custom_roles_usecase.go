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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type CustomRolesAndBindings struct {
	Roles    				map[string]bool
	RoleBindings 			map[string]bool
	ClusterRoles 			map[string]bool
	ClusterRoleBindings 	map[string]bool
	ServiceAccounts 		map[string]bool
}

func ReadAndApplyCustomRolesAndBindings() CustomRolesAndBindings {
	res := CustomRolesAndBindings{}

	customDir := getCustomRoleDir()
	customRolesPresent, err := fileExists(customDir)
	if err != nil {
		log.Printf("An error occurred while trying to read custom roles from directory %s. Err: %s", customDir, err)
		return res
	}
	if !customRolesPresent {
		log.Println("No custom-roles directory present, skipping step...")
		return res
	}

	files, err := ioutil.ReadDir(customDir)
	if err != nil {
		log.Printf("An error occurred while trying to read custom roles from directory %s. Err: %s", customDir, err)
		return res
	}

	regExp := regexp.MustCompile(`.*(\.yml|\.yaml)`)
	k8sclient := getK8sClient()
	for _, f := range files {
		isYaml := regExp.MatchString(f.Name())

		if !f.IsDir() && isYaml {
			fileR, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", customDir, f.Name()))
			if err != nil {
				log.Printf("An error occurred while reading file %s from directory %s. Err: %s", f.Name(), customDir, err)
				return res
			}


			objects := parseK8sYaml(fileR)
			for _, o := range objects {
				switch o := o.(type) {
				case *v1beta1.Role:
					res.Roles[o.Name] = true
					k8sclient.RbacV1beta1().Roles(o.Namespace).Create(o)
					log.Printf("Applied Custom Role %s in Namespace %s", o.Name, o.Namespace)
				case *v1beta1.RoleBinding:
					res.RoleBindings[o.Name] = true
					k8sclient.RbacV1beta1().RoleBindings(o.Namespace).Create(o)
					log.Printf("Applied Custom RoleBinding %s in Namespace %s", o.Name, o.Namespace)
				case *v1beta1.ClusterRole:
					res.ClusterRoles[o.Name] = true
					k8sclient.RbacV1beta1().ClusterRoles().Create(o)
					log.Printf("Applied Custom ClusterRole %s", o.Name)
				case *v1beta1.ClusterRoleBinding:
					res.ClusterRoleBindings[o.Name] = true
					k8sclient.RbacV1beta1().ClusterRoleBindings().Create(o)
					log.Printf("Applied Custom ClusterRoleBinding %s", o.Name)
				case *v1.ServiceAccount:
					res.ServiceAccounts[o.Name] = true
					k8sclient.CoreV1().ServiceAccounts(o.Namespace).Create(o)
					log.Printf("Applied Custom ServiceAccount %s in Namespace %s", o.Name, o.Namespace)
				}
			}
		}
	}
	return res
}

func getK8sClient() *kubernetes.Clientset {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if check(err) {
		log.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if check(err) {
		log.Fatal(err)
	}
	return clientset
}

func parseK8sYaml(fileR []byte) []runtime.Object {

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
