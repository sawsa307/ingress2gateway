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
	"testing"

	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/ingress-gce/pkg/annotations"
	backendconfigv1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1"
	backendconfigv1beta1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1beta1"
)

func TestGetBackendConfigMapping(t *testing.T) {
	t.Parallel()
	testNamespace := "test-namespace"

	testServiceName1 := "test-service-1"
	testBeConfigName := "backendconfig-1"

	testServiceName2 := "test-service-2"
	testBetaBeConfigName := "betabackendconfig-1"

	services := map[types.NamespacedName]*apiv1.Service{
		{Namespace: testNamespace, Name: testServiceName1}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceName1,
				Namespace: testNamespace,
				Annotations: map[string]string{
					annotations.BackendConfigKey: `{"default":"backendconfig-1"}`,
				},
			},
		},
		{Namespace: testNamespace, Name: testServiceName2}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceName2,
				Namespace: testNamespace,
				Annotations: map[string]string{
					annotations.BetaBackendConfigKey: `{"default":"betabackendconfig-1"}`,
				},
			},
		},
	}
	backendConfigs := map[types.NamespacedName]*backendconfigv1.BackendConfig{
		{Namespace: testNamespace, Name: testBeConfigName}: {},
	}
	betaBackendConfigs := map[types.NamespacedName]*backendconfigv1beta1.BackendConfig{
		{Namespace: testNamespace, Name: testBetaBeConfigName}: {},
	}

	provider := NewProvider(&i2gw.ProviderConf{})
	gceProvider := provider.(*Provider)
	gceProvider.storage = newResourcesStorage()
	gceProvider.storage.Services = services
	gceProvider.storage.BackendConfigs = backendConfigs
	gceProvider.storage.BetaBackendConfigs = betaBackendConfigs

	beConfigToSvcs, betaBeConfigToSvcs := getBackendConfigMapping(context.TODO(), gceProvider.storage)

	beConfigSvcList := beConfigToSvcs[types.NamespacedName{Namespace: testNamespace, Name: testBeConfigName}]
	if len(beConfigSvcList) != 1 ||
		beConfigSvcList[0].Namespace != testNamespace ||
		beConfigSvcList[0].Name != testServiceName1 {
		t.Errorf("Got BackendConfig %s mapped to %s, expected [%s/%s]", testBeConfigName, beConfigSvcList, testNamespace, testServiceName1)
	}

	betaBeConfigSvcList := betaBeConfigToSvcs[types.NamespacedName{Namespace: testNamespace, Name: testBetaBeConfigName}]
	if len(betaBeConfigSvcList) != 1 ||
		betaBeConfigSvcList[0].Namespace != testNamespace ||
		betaBeConfigSvcList[0].Name != testServiceName2 {
		t.Errorf("Got BataBackendConfig %s mapped to %s, expected [%s/%s]", testBetaBeConfigName, betaBeConfigSvcList, testNamespace, testServiceName2)
	}
}

func TestGetBackendConfigName(t *testing.T) {
	t.Parallel()

	testNamespace := "test-namespace"
	testServiceName := "test-service"
	testBeConfigName := "backendconfig-1"

	testCases := []struct {
		desc           string
		service        *apiv1.Service
		beConfigKey    string
		expectedName   string
		expectedExists bool
	}{
		{
			desc: "Service with v1 BackendConfig, using default Config over all ports",
			service: &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						annotations.BackendConfigKey: `{"default":"backendconfig-1"}`,
					},
				},
			},
			beConfigKey:    annotations.BackendConfigKey,
			expectedName:   testBeConfigName,
			expectedExists: true,
		},
		{
			desc: "Service with v1beta1 BackendConfig, using default Config over all ports",
			service: &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						annotations.BetaBackendConfigKey: `{"default":"backendconfig-1"}`,
					},
				},
			},
			beConfigKey:    annotations.BetaBackendConfigKey,
			expectedName:   testBeConfigName,
			expectedExists: true,
		},
		{
			desc: "Service with v1 BackendConfig, using Port Config, not supported",
			service: &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						annotations.BackendConfigKey: `{"port1": "beconfig1", "port2": "beconfig2"}`,
					},
				},
			},
			beConfigKey:    annotations.BackendConfigKey,
			expectedName:   "",
			expectedExists: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			ctx = context.WithValue(ctx, serviceKey, tc.service)
			gotName, gotExists := getBackendConfigName(ctx, tc.service, tc.beConfigKey)
			if gotExists != tc.expectedExists {
				t.Errorf("getBackendConfigName() got exist = %v, expected %v", gotExists, tc.expectedExists)
			}
			if gotName != tc.expectedName {
				t.Errorf("getBackendConfigName() got exist = %v, expected %v", gotName, tc.expectedName)
			}
		})
	}
}
