package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"tst-operator/client"
	"tst-operator/controller"
)

func main() {
	var kubeconfig string
	home := os.Getenv("HOME")
	if home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig path")
	flag.Parse()

	// 1. 加载 k8s 配置
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	}

	// 2. 创建标准 kube client & 自定义 client
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	tstClient, err := client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// 3. 创建自定义资源 informer
	informer := client.NewTstInformer(tstClient, 10*time.Minute)

	// 4. Leader 选举（多副本高可用）
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "local-run"
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "tst-operator-lock",
			Namespace: "default",
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: podName,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				ctrl := controller.NewController(ctx, tstClient, informer, 3)
				ctrl.Start()
			},
			OnStoppedLeading: func() {
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
			},
		},
	})
}