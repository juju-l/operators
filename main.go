package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	kubeconfig := os.Getenv("KUBECONFIG")
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// allow flags to override env
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file (optional)")
	flag.StringVar(&namespace, "namespace", namespace, "Operator namespace")
	flag.Parse()

	ctx := context.Background()

	// create helm client
	helm, err := NewHelmClientInCluster(namespace, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create helm client: %v\n", err)
		os.Exit(1)
	}

	// create controller
	ctrl, err := NewController(kubeconfig, namespace, helm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create controller: %v\n", err)
		os.Exit(1)
	}

	stopCh := make(chan struct{})
	// run controller in a goroutine
	go func() {
		if err := ctrl.Run(ctx, stopCh); err != nil {
			fmt.Fprintf(os.Stderr, "controller exited with error: %v\n", err)
			os.Exit(1)
		}
	}()

	// handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("shutting down operator...")
	close(stopCh)
	// give some time for shutdown
	time.Sleep(1 * time.Second)
}
