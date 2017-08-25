# Kubernetes Gitlab Integrator Service

This service consumes Gitlab Webhook Calls and translates them into namespaces and roles in Kubernetes. 
Users in Gitlab are then bound to the roles according to their membership in Gitlab. A change has immediate effect to K8s
due to the use of Gitlab Webhooks. Additionally the Integrator has a recurring job which synchronizes the state of Gitlab
with Kubernetes to make 

### Namespaces will be created according to the following scheme:


| Gitlab        | K8s           | Example |
| ------------- |:-------------:|:-------:| 
| Personal Repositories| Namespace of the same name | student-Bob -> student-bob
| Groups        | Namespace of the same name | Foo-Group -> foo-group
| Sub-Groups | Namespaces of the same name, prefixed with "$GroupName$-". | Foo-Group/bar-subgroup -> foo-group-bar-subgroup    
| Projects | Namespace of the same name, prefixed with "$GroupName$-" and "$SubGroupName$-" if applicable | Foo-Group/bar-subgroup/MyProject -> foo-group-bar-subgroup-myproject 

#### Additional Rules:
- All Gitlab Names will be lower cased in K8s 
- If a namespace-name is already taken due to group and sub-group concatenation (e.g. foo-group/bar-project vs. foo-group-bar-project as single group name) 
a counter will be added to at the end of the namespace name with a "-" as prefix.

### Roles will be created according to the following schema

| Gitlab        | K8s           | 
| ------------- |:-------------:|
|Guest | nothing 
|Reporter | Get, List, Watch for Pods & Pods/Logs
|Developer | same as Reporter
|Master | see [Master Role](#masterrole)

#### Master Role<a name="masterrole"></a>
```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: gitlab-group-master
rules:
- apiGroups:
   - ""
  resources:
   - configmaps
   - pods
   - pods/attach
   - pods/exec
   - pods/portforward
   - persistentvolumeclaims
   - secrets
   - services
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
   - persistentvolumes
   - pods/status
   - pods/log
  verbs:
   - get
   - list
   - watch
- apiGroups:
   - apps
  resources:
   - statefulsets
   - deployments
  verbs:
   - get
   - watch
   - list
   - create
   - update
   - patch
   - delete
- apiGroups:
   - extensions
  resources:
   - deployments
   - deployments/rollback
   - deployments/scale
   - ingresses
   - replicasets
   - replicasets/scale
  verbs:
   - create
   - delete
   - deletecollection
   - get
   - list
   - patch
   - update
   - watch
```
