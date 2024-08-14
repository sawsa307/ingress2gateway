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
	gkegatewayv1 "github.com/GoogleCloudPlatform/gke-gateway-api/apis/networking/v1"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1"
	"k8s.io/ingress-gce/pkg/apis/backendconfig/v1beta1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
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
	sessionAffinity *sessionAffinityConfig
}

type sessionAffinityConfig struct {
	affinityType string
	cookieTtlSec *int64
}

func toServiceIRs(storage *storage) map[types.NamespacedName]gceServiceIR {
	gceServiceIRs := make(map[types.NamespacedName]gceServiceIR)

	beConfigToSvcs, betaBEConfigToSvcs := getBEConfigMapping(storage)

	for beConfigKey, beConfig := range storage.BackendConfigs {
		serviceIR := beConfigToServiceIR(beConfig)
		serviceList := beConfigToSvcs[beConfigKey]
		for _, svcKey := range serviceList {
			gceServiceIRs[svcKey] = serviceIR
		}
	}

	for betaBeConfigKey, betaBeConfig := range storage.BetaBackendConfigs {
		serviceIR := betaBeConfigToServiceIR(betaBeConfig)
		serviceList := betaBEConfigToSvcs[betaBeConfigKey]
		for _, svcKey := range serviceList {
			gceServiceIRs[svcKey] = serviceIR
		}
	}

	return gceServiceIRs
}

func beConfigToServiceIR(beConfig *v1.BackendConfig) gceServiceIR {
	var serviceIR gceServiceIR

	serviceIR.namespace = beConfig.Namespace
	serviceIR.name = beConfig.Name
	if beConfig.Spec.SessionAffinity != nil {
		serviceIR.sessionAffinity = &sessionAffinityConfig{
			affinityType: beConfig.Spec.SessionAffinity.AffinityType,
		}
		if beConfig.Spec.SessionAffinity.AffinityType == "GENERATED_COOKIE" &&
			beConfig.Spec.SessionAffinity.AffinityCookieTtlSec != nil {
			serviceIR.sessionAffinity.cookieTtlSec = beConfig.Spec.SessionAffinity.AffinityCookieTtlSec
		}
	}
	return serviceIR
}

func betaBeConfigToServiceIR(betaBeConfig *v1beta1.BackendConfig) gceServiceIR {
	var serviceIR gceServiceIR

	serviceIR.namespace = betaBeConfig.Namespace
	serviceIR.name = betaBeConfig.Name

	if betaBeConfig.Spec.SessionAffinity != nil {
		serviceIR.sessionAffinity = &sessionAffinityConfig{
			affinityType: betaBeConfig.Spec.SessionAffinity.AffinityType,
		}
		if betaBeConfig.Spec.SessionAffinity.AffinityType == "GENERATED_COOKIE" &&
			betaBeConfig.Spec.SessionAffinity.AffinityCookieTtlSec != nil {
			serviceIR.sessionAffinity.cookieTtlSec = betaBeConfig.Spec.SessionAffinity.AffinityCookieTtlSec
		}
	}
	return serviceIR
}

func toServiceExtension(serviceIR gceServiceIR, serviceName string) i2gw.ServiceExtension {
	serviceExtension := i2gw.ServiceExtension{GCE: &i2gw.GCEServiceExtension{}}
	if serviceIR.sessionAffinity != nil {
		serviceExtension.GCE.GCPBackendPolicy = toBackendPolicy(serviceIR, serviceName)
	}

	return serviceExtension
}

func toBackendPolicy(serviceIR gceServiceIR, serviceName string) *gkegatewayv1.GCPBackendPolicy {
	if serviceIR.sessionAffinity != nil {
		affinityType := serviceIR.sessionAffinity.affinityType
		cookieTTLSec := serviceIR.sessionAffinity.cookieTtlSec
		backendPolicy := gkegatewayv1.GCPBackendPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: serviceIR.namespace,
				Name:      serviceIR.name,
			},
			Spec: gkegatewayv1.GCPBackendPolicySpec{
				Default: &gkegatewayv1.GCPBackendPolicyConfig{
					SessionAffinity: &gkegatewayv1.SessionAffinityConfig{
						Type:         &affinityType,
						CookieTTLSec: cookieTTLSec,
					},
				},
				TargetRef: v1alpha2.PolicyTargetReference{
					Group: "",
					Kind:  "Service",
					Name:  gatewayv1.ObjectName(serviceName),
				},
			},
		}
		backendPolicy.SetGroupVersionKind(GCPBackendPolicyGVK)
		return &backendPolicy
	}
	return nil
}
