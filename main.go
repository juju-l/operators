package main

import (
			"context"
			metasv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
			"k8s.io/client-go/kubernetes"
			"k8s.io/client-go/tools/clientcmd"
			"k8s.io/client-go/tools/leaderelection"
			"k8s.io/client-go/tools/leaderelection/resourcelock"
			//"k8s.io/client-go/rest"
			"os"
			"time"
)

func main() {
			//sigCh := make(chan os.Signal, 1)
			ctx, cancel := context.WithCancel(context.Background())
			cfg, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG") /**/)

			kubeCli := kubernetes.NewForConfigOrDie( cfg )
			//discoverysCli := discovery.NewDiscoveryClientForConfigOrDie( cfg )
			//dyCli := dynamic.NewForConfigOrDie( cfg )

			///factory := dynamicinformer.NewDynamicSharedInformerFactory(/**/ dyCli, 0)-|-|-|-|-|-|-|-|-|-|-

			ldLck := &resourcelock.LeaseLock{
			LeaseMeta: metasv1.ObjectMeta{
			Name: "hlm-operators-lck",
			Annotations: map[string]string{},
			Namespace: "default",
			},
			Client: kubeCli.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
			Identity: os.Getenv("POD_NAME"),
			},
			}
			kdFuc := leaderelection.LeaderCallbacks{
			OnNewLeader: func(identity string) {
			
			},
			OnStartedLeading: func(ctx context.Context) {
			New(ctx, cancel, 3).run()
			},
			OnStoppedLeading: func() {
			os.Exit(0)
			},
			}
			ldCfg := leaderelection.LeaderElectionConfig{
			Lock: ldLck,
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod: 2 * time.Second,
			Callbacks: kdFuc,
			}

			//if lec,err:=leaderelection.NewLeaderElector(ldCfg); err!=nil {
			//panic(err)
			//}

			//
			///
			leaderelection.RunOrDie(ctx, ldCfg /**/)
			///
			//

			//<-sigCh
			<-ctx.Done()
			cancel()
}

// type

// var