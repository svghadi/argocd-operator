package cacheutils

import (
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	oappsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/argoproj-labs/argocd-operator/common"
)

var (
	watchedByArgoCDSelector = labels.SelectorFromSet(
		labels.Set{common.WatchedByOperatorKey: common.ArgoCDAppName},
	)
)

func SetupCache() cache.Options {
	cacheOpts := cache.Options{}

	//if watchedNsCache := getDefaultWatchedNamespacesCacheOptions(); watchedNsCache != nil {
	//	cacheOpts.DefaultNamespaces = watchedNsCache
	//}

	//managedLabelReq, _ := labels.NewRequirement("reconcile.external-secrets.io/managed", selection.Equals, []string{"true"})

	cacheOpts.ByObject = map[client.Object]cache.ByObject{
		// Core APIs
		&corev1.Pod{}:                            {Label: watchedByArgoCDSelector},
		&corev1.ServiceAccount{}:                 {Label: watchedByArgoCDSelector},
		&corev1.Service{}:                        {Label: watchedByArgoCDSelector},
		&corev1.Secret{}:                         {Label: watchedByArgoCDSelector},
		&corev1.ConfigMap{}:                      {Label: watchedByArgoCDSelector},
		&rbacv1.Role{}:                           {Label: watchedByArgoCDSelector},
		&rbacv1.RoleBinding{}:                    {Label: watchedByArgoCDSelector},
		&rbacv1.ClusterRole{}:                    {Label: watchedByArgoCDSelector},
		&rbacv1.ClusterRoleBinding{}:             {Label: watchedByArgoCDSelector},
		&appsv1.Deployment{}:                     {Label: watchedByArgoCDSelector},
		&appsv1.StatefulSet{}:                    {Label: watchedByArgoCDSelector},
		&appsv1.ReplicaSet{}:                     {Label: watchedByArgoCDSelector},
		&autoscalingv1.HorizontalPodAutoscaler{}: {Label: watchedByArgoCDSelector},
		&networkingv1.Ingress{}:                  {Label: watchedByArgoCDSelector},
		&batchv1.CronJob{}:                       {Label: watchedByArgoCDSelector},
		&batchv1.Job{}:                           {Label: watchedByArgoCDSelector},
		// Prometheus APIs
		&monitoringv1.Prometheus{}:     {Label: watchedByArgoCDSelector},
		&monitoringv1.ServiceMonitor{}: {Label: watchedByArgoCDSelector},
		&monitoringv1.PrometheusRule{}: {Label: watchedByArgoCDSelector},
		// OpenShift APIs
		&oappsv1.DeploymentConfig{}: {Label: watchedByArgoCDSelector},
		&routev1.Route{}:            {Label: watchedByArgoCDSelector},
	}

	return cacheOpts
}

func setupCacheClient() client.Options {
	cacheClientOpts := client.Options{
		Cache: &client.CacheOptions{
			// operator doesn't watch/react to these resources so there is no need to cache them. This will save memory.
			DisableFor: []client.Object{
				//&corev1.Secret{
				//	ObjectMeta: metav1.ObjectMeta{
				//		Labels: map[string]string{
				//			"kubernetes.io/service-account.name": //"test-argocd-dex-server",
				//		},
				//	},
				//},
				//&k8sappsv1.ReplicaSet{},
			},
		},
	}
	return cacheClientOpts
}
