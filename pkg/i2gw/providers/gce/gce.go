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
	"fmt"

	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/notifications"
	"k8s.io/apimachinery/pkg/util/validation/field"
	backendconfigv1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1"
	backendconfigv1beta1 "k8s.io/ingress-gce/pkg/apis/backendconfig/v1beta1"
)

const ProviderName = "gce"

func init() {
	i2gw.ProviderConstructorByName[ProviderName] = NewProvider
}

// Provider implements the i2gw.Provider interface.
type Provider struct {
	storage                       *storage
	reader                        reader
	resourcesToIRConverter        resourcesToIRConverter
	irToGatewayResourcesConverter irToGatewayResourcesConverter
}

func NewProvider(conf *i2gw.ProviderConf) i2gw.Provider {
	// Add BackendConfig and FrontendConfig to Schema when reading in-cluster
	// so these resources can be recognized.
	if conf.Client != nil {
		if err := backendconfigv1.AddToScheme(conf.Client.Scheme()); err != nil {
			notify(notifications.ErrorNotification, "Failed to add v1 BackendConfig Scheme")
		}
		if err := backendconfigv1beta1.AddToScheme(conf.Client.Scheme()); err != nil {
			notify(notifications.ErrorNotification, "Failed to add v1beta1 BackendConfig Scheme")
		}
	}
	return &Provider{
		storage:                       newResourcesStorage(),
		reader:                        newResourceReader(conf),
		resourcesToIRConverter:        newResourcesToIRConverter(conf),
		irToGatewayResourcesConverter: newIRToGatewayResourcesConverter(),
	}
}

func (p *Provider) ReadResourcesFromCluster(ctx context.Context) error {
	storage, err := p.reader.readResourcesFromCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to read gce resources from cluster: %w", err)
	}

	p.storage = storage
	return nil
}

func (p *Provider) ReadResourcesFromFile(_ context.Context, filename string) error {
	storage, err := p.reader.readResourcesFromFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read gce resources from file: %w", err)
	}
	p.storage = storage
	return nil
}

// ToIR converts stored Ingress GCE API entities to i2gw.IR including the
// ingress-gce specific features.
func (p *Provider) ToIR() (i2gw.IR, field.ErrorList) {
	return p.resourcesToIRConverter.convertToIR(p.storage)
}

func (p *Provider) ToGatewayResources(ir i2gw.IR) (i2gw.GatewayResources, field.ErrorList) {
	return p.irToGatewayResourcesConverter.irToGateway(ir)
}
