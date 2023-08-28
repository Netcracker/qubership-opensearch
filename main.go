package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/disasterrecovery"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	netcrackercomv1 "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	opensearchProtocolEnvVar = "OPENSEARCH_PROTOCOL"
	opensearchNameEnvVar     = "OPENSEARCH_NAME"
	opensearchUsernameEnvVar = "OPENSEARCH_USERNAME"
	opensearchPasswordEnvVar = "OPENSEARCH_PASSWORD"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netcrackercomv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	namespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Namespace:               namespace,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        fmt.Sprintf("opensearchservice.%s.netcracker.com", namespace),
		LeaderElectionNamespace: namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	var mutex sync.Mutex
	var mutexTwo sync.Mutex
	if err = (&controllers.OpenSearchServiceReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		ResourceHashes:        map[string]string{},
		ReplicationWatcher:    controllers.NewReplicationWatcher(&mutex),
		SlowLogIndicesWatcher: controllers.NewSlowLogIndicesWatcher(&mutexTwo),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenSearchService")
		os.Exit(1)
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

	opensearchName := os.Getenv(opensearchNameEnvVar)
	if opensearchName == "" {
		setupLog.Error(fmt.Errorf("%s must be set", opensearchName), "The operator can't work with Opensearch because Opensearch name is not given")
		os.Exit(1)
	}
	opensearchProtocol := os.Getenv(opensearchProtocolEnvVar)
	opensearchUsername := os.Getenv(opensearchUsernameEnvVar)
	opensearchPassword := os.Getenv(opensearchPasswordEnvVar)
	replicationChecker := disasterrecovery.NewReplicationChecker(opensearchName, opensearchProtocol, opensearchUsername, opensearchPassword)

	setupLog.Info("Starting disaster recovery REST server.")
	go func() {
		if err = disasterrecovery.StartServer(replicationChecker); err != nil {
			setupLog.Error(err, "Disaster recovery REST server cannot be created because of error")
			os.Exit(1)
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
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
