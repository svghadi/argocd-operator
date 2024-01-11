Steps to reproduce https://github.com/argoproj/argo-cd/issues/9070 

1. Install operator using `make install run`
2. Create webhook server `kubectl apply -f 00-setup-webhook-server.yaml`
3. Create ArgoCD instance `kubectl apply -f 01-install-argocd.yaml`
4. Create notification configs `kubectl apply -f 02-update-notifications-cm.yaml`
5. Create application `kubectl apply -f app.yaml`

6. Watch the notification controller & webhook server logs
   ```bash
   kubectl logs --follow deployment.apps/example-notifications-controller
   ```

   ```bash
   kubectl logs --follow deployment.apps/webhook
   ```

7. Change application image tag to non existing one to make app unhealthy 
   > NOTE  
   The issue doesn't always but often enough trigger false positives. Changing image tag back and forth to existing & non-existing a couple of times will trigger false positive notification. 