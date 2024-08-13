/*
Copyright 2024 The Kubernetes Authors.

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

package gce

import (
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1"
	"k8s.io/ingress-gce/pkg/apis/backendconfig/v1beta1"
)

type gceIR struct {
	gceGatewayIRs   map[types.NamespacedName]gceGatewayIR
	gceHTTPRouteIRs map[types.NamespacedName]gceHTTPRouteIR
	gceServiceIRs   map[types.NamespacedName]gceServiceIR
}

// gceGatewayIR holds the abstraction of GCE gateway extension.
type gceGatewayIR struct{}

type gceHTTPRouteIR struct{}

type gceServiceIR struct {
	namespace       string
	name            string
	sessionAffinity sessionAffinityConfig
}

type sessionAffinityConfig struct {
	affinityType string
	cookieTtlSec int64
}

func toServiceIR(storage *storage) map[types.NamespacedName]gceServiceIR {
	gceServiceIR := make(map[types.NamespacedName]gceServiceIR)

	beConfigToSvcs, betaBEConfigToSvcs := getBEConfigMapping(storage)

	for beConfigKey, beConfig := range storage.BackendConfigs {
		serviceIR := beConfigToServiceIR(beConfig)
		serviceList := beConfigToSvcs[beConfigKey]
		for _, svcKey := range serviceList {
			gceServiceIR[svcKey] = serviceIR
		}
	}

	for betaBeConfigKey, betaBeConfig := range storage.BetaBackendConfigs {
		serviceIR := betaBeConfigToServiceIR(betaBeConfig)
		serviceList := betaBEConfigToSvcs[betaBeConfigKey]
		for _, svcKey := range serviceList {
			gceServiceIR[svcKey] = serviceIR
		}
	}

	return gceServiceIR
}

func beConfigToServiceIR(beConfig *v1.BackendConfig) gceServiceIR {
	var serviceIR gceServiceIR

	serviceIR.namespace = beConfig.Namespace
	serviceIR.name = beConfig.Name
	serviceIR.sessionAffinity.affinityType = beConfig.Spec.SessionAffinity.AffinityType
	serviceIR.sessionAffinity.cookieTtlSec = *beConfig.Spec.SessionAffinity.AffinityCookieTtlSec

	return serviceIR
}

func betaBeConfigToServiceIR(betaBeConfig *v1beta1.BackendConfig) gceServiceIR {
	var serviceIR gceServiceIR

	serviceIR.namespace = betaBeConfig.Namespace
	serviceIR.name = betaBeConfig.Name
	serviceIR.sessionAffinity.affinityType = betaBeConfig.Spec.SessionAffinity.AffinityType
	serviceIR.sessionAffinity.cookieTtlSec = *betaBeConfig.Spec.SessionAffinity.AffinityCookieTtlSec

	return serviceIR
}
