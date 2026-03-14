package client

import (
	"context"
	"time"

	"tst-operator/apis/example.com/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func NewTstInformer(client *TstClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return client.Tsts(metav1.NamespaceAll).List(context.TODO(),opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return client.Tsts(metav1.NamespaceAll).Watch(context.TODO(),opts)
			},
		},
		&v1beta1.Tst{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, // 添加命名空间索引 cache.Indexers{},
	)
}