package k8sclient

import (
	"os"
	"strconv"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"log"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func CreateLimitRange(namespace string) {
	if namespace == "" {
		return
	}
	enableLimitRanges, err := strconv.ParseBool(os.Getenv("ENABLE_LIMITRANGES"))
	if err != nil || !enableLimitRanges {
		return
	}

	defaultCpuReq, err := strconv.Atoi(os.Getenv("DEFAULT_CPU_REQ"))
	if err != nil || defaultCpuReq <= 0 {
		defaultCpuReq = 20
	}
	defaultCpuReqQty := resource.NewQuantity(int64(defaultCpuReq), resource.BinarySI)

	defaulCpuLimit, err := strconv.Atoi(os.Getenv("DEFAULT_CPU_LIM"))
	if err != nil || defaulCpuLimit <= 0 {
		defaulCpuLimit = 150
	}
	defaultCpuLimitQty := resource.NewQuantity(int64(defaulCpuLimit), resource.BinarySI)

	defaultMemReq, err := strconv.Atoi(os.Getenv("DEFAULT_MEM_REQ"))
	if err != nil || defaultMemReq <= 0 {
		defaultMemReq = 25
	}
	defaultMemReqQty := resource.NewQuantity(int64(defaultMemReq), resource.BinarySI)

	defaultMemLimit, err := strconv.Atoi(os.Getenv("DEFAULT_MEM_LIM"))
	if err != nil || defaultMemLimit <= 0 {
		defaultMemLimit = 120
	}

	defaultMemLimitQty := resource.NewQuantity(int64(defaultMemLimit), resource.BinarySI)

	// build LimitRange
	lR := &v1.LimitRange{Spec: v1.LimitRangeSpec{
		Limits:[]v1.LimitRangeItem{
			{Type: "Container",
			DefaultRequest: v1.ResourceList{v1.ResourceMemory: *defaultMemReqQty, v1.ResourceCPU: *defaultCpuReqQty},
			Default: v1.ResourceList{v1.ResourceMemory: *defaultMemLimitQty, v1.ResourceCPU: *defaultCpuLimitQty}},
		}}}

	// write to Cluster
	client := getK8sClient()
	_, err = client.CoreV1().LimitRanges(namespace).Create(lR)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		log.Fatalf("Error creating LimitRange for Namespace %s. Error was: %s", namespace, err.Error())
	}
}