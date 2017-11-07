# Kubernetes Gitlab Integrator Service

[![Go Report Card](https://goreportcard.com/badge/github.com/k8s-tamias/gitlab-k8s-integrator)](https://goreportcard.com/report/github.com/k8s-tamias/gitlab-k8s-integrator)
[![GoDoc](https://godoc.org/github.com/k8s-tamias/gitlab-k8s-integrator?status.svg)](https://godoc.org/github.com/k8s-tamias/gitlab-k8s-integrator)

This service consumes Gitlab Webhook Calls and translates them into namespaces and roles in Kubernetes. Every Gitlab Group,
Project and User Repository (Private Namespace) is turned into a namespace on Kubernetes.

Users in Gitlab are then bound to the roles according to their membership in Gitlab. A change has immediate effect to K8s
due to the use of Gitlab Webhooks. Additionally the Integrator has a recurring job which synchronizes the state of Gitlab
with Kubernetes to ensure that the two systems do not drift appart due to lost events. In case of a conflict, Gitlab will act as the authorative system.

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
- If namespace is present by name, but does not have a gitlab-origin label attached AND is not(!) labeled with
 'gitlab-ignored' it get's labeled with its origin name.

### Webhook Feature

This service provides an endpoint which, if registered with [Gitlab for System Hooks](https://docs.gitlab.com/ee/system_hooks/system_hooks.html), provides support for
all the push events which refer to the state of Groups, Projects and Users as well as their respective members. So every 
change in Gitlab will be directly and promptly reflected through this webhook.

The endpoint is: **/hook**

Note: Renaming Groups or Projects and transferring projects results in the deletion and recreation of the corresponding namespace
in Kubernetes. Thus all contents of the original namespace will be deleted as well.

In the case that the service is offline of for some other reason misses a webhook call, a sync mechanism is provided (see below).

### Sync Feature

In addition to the webhook feature a recurring sync task is being executed every 3 hours, which
synchronizes Gitlab with the K8s Cluster according to the following algorithm:

1. Delete all Namespaces, which are present in the K8s Cluster, but do not correspond to an entity in Gitlab.
(This is ensured by using the "gitlab-origin" label on each created namespace, which contains the original name of the entity from gitlab).
This does not touch namespaces unrelated to Gitlab (i.e. that do not match with a Gitlab name after its transformation)
2. Iterate all Gitlab entities (Users, Groups and Projects) and for each 
    1. Create namespace, if not present
    2. Iterate all Members and for each:
        1. Create a RoleBinding corresponding to the role in Gitlab (see below for details)
        2. Delete RoleBindings for Members which are no longer present in the Gitlab Entity
        3. Adjust RoleBindings for Members whose Role has changed
    3. Create ceph-secret-user in the namespace, if ENV CEPH_USER_KEY has been set

#### Prevent namespace from being synced
If you don't want a specific namespace to be synced with gitlab, just add a 'gitlab-ignored' label with an arbitrary value to
the namespace. The integrator will then not attempt to sync it.      

#### Add custom roles and bindings
Sometimes additional roles and bindings beyond those defined for the gitlab cluster roles are required (i.e. a ServiceAccount 
with elevated permissions for some special project in a certain namespace). If you keep the sync feature of this 
service enabled for the namespace you want to have custom roles and bindings in, these will be deleted upon every sync run
as they are seen as invalid since they are not present in Gitlab by any means.

Therefore, if you want to add custom roles and bindings you may add them as a ConfigMap object to your cluster and mount 
that object into the /etc/custom-roles folder of the Pod running the Gitlab-K8s-Integrator.

The ConfigMap must contain at least one key which is a valid YAML file. It may contain as many files as you like. This should help
to structure things. Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-roles-and-bindings
  namespace: integration
  labels:
    service: gl-k8s-integrator
data:
  nodeAdminRBAC.yaml: |-
   kind: ClusterRole
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: nodes-reader
   rules:
   - apiGroups:
     - ""
     resources:
      - nodes
      - events
     verbs:
      - get
      - list
      - watch
   ---
   # This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
   kind: ClusterRoleBinding
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: bob-read-nodes-global
   subjects:
   - kind: User
     name: bob
     apiGroup: rbac.authorization.k8s.io
   roleRef:
     kind: ClusterRole
     name: nodes-reader
     apiGroup: rbac.authorization.k8s.io
   ---
   # This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
   kind: ClusterRoleBinding
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: john-read-nodes-global
   subjects:
   - kind: User
     name: john
     apiGroup: rbac.authorization.k8s.io
   roleRef:
     kind: ClusterRole
     name: nodes-reader
     apiGroup: rbac.authorization.k8s.io
  loggingAdminRBAC.yaml: |-
   kind: Role
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     namespace: logging
     name: logging-admin
   rules:
   - apiGroups:
      - "*"
     resources:
      - "*"
     verbs:
      - "*"
   ---
   # This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
   kind: RoleBinding
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: bob-logging-admin-role
     namespace: logging
   subjects:
   - kind: User
     name: bob
     apiGroup: rbac.authorization.k8s.io
   roleRef:
     kind: Role
     name: logging-admin
     apiGroup: rbac.authorization.k8s.io
```

Then mount it like so:

```yaml
kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: gl-k8s-integrator
  namespace: integration
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: gl-k8s-integrator
    spec:
      serviceAccount: gl-k8s-integrator
      volumes:
        - name: custom-roles
          configMap:
            name: custom-roles-and-bindings
      containers:
      - name: gl-k8s-integrator
        image: my-gl-k8s-image:stable
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: custom-roles
          mountPath: /etc/custom-roles
        resources:
          requests:
            cpu: 250m
            memory: 250Mi
        env:
        - name: ENABLE_GITLAB_HOOKS_DEBUG
          value: "false"
        - name: ENABLE_GITLAB_SYNC_DEBUG
          value: "false"
        - name: GITLAB_HOSTNAME
          value: "my-gitlab.example.com"
        - name: GITLAB_API_VERSION
          value: "v4"
        - name: ENABLE_SYNC_ENDPOINT
          value: "false"
        - name: GITLAB_PRIVATE_TOKEN
          valueFrom:
            secretKeyRef:
              name: gitlab-integrator-private-token
              key: token
        - name: CEPH_USER_KEY
          valueFrom:
            secretKeyRef:
              name: ceph-user-key
              key: key
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
```

### CEPH Secret User Features
In order to allow for all namespaces to access a DefaultStorageClass of type CEPH, this 
service will automatically create a ceph-secret-user Secret in every created namespace if 
ENV 'CEPH_USER_KEY' is set. (see below)

### Config ENV Variables

| ENV        | Required? | Description           | 
|:-------------:|:-------------:|:-------------:|
|GITLAB_HOSTNAME | yes | The hostname of the Gitlab server to work with
|GITLAB_API_VERSION| no (default: v4) | The Version of the Gitlab API to use.
|GITLAB_PRIVATE_TOKEN| yes | The private access token from a Gitlab admin user to use when calling the API
|CEPH_USER_KEY| no | The key of the ceph-secret-user secret. The secret only gets created if this variable is set.
|ENABLE_SYNC_ENDPOINT| no|If set to 'true' this will enable a /sync endpoint, which may be triggered with a PUSH REST call to start a sync run. (USE WITH CAUTION, may be abused!)
|ENABLE_GITLAB_HOOKS_DEBUG| no| If set to 'true' the raw hooks messages get printed to stdout upon receiving, Default: no
|ENABLE_GITLAB_SYNC_DEBUG| no| If set to 'true' the sync process will output debug info

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
  name: gitlab-group-developer
rules:
- apiGroups:
   - ""
  resources:
   - events
   - persistentvolumeclaims
   - pods
   - pods/status
   - pods/logs
   - services
   - services/proxy
  verbs:
   - get
   - list
   - watch
- apiGroups:
   - ""
  resources:
   - pods/portforward
   - pods/exec
  verbs:
   - get
   - watch
   - list
   - create
   - update
   - patch
   - delete
```

#### Recommended Master Role<a name="masterrole"></a>
```yaml
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: gitlab-project-master
rules:
- apiGroups:
   - ""
  resources:
   - configmaps
   - pods
   - pods/logs
   - pods/attach
   - pods/exec
   - pods/portforward
   - persistentvolumeclaims
   - secrets
   - services
   - services/proxy
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
   - pods/proxy
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
