package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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

	// Recorder placeholder
	recorder := record.NewBroadcaster()
	recorder.StartStructuredLogging(0)
	_ = recorder

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
				startController(ctx, dynClient, gvr, workers)
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

func startController(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource, workers int) {
	factory := informers.NewSharedInformerFactoryWithOptions(nil, 0)
	// dynamic informer not directly via factory; use dynamicinformer in full impl
	// ... simplified for skeleton

	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tst-controller")

	// placeholder informer handlers for example
	handleAdd := func(obj interface{}) {
		metaObj := obj.(metav1.Object)
		key := metaObj.GetNamespace() + "/" + metaObj.GetName()
		queue.Add(key)
	}
	// ... real informer wiring omitted in skeleton

	for i := 0; i < workers; i++ {
		go func() {
			for {
				item, shutdown := queue.Get()
				if shutdown {
					return
				}
				key := item.(string)
				// process key
				klog.Infof("processing %s", key)
				queue.Done(item)
				queue.Forget(item)
			}
		}()
	}

	<-ctx.Done()
	queue.ShutDown()
}
