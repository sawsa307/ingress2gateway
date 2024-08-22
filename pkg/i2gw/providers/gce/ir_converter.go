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

	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/common"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type contextKey int

const (
	serviceKey contextKey = iota
)

// resourcesToIRConverter implements the ToIR function of i2gw.ResourcesToIRConverter interface.
type resourcesToIRConverter struct {
	conf *i2gw.ProviderConf

	featureParsers                []i2gw.FeatureParser
	implementationSpecificOptions i2gw.ProviderImplementationSpecificOptions
	ctx                           context.Context
}

// newResourcesToIRConverter returns an ingress-gce irConverter instance.
func newResourcesToIRConverter(conf *i2gw.ProviderConf) resourcesToIRConverter {
	return resourcesToIRConverter{
		conf:           conf,
		featureParsers: []i2gw.FeatureParser{},
		implementationSpecificOptions: i2gw.ProviderImplementationSpecificOptions{
			ToImplementationSpecificHTTPPathTypeMatch: implementationSpecificHTTPPathTypeMatch,
		},
		ctx: context.Background(),
	}
}

func (c *resourcesToIRConverter) convertToIR(storage *storage) (i2gw.IR, field.ErrorList) {
	ingressList := []networkingv1.Ingress{}
	for _, ing := range storage.Ingresses {
		if ing != nil && common.GetIngressClass(*ing) == "" {
			if ing.Annotations == nil {
				ing.Annotations = make(map[string]string)
			}
			ing.Annotations[networkingv1beta1.AnnotationIngressClass] = gceIngressClass
		}
		ingressList = append(ingressList, *ing)
	}

	// Convert plain ingress resources to gateway resources, ignoring all
	// provider-specific features.
	ir, errs := common.ToIR(ingressList, c.implementationSpecificOptions)
	if len(errs) > 0 {
		return i2gw.IR{}, errs
	}

	errs = setGCEGatewayClasses(ingressList, ir.Gateways)
	if len(errs) > 0 {
		return i2gw.IR{}, errs
	}

	for _, parseFeatureFunc := range c.featureParsers {
		// Apply the feature parsing function to the gateway resources, one by one.
		parseErrs := parseFeatureFunc(ingressList, &ir)
		// Append the parsing errors to the error list.
		errs = append(errs, parseErrs...)
	}
	buildGceServiceIR(c.ctx, storage, &ir)
	return ir, errs
}
