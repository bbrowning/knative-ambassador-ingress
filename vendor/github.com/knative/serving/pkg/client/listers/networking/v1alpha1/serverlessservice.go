/*
Copyright 2019 The Knative Authors

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
package v1alpha1

import (
	v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServerlessServiceLister helps list ServerlessServices.
type ServerlessServiceLister interface {
	// List lists all ServerlessServices in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ServerlessService, err error)
	// ServerlessServices returns an object that can list and get ServerlessServices.
	ServerlessServices(namespace string) ServerlessServiceNamespaceLister
	ServerlessServiceListerExpansion
}

// serverlessServiceLister implements the ServerlessServiceLister interface.
type serverlessServiceLister struct {
	indexer cache.Indexer
}

// NewServerlessServiceLister returns a new ServerlessServiceLister.
func NewServerlessServiceLister(indexer cache.Indexer) ServerlessServiceLister {
	return &serverlessServiceLister{indexer: indexer}
}

// List lists all ServerlessServices in the indexer.
func (s *serverlessServiceLister) List(selector labels.Selector) (ret []*v1alpha1.ServerlessService, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServerlessService))
	})
	return ret, err
}

// ServerlessServices returns an object that can list and get ServerlessServices.
func (s *serverlessServiceLister) ServerlessServices(namespace string) ServerlessServiceNamespaceLister {
	return serverlessServiceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServerlessServiceNamespaceLister helps list and get ServerlessServices.
type ServerlessServiceNamespaceLister interface {
	// List lists all ServerlessServices in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ServerlessService, err error)
	// Get retrieves the ServerlessService from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ServerlessService, error)
	ServerlessServiceNamespaceListerExpansion
}

// serverlessServiceNamespaceLister implements the ServerlessServiceNamespaceLister
// interface.
type serverlessServiceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServerlessServices in the indexer for a given namespace.
func (s serverlessServiceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ServerlessService, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServerlessService))
	})
	return ret, err
}

// Get retrieves the ServerlessService from the indexer for a given namespace and name.
func (s serverlessServiceNamespaceLister) Get(name string) (*v1alpha1.ServerlessService, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("serverlessservice"), name)
	}
	return obj.(*v1alpha1.ServerlessService), nil
}
