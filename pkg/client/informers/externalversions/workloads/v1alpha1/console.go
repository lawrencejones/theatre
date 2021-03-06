/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	workloadsv1alpha1 "github.com/gocardless/theatre/pkg/apis/workloads/v1alpha1"
	versioned "github.com/gocardless/theatre/pkg/client/clientset/versioned"
	internalinterfaces "github.com/gocardless/theatre/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/gocardless/theatre/pkg/client/listers/workloads/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ConsoleInformer provides access to a shared informer and lister for
// Consoles.
type ConsoleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ConsoleLister
}

type consoleInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewConsoleInformer constructs a new informer for Console type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewConsoleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredConsoleInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredConsoleInformer constructs a new informer for Console type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredConsoleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.WorkloadsV1alpha1().Consoles(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.WorkloadsV1alpha1().Consoles(namespace).Watch(options)
			},
		},
		&workloadsv1alpha1.Console{},
		resyncPeriod,
		indexers,
	)
}

func (f *consoleInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredConsoleInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *consoleInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&workloadsv1alpha1.Console{}, f.defaultInformer)
}

func (f *consoleInformer) Lister() v1alpha1.ConsoleLister {
	return v1alpha1.NewConsoleLister(f.Informer().GetIndexer())
}
