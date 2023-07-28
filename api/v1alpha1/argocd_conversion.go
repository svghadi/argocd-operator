package v1alpha1

import (
	v1beta1 "github.com/argoproj-labs/argocd-operator/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var conversionLogger = ctrl.Log.WithName("argocd-conversion")

// ConvertTo converts this CronJob to the Hub version (v1).
func (src *ArgoCD) ConvertTo(dstRaw conversion.Hub) error {
	conversionLogger.Info("Converting to beta")
	dst := dstRaw.(*v1beta1.ArgoCD)
	dst.ObjectMeta = src.ObjectMeta
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *ArgoCD) ConvertFrom(srcRaw conversion.Hub) error {
	conversionLogger.Info("Converting to alpha")
	src := srcRaw.(*v1beta1.ArgoCD)
	dst.ObjectMeta = src.ObjectMeta
	return nil
}
