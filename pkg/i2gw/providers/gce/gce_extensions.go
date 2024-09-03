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
	"context"
	"encoding/json"

	gkegatewayv1 "github.com/GoogleCloudPlatform/gke-gateway-api/apis/networking/v1"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/ir"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/notifications"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/ingress-gce/pkg/annotations"
	backendconfigv1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1"
	backendconfigv1beta1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1beta1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type serviceList []types.NamespacedName

func getBackendConfigMapping(ctx context.Context, storage *storage) (beConfigToSvcs map[types.NamespacedName]serviceList, betaBEConfigToSvcs map[types.NamespacedName]serviceList) {
	beConfigToSvcs = make(map[types.NamespacedName]serviceList)
	betaBEConfigToSvcs = make(map[types.NamespacedName]serviceList)

	for _, service := range storage.Services {
		svc := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
		ctx = context.WithValue(ctx, serviceKey, service)

		// Read BackendConfig based on v1 BackendConfigKey.
		beConfigName, exists := getBackendConfigName(ctx, service, annotations.BackendConfigKey)
		if exists {
			beConfigKey := types.NamespacedName{Namespace: service.Namespace, Name: beConfigName}
			beConfigToSvcs[beConfigKey] = append(beConfigToSvcs[beConfigKey], svc)
			continue
		}

		// Read BackendConfig based on v1beta1 BackendConfigKey.
		beConfigName, exists = getBackendConfigName(ctx, service, annotations.BetaBackendConfigKey)
		if exists {
			beConfigKey := types.NamespacedName{Namespace: service.Namespace, Name: beConfigName}
			betaBEConfigToSvcs[beConfigKey] = append(betaBEConfigToSvcs[beConfigKey], svc)
			continue
		}
	}
	return beConfigToSvcs, betaBEConfigToSvcs
}

// Get names of the BackendConfig in the cluster based on the BackendConfig
// annotation on k8s Services.
func getBackendConfigName(ctx context.Context, service *apiv1.Service, backendConfigKey string) (string, bool) {
	val, exists := getBackendConfigAnnotation(service, backendConfigKey)
	if !exists {
		return "", false
	}

	return parseBackendConfigName(ctx, val)
}

// Get the backend config annotation from the K8s service if it exists.
func getBackendConfigAnnotation(service *apiv1.Service, backendConfigKey string) (string, bool) {
	val, ok := service.Annotations[backendConfigKey]
	if ok {
		return val, ok
	}
	return "", false
}

// Parse the name of the BackendConfig based on the annotation only if the same
// BackendConfig is used for all Service ports.
func parseBackendConfigName(ctx context.Context, val string) (string, bool) {
	service := ctx.Value(serviceKey).(*apiv1.Service)

	configs := annotations.BackendConfigs{}
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		notify(notifications.ErrorNotification, "BackendConfig annotation is invalid json", service)
		return "", false
	}

	if configs.Default == "" && len(configs.Ports) == 0 {
		notify(notifications.ErrorNotification, "No BackendConfig's found in annotation", service)
		return "", false
	}

	if len(configs.Ports) != 0 {
		notify(notifications.ErrorNotification, "Only config with default is supported since HealthCheckPolicy is attached on the whole service", service)
		return "", false
	}
	return configs.Default, true
}

func buildGceServiceIR(ctx context.Context, storage *storage, ir *i2gw.IR) {
	if ir.Services == nil {
		ir.Services = make(map[types.NamespacedName]*i2gw.ServiceIR)
	}

	beConfigToSvcs, betaBEConfigToSvcs := getBackendConfigMapping(ctx, storage)

	for beConfigKey, beConfig := range storage.BackendConfigs {
		gceServiceIR := beConfigToGceServiceIR(beConfig)
		if gceServiceIR == nil {
			continue
		}

		serviceList := beConfigToSvcs[beConfigKey]
		for _, svcKey := range serviceList {
			if ir.Services[svcKey] == nil {
				ir.Services[svcKey] = &i2gw.ServiceIR{}
			}
			ir.Services[svcKey].Gce = gceServiceIR
		}
	}

	for betaBeConfigKey, betaBeConfig := range storage.BetaBackendConfigs {
		gceServiceIR := betaBeConfigToGceServiceIR(betaBeConfig)
		if gceServiceIR == nil {
			continue
		}
		serviceList := betaBEConfigToSvcs[betaBeConfigKey]
		for _, svcKey := range serviceList {
			if ir.Services[svcKey] == nil {
				ir.Services[svcKey] = &i2gw.ServiceIR{}
			}
			ir.Services[svcKey].Gce = gceServiceIR
		}
	}
}

func beConfigToGceServiceIR(beConfig *backendconfigv1.BackendConfig) *ir.GceServiceIR {
	if beConfig == nil {
		return nil
	}
	var gceServiceIR ir.GceServiceIR
	if beConfig.Spec.SessionAffinity != nil {
		saConfig := ir.SessionAffinityConfig{
			AffinityType: beConfig.Spec.SessionAffinity.AffinityType,
			CookieTTLSec: beConfig.Spec.SessionAffinity.AffinityCookieTtlSec,
		}
		gceServiceIR.SessionAffinity = &saConfig
	}

	return &gceServiceIR
}

func betaBeConfigToGceServiceIR(betaBeConfig *backendconfigv1beta1.BackendConfig) *ir.GceServiceIR {
	if betaBeConfig == nil {
		return nil
	}
	var gceServiceIR ir.GceServiceIR
	if betaBeConfig.Spec.SessionAffinity != nil {
		saConfig := ir.SessionAffinityConfig{
			AffinityType: betaBeConfig.Spec.SessionAffinity.AffinityType,
			CookieTTLSec: betaBeConfig.Spec.SessionAffinity.AffinityCookieTtlSec,
		}
		gceServiceIR.SessionAffinity = &saConfig
	}

	return &gceServiceIR
}

func buildGceServiceExtensions(ir i2gw.IR, gatewayResources *i2gw.GatewayResources) {
	for svcKey, serviceIR := range ir.Services {
		if serviceIR == nil {
			continue
		}
		bePolicy := toGCPBackendPolicy(svcKey, serviceIR)
		if bePolicy == nil {
			continue
		}
		obj, err := i2gw.CastToUnstructured(bePolicy)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast GCPBackendPolicy to unstructured", bePolicy)
			continue
		}
		gatewayResources.GatewayExtensions = append(gatewayResources.GatewayExtensions, *obj)
	}
}

func toGCPBackendPolicy(serviceNamespacedName types.NamespacedName, serviceIR *i2gw.ServiceIR) *gkegatewayv1.GCPBackendPolicy {
	if serviceIR.Gce == nil || serviceIR.Gce.SessionAffinity == nil {
		return nil
	}
	affinityType := serviceIR.Gce.SessionAffinity.AffinityType
	backendPolicy := gkegatewayv1.GCPBackendPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: serviceNamespacedName.Namespace,
			Name:      serviceNamespacedName.Name + "-GCPBackendPolicy",
		},
		Spec: gkegatewayv1.GCPBackendPolicySpec{
			Default: &gkegatewayv1.GCPBackendPolicyConfig{
				SessionAffinity: &gkegatewayv1.SessionAffinityConfig{
					Type: &affinityType,
				},
			},
			TargetRef: v1alpha2.NamespacedPolicyTargetReference{
				Group: "",
				Kind:  "Service",
				Name:  gatewayv1.ObjectName(serviceNamespacedName.Name),
			},
		},
	}
	if affinityType == "GENERATED_COOKIE" {
		backendPolicy.Spec.Default.SessionAffinity.CookieTTLSec = serviceIR.Gce.SessionAffinity.CookieTTLSec
	}

	backendPolicy.SetGroupVersionKind(GCPBackendPolicyGVK)
	return &backendPolicy
}
