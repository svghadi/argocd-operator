package cacheutils

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	oappsv1 "github.com/openshift/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"

	v1alpha1 "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	v1beta1 "github.com/argoproj-labs/argocd-operator/api/v1beta1"

	"github.com/argoproj-labs/argocd-operator/controllers/argocd"
)

// setupScheme registers necessary API groups to the given scheme.
func SetupScheme(scheme *runtime.Scheme) {
	registerCoreAPIs(scheme)
	registerArgoCDAPIs(scheme)
	registerPrometheusAPIsIfAvailable(scheme)
	registerOpenShiftAPIsIfAvailable(scheme)
	//setupLog.Info("Scheme setup complete.")
}

func registerCoreAPIs(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(autoscalingv1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
}

func registerArgoCDAPIs(scheme *runtime.Scheme) {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
}

func registerPrometheusAPIsIfAvailable(scheme *runtime.Scheme) {
	if argocd.IsPrometheusAPIAvailable() {
		utilruntime.Must(monitoringv1.AddToScheme(scheme))
	}
}

func registerOpenShiftAPIsIfAvailable(scheme *runtime.Scheme) {
	// Setup Scheme for OpenShift Routes if available.
	if argocd.IsRouteAPIAvailable() {
		utilruntime.Must(routev1.Install(scheme))
	}

	// Setup the scheme for openshift config if available
	if argocd.IsVersionAPIAvailable() {
		utilruntime.Must(configv1.Install(scheme))
	}

	// Setup Schemes for SSO if template instance is available.
	if argocd.CanUseKeycloakWithTemplate() {
		//setupLog.Info("Keycloak instance can be managed using OpenShift Template.")
		utilruntime.Must(oappsv1.Install(scheme))
		utilruntime.Must(templatev1.Install(scheme))
		utilruntime.Must(oauthv1.Install(scheme))
	}
	//else {
	//	setupLog.Info("Keycloak instance cannot be managed using OpenShift Template, as //DeploymentConfig/Template API is not present.")
	//}
}
