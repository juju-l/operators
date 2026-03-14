package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juju-l/operators/tst-operator/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicinformer "k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	var kubeconfig string
	var masterURL string
	var metricsAddr string
	var leaderElectionNamespace string
	var leaderElectionID string
	var workers int

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig.")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "default", "namespace for leader election lease")
	flag.StringVar(&leaderElectionID, "leader-election-id", "tst-operator-lease", "id for leader election")
	flag.IntVar(&workers, "workers", 2, "number of worker goroutines")
	flag.Parse()

	// Build config
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("failed to build kube config: %v", err)
	}

	// Clients
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create kubernetes client: %v", err)
	}
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create dynamic client: %v", err)
	}

	// Setup signal handler
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		close(stopCh)
	}()

	// Setup leader election
	id, _ := os.Hostname()
	rlConfig := resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaderElectionID,
			Namespace: leaderElectionNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1beta1", Resource: "tsts"}

	// Start simple health/metrics server
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); w.Write([]byte("ok")) })
		http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); w.Write([]byte("ready")) })
		klog.Infof("metrics/health server listening on %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, nil); err != nil {
			klog.Errorf("metrics server exited: %v", err)
		}
	}()

	// Leader election callbacks
	ctx, cancel := context.WithCancel(context.Background())
	leConfig := leaderelection.LeaderElectionConfig{
		Lock:          &rlConfig,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Info("became leader, starting controller")

				// dynamic informer factory
				resync := 30 * time.Second
				dynFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynClient, resync)
				informer := dynFactory.ForResource(gvr).Informer()

				ctrl := controller.NewController(dynClient, gvr, informer)
				// start informers + controller
				go dynFactory.Start(ctx.Done())
				ctrl.Run(ctx, workers)
			},
			OnStoppedLeading: func() {
				klog.Info("stopped leading")
				cancel()
			},
			OnNewLeader: func(identity string) {
				klog.Infof("new leader: %s", identity)
			},
		},
	}

	leaderelection.RunOrDie(ctx, leConfig)

	<-stopCh
	klog.Info("shutting down")
}
