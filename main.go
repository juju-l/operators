package main

import (
	"flag"
	"path/filepath"

	"your/hlm-operator/controller"
	"your/hlm-operator/hlm"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig path")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	hlmClient, err := hlm.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ctrl := controller.NewController(hlmClient)
	stop := make(chan struct{})

	if err := ctrl.Run(2, stop); err != nil {
		panic(err)
	}
}