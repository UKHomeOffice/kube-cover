/*

Copyright 2015 All rights reserved.
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

package kubecover

import (
	"fmt"
	"net/http/httputil"
	"net/url"

	"github.com/gambol99/kube-cover/policy"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

// NewKubeCover creates a new kube cover service
func NewKubeCover(upstream, policyPath string) (*KubeCover, error) {

	service := new(KubeCover)
	// step: parse and validate the upstreams
	location, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstrem url, %s", err)
	}
	service.upstream = location

	glog.Infof("kubernetes api: %s", service.upstream.String())

	// step: create the policy controller
	acl, err := policy.NewController(policyPath)
	if err != nil {
		return nil, err
	}
	service.acl = acl

	// step: create the gin router
	router := gin.Default()

	// step: handle operations related to replication controllers]
	{
		replicationEndpoint := "/api/v1/namespaces/:namespace/replicationcontrollers"
		router.POST(replicationEndpoint, service.handleReplicationController, service.proxyHandler)
		router.PATCH(replicationEndpoint, service.handleReplicationController, service.proxyHandler)
		router.PUT(replicationEndpoint, service.handleReplicationController, service.proxyHandler)
	}
	// step: handle the post operations
	{
		podEndpoint := "/api/v1/namespaces/:namespace/pods"
		router.POST(podEndpoint, service.handlePods, service.proxyHandler)
		router.PATCH(podEndpoint, service.handlePods, service.proxyHandler)
		router.PUT(podEndpoint, service.handlePods, service.proxyHandler)
	}
	router.Use(service.proxyHandler)

	service.engine = router

	// step: create and setup the reverse proxy
	service.proxy = httputil.NewSingleHostReverseProxy(service.upstream)
	service.proxy.Transport = buildTransport()

	return service, nil
}

// Run start the gin engine and begins serving content
func (r *KubeCover) Run(address, certFile, privateFile string) error {
	if err := r.engine.RunTLS(address, certFile, privateFile); err != nil {
		return err
	}

	return nil
}
