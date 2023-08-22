# Reproducer for [GITOPS-2867](https://issues.redhat.com/browse/GITOPS-2867)

## 1. Deploy a SMTP server

```yaml
kind: Namespace
apiVersion: v1
metadata:
  name: smtp
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: smtp4dev
  namespace: smtp
  labels:
    app: smtp4dev
spec:
  replicas: 1
  selector:
    matchLabels:
      app: smtp4dev
  template:
    metadata:
      labels:
        app: smtp4dev
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: kubernetes.io/os
                    operator: In
                    values:
                      - linux
      containers:
        - name: smtp4dev
          image: quay.io/argoprojlabs/argocd-notifications-e2e-smtplistener:multiarch
          ports:
            - containerPort: 80
            - containerPort: 2525
---
apiVersion: v1
kind: Service
metadata:
  name: smtp4dev
  namespace: smtp
spec:
  selector:
    app: smtp4dev
  ports:
    - name: smtp
      protocol: TCP
      port: 2525
      targetPort: 2525
    - name: http
      protocol: TCP
      port: 80
      targetPort: 80
```

## 2. Enable notification in gitops-operator

Edit default `openshift-gitops` argocd instance in `openshift-gitops` namespace.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ArgoCD
metadata:
  name: openshift-gitops
  namespace: openshift-gitops
spec:
  notifications:
    enabled: true
```

## 3. Update `argocd-notifications-cm` configmap

#### Use `.repo` function in notification template. Update `template.app-deployed` key with below.

```yaml
  template.app-deployed: |-
    email:
      subject: New version of an application {{(call .repo.GetAppDetails).Type}} is up and running.
    message: |
      {{if eq .serviceType "slack"}}:white_check_mark:{{end}} Application {{(call .repo.GetAppDetails)}} is now running new version of deployments manifests.
    slack:
      attachments: |
        [{
          "title": "{{ .app.metadata.name}}",
          "title_link":"{{.context.argocdUrl}}/applications/{{.app.metadata.name}}",
          "color": "#18be52",
          "fields": [
          {
            "title": "Sync Status",
            "value": "{{.app.status.sync.status}}",
            "short": true
          },
          {
            "title": "Repository",
            "value": "{{.app.spec.source.repoURL}}",
            "short": true
          },
          {
            "title": "Revision",
            "value": "{{.app.status.sync.revision}}",
            "short": true
          }
          {{range $index, $c := .app.status.conditions}}
          {{if not $index}},{{end}}
          {{if $index}},{{end}}
          {
            "title": "{{$c.type}}",
            "value": "{{$c.message}}",
            "short": true
          }
          {{end}}
          ]
        }]
      deliveryPolicy: Post
      groupingKey: ""
      notifyBroadcast: false
    teams:
      facts: |
        [{
          "name": "Sync Status",
          "value": "{{.app.status.sync.status}}"
        },
        {
          "name": "Repository",
          "value": "{{.app.spec.source.repoURL}}"
        },
        {
          "name": "Revision",
          "value": "{{.app.status.sync.revision}}"
        }
        {{range $index, $c := .app.status.conditions}}
          {{if not $index}},{{end}}
          {{if $index}},{{end}}
          {
            "name": "{{$c.type}}",
            "value": "{{$c.message}}"
          }
        {{end}}
        ]
      potentialAction: |-
        [{
          "@type":"OpenUri",
          "name":"Operation Application",
          "targets":[{
            "os":"default",
            "uri":"{{.context.argocdUrl}}/applications/{{.app.metadata.name}}"
          }]
        },
        {
          "@type":"OpenUri",
          "name":"Open Repository",
          "targets":[{
            "os":"default",
            "uri":"{{.app.spec.source.repoURL | call .repo.RepoURLToHTTPS}}"
          }]
        }]
      themeColor: '#000080'
      title: New version of an application {{.app.metadata.name}} is up and running.
```

#### Add gmail service

```yaml
service.email.gmail: '{host: smtp4dev.smtp.svc, port: 2525, from: fake@email.com }'
```

## 4. Create a sample app

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app-3
  namespace: openshift-gitops
  annotations:
    "notifications.argoproj.io/subscribe.on-deployed.gmail": "jdfake@email.com"
spec:
  destination:
    namespace: openshift-gitops
    server: https://kubernetes.default.svc
  project: default
  source:
    repoURL: https://github.com/redhat-developer/gitops-operator
    path: test/examples/nginx
    targetRevision: HEAD
```

## 5. Perform a sync from ArgoCD UI

## 6. Look for errors in notification controller pods

```
time="2023-08-22T08:25:23Z" level=error msg="Failed to notify recipient {gmail jdfake@email.com} defined in resource openshift-gitops/my-app-3: template: app-deployed:1:71: executing \"app-deployed\" at <call .repo.GetAppDetails>: error calling call: rpc error: code = Unavailable desc = connection error: desc = \"transport: Error while dialing dial tcp: lookup argocd-repo-server on 172.30.0.10:53: no such host\"" resource=openshift-gitops/my-app-3
```

## 7. Check smtp server pod

Mail will not be sent.

```
$ cat /tmp/a*
```
