package usecases

import (
	"testing"
)

func TestParseK8sYaml(t *testing.T) {
	var deployment = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mars-group-serviceaccount
  namespace: abb256

---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: mars-group-sim-runner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: mars-sim-runner-role
subjects:
  - kind: ServiceAccount
    name: mars-group-serviceaccount
    namespace: abb256

---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: mars-sim-runner-role
rules:
- apiGroups:
   - ""
  resources:
   - pods
   - pods/logs
   - pods/attach
   - pods/exec
  verbs:
   - get
   - watch
   - list
   - create
   - update
   - patch
   - delete
- apiGroups:
   - ""
  resources:
   - events
   - nodes
   - pods/status
   - pods/log
   - pods/proxy
  verbs:
   - get
   - list
   - watch
- apiGroups:
   - batch
  resources:
   - cronjobs
   - jobs
  verbs:
   - create
   - delete
   - deletecollection
   - get
   - list
   - patch
   - update
   - watch

---
# Allow the mars group serviceAccount to use the priviliged PSP
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: mars-group-serviceaccount-psp
  namespace: abb256
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: privileged-psp-user
subjects:
  - kind: ServiceAccount
    name: mars-group-serviceaccount
    namespace: abb256
---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: daemon-set-allowance-role
rules:
- apiGroups:
   - apps
   - extensions
  resources:
   - daemonset
  verbs:
   - get
   - watch
   - list
   - create
   - update
   - patch
   - delete
---
# This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: abb256-daemon-set-allowance
  namespace: abb256
subjects:
- kind: User
  name: abb256
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: daemon-set-allowance-role
  apiGroup: rbac.authorization.k8s.io
`
	objects := parseK8sYaml([]byte(deployment))
	if objects == nil { t.Error("result was nil")}
	if len(objects) != 6 { t.Error("not enough objects deserialized")}

	var deployment2 = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mars-group-serviceaccount
  namespace: abb256
`
	objects = parseK8sYaml([]byte(deployment2))
	if objects == nil { t.Error("result was nil")}
	if len(objects) != 1 { t.Error("not enough objects deserialized")}

	var deployment3 = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mars-group-serviceaccount
  namespace: abb256
---
`
	objects = parseK8sYaml([]byte(deployment3))
	if objects == nil { t.Error("result was nil")}
	if len(objects) != 1 { t.Error("not enough objects deserialized")}

	var deployment4 = `
apiVersion: v1
kind: ServiceAccount
metadata:
 name: mars-group-serviceaccount
 namespace: mars

---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
 name: mars-group-sim-runner
roleRef:
 apiGroup: rbac.authorization.k8s.io
 kind: ClusterRole
 name: mars-sim-runner-role
subjects:
 - kind: ServiceAccount
   name: mars-group-serviceaccount
   namespace: mars

---`
	objects = parseK8sYaml([]byte(deployment4))
	if objects == nil { t.Error("result was nil")}
	if len(objects) != 2 { t.Error("not enough objects deserialized")}
}
