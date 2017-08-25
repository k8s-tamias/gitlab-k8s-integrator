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
- "_" and "." in Gitlab Names will be swapped for "-" in K8s Namespaces
- If a namespace-name is already taken due to group and sub-group concatenation (e.g. foo-group/bar-project vs. foo-group-bar-project as single group name) 
a counter will be added to at the end of the namespace name with a "-" as prefix. I.e.: Gitlab Group "foo_bar" becomes K8s namespace
"foo-bar". A new Gitlab group by the name of "foo.bar" would now become "foo-bar-1".
- To avoid wrong deletion, a label with `gitlab-origin` is added to each namespace which is used to discover the correct
namespace when attempting to delete a namespace.

### Roles and Permissions

We came up with a default for Roles and Persmissions as follows:

| Gitlab        | K8s           | 
| ------------- |:-------------:|
|Guest | nothing 
|Reporter | see [Report Role](#reporterrole)
|Developer | same as Reporter
|Master | see [Master Role](#masterrole)

The names of the ClusterRoles which get bound are defaulting to this scheme: `gitlab-<group|project>-<master|developer|reporter|guest>`
However you may change each Role name by setting the following ENV variables as you see fit:

| ENV        | Default           | 
|:-------------:|:-------------:|
|GROUP_MASTER_ROLENAME | gitlab-group-master
|GROUP_DEVELOPER_ROLENAME| gitlab-group-developer
|GROUP_REPORTER_ROLENAME| gitlab-group-reporter
|GROUP_DEFAULT_ROLENAME| gitlab-group-guest
|PROJECT_MASTER_ROLENAME|gitlab-project-master
|PROJECT_DEVELOPER_ROLENAME|gitlab-project-developer
|PROJECT_REPORTER_ROLENAME|gitlab-project-reporter
|PROJECT_DEFAULT_ROLENAME|gitlab-project-guest

#### RoleBinding Naming

The RoleBindings are created inside the affected namespace and are given a name which is created after the following scheme:
`username + rolename + namespace`

So User "foo" with Role "Master" in Group "bar" would become `foo-gitlab-group-master-bar`

#### Recommended Reporter Role<a name="reporterrole"></a>
```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: gitlab-group-reporter
rules:
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
```

#### Recommended Master Role<a name="masterrole"></a>
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
