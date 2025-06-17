/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/argoproj/argo-cd/v2/util/env"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/argoproj-labs/argocd-operator/common"
	"github.com/argoproj-labs/argocd-operator/controllers/argocd"
	"github.com/argoproj-labs/argocd-operator/controllers/argocdexport"
	"github.com/argoproj-labs/argocd-operator/controllers/argoutil"
	"github.com/argoproj-labs/argocd-operator/controllers/cacheutils"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	notificationsConfig "github.com/argoproj-labs/argocd-operator/controllers/notificationsconfiguration"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1beta1 "github.com/argoproj-labs/argocd-operator/api/v1beta1"
	"github.com/argoproj-labs/argocd-operator/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func printVersion() {
	setupLog.Info(fmt.Sprintf("Go Version: %s", goruntime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", goruntime.GOOS, goruntime.GOARCH))
	setupLog.Info(fmt.Sprintf("Version of %s-operator: %v", common.ArgoCDAppName, version.Version))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var labelSelectorFlag string

	var secureMetrics = false
	var enableHTTP2 = false

	flag.StringVar(&metricsAddr, "metrics-bind-address", fmt.Sprintf(":%d", common.OperatorMetricsPort), "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&labelSelectorFlag, "label-selector", env.StringFromEnv(common.ArgoCDLabelSelectorKey, common.ArgoCDDefaultLabelSelector), "The label selector is used to map to a subset of ArgoCD instances to reconcile")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the metrics and webhook servers.")
	flag.BoolVar(&secureMetrics, "metrics-secure", secureMetrics, "If the metrics endpoint should be served securely.")

	//Configure log level
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	logLevel := zapcore.InfoLevel
	switch logLevelStr {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	case "panic":
		logLevel = zapcore.PanicLevel
	case "fatal":
		logLevel = zapcore.FatalLevel
	}

	opts := zap.Options{
		Level:       logLevel,
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	disableHTTP2 := func(c *tls.Config) {
		if enableHTTP2 {
			return
		}
		c.NextProtos = []string{"http/1.1"}
	}
	webhookServerOptions := webhook.Options{
		TLSOpts: []func(config *tls.Config){disableHTTP2},
		Port:    9443,
	}
	webhookServer := webhook.NewServer(webhookServerOptions)

	metricsServerOptions := metricsserver.Options{
		SecureServing: secureMetrics,
		BindAddress:   metricsAddr,
		TLSOpts:       []func(*tls.Config){disableHTTP2},
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	// Check the label selector format eg. "foo=bar"
	if _, err := labels.Parse(labelSelectorFlag); err != nil {
		setupLog.Error(err, "error parsing the labelSelector '%s'.", labelSelectorFlag)
		os.Exit(1)
	}
	setupLog.Info(fmt.Sprintf("Watching labelselector \"%s\"", labelSelectorFlag))

	// Inspect cluster to verify availability of extra features
	if err := argocd.InspectCluster(); err != nil {
		setupLog.Info("unable to inspect cluster")
	}

	namespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get watch namespace, defaulting to all namespace mode")
	}
	setupLog.Info(fmt.Sprintf("Watching namespace \"%s\"", namespace))

	// Register the API groups to the scheme
	setupLog.Info("Registering API groups to the scheme")
	cacheutils.SetupScheme(scheme)

	// Create a manager with optimized cache and client options
	// Refer cache.go & scheme.go for details
	cache := cacheutils.SetupCache()
	//_cacheClient := setupCacheClient()

	options := manager.Options{
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "b674928d.argoproj.io",
		//Client:                 cacheClient,
		Cache: cache,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	liveClient, _ := client.New(mgr.GetConfig(), client.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		Cache:      nil,
	})
	clientWrapper := argoutil.NewClientWrapper(mgr.GetClient(), liveClient)

	// Setup all Controllers
	if err = (&argocd.ReconcileArgoCD{
		Client:        clientWrapper,
		Scheme:        mgr.GetScheme(),
		LabelSelector: labelSelectorFlag,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ArgoCD")
		os.Exit(1)
	}
	if err = (&argocdexport.ReconcileArgoCDExport{
		Client: clientWrapper,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ArgoCDExport")
		os.Exit(1)
	}
	if err = (&notificationsConfig.NotificationsConfigurationReconciler{
		Client: clientWrapper,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NotificationsConfiguration")
		os.Exit(1)
	}

	// Start webhook only if ENABLE_CONVERSION_WEBHOOK is set
	if strings.EqualFold(os.Getenv("ENABLE_CONVERSION_WEBHOOK"), "true") {
		if err = (&v1beta1.ArgoCD{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ArgoCD")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Monitor memory usage and cache stats
	go monitorSystemStats(mgr.GetCache(), mgr.GetScheme(), clientWrapper)

	// Start the manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getDefaultWatchedNamespacesCacheOptions() map[string]cache.Config {
	watchedNamespaces, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get watch namespace, defaulting to all namespace mode")
		return nil
	}

	if watchedNamespaces == "" {
		return nil
	}

	watchedNsList := strings.Split(watchedNamespaces, ",")
	setupLog.Info(fmt.Sprintf("Watching namespaces: %v", watchedNsList))

	defaultNamespacesCacheConfig := map[string]cache.Config{}
	for _, ns := range watchedNsList {
		defaultNamespacesCacheConfig[ns] = cache.Config{}
	}

	return defaultNamespacesCacheConfig
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// monitorSystemStats periodically prints memory usage and informer cache stats
func monitorSystemStats(c cache.Cache, s *runtime.Scheme, cw *argoutil.ClientWrapper) {
	var memStats goruntime.MemStats

	ctx := context.TODO()
	c.WaitForCacheSync(ctx)

	for {
		fmt.Printf("\n------------------------------------\n")
		// Memory stats
		goruntime.ReadMemStats(&memStats)
		fmt.Printf("Memory Usage: Alloc = %.2f MB, TotalAlloc = %.2f MB, Sys = %.2f MB, NumGC = %v\n",
			float64(memStats.Alloc)/(1024*1024),
			float64(memStats.TotalAlloc)/(1024*1024),
			float64(memStats.Sys)/(1024*1024),
			memStats.NumGC)

		// cache stats
		stats := c.GetCacheStats()
		count := 0
		size := 0
		for gvk, stat := range stats {
			fmt.Printf("Resource: %s, Count: %d, Size: %dKB\n", gvk, stat.Count, stat.Size/1024)
			count += stat.Count
			size += stat.Size
		}

		fmt.Printf("\nTotal Count: %d", count)
		fmt.Printf("\nTotal Size: %dKB", size/1024)
		fmt.Printf("\nLive API Calls: %d", cw.GetLiveCount())
		fmt.Printf("\n------------------------------------\n")

		// Sleep for a while before printing again
		time.Sleep(10 * time.Second)

	}
}
