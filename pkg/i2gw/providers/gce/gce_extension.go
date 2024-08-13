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
	"encoding/json"

	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/notifications"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/ingress-gce/pkg/annotations"
)

type serviceList []types.NamespacedName

func getBEConfigMapping(storage *storage) (beConfigToSvcs map[types.NamespacedName]serviceList, betaBEConfigToSvcs map[types.NamespacedName]serviceList) {
	beConfigToSvcs = make(map[types.NamespacedName]serviceList)
	betaBEConfigToSvcs = make(map[types.NamespacedName]serviceList)

	for _, service := range storage.Services {
		serviceName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}

		// Read BackendConfig based on v1 BackendConfigKey.
		name, exists := getBackendConfigName(service, annotations.BackendConfigKey)
		if exists {
			beConfigName := types.NamespacedName{Namespace: service.Namespace, Name: name}
			beConfigToSvcs[beConfigName] = append(beConfigToSvcs[beConfigName], serviceName)
			continue
		}

		// Read BackendConfig based on v1beta1 BackendConfigKey.
		name, exists = getBackendConfigName(service, annotations.BetaBackendConfigKey)
		if exists {
			beConfigName := types.NamespacedName{Namespace: service.Namespace, Name: name}
			betaBEConfigToSvcs[beConfigName] = append(betaBEConfigToSvcs[beConfigName], beConfigName)
			continue
		}
	}
	return beConfigToSvcs, betaBEConfigToSvcs
}

// Get names of the BackendConfig in the cluster based on the BackendConfig
// annotation on k8s Services.
func getBackendConfigName(service *apiv1.Service, backendConfigKey string) (string, bool) {
	val, exists := getBackendConfigAnnotation(service, backendConfigKey)
	if !exists {
		return "", false
	}

	return parseBackendConfigName(val)
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
func parseBackendConfigName(val string) (string, bool) {
	configs := annotations.BackendConfigs{}
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		notify(notifications.ErrorNotification, "BackendConfig annotation is invalid json")
		return "", false
	}

	if configs.Default == "" && len(configs.Ports) == 0 {
		notify(notifications.ErrorNotification, "No BackendConfig's found in annotation")
		return "", false
	}

	if len(configs.Ports) != 0 {
		notify(notifications.ErrorNotification, "Only config with default is supported since HealthCheckPolicy is attached on the whole service")
		return "", false
	}
	return configs.Default, true
}
