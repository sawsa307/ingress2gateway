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
)

// reader implements the i2gw.CustomResourceReader interface.
type reader struct {
	conf *i2gw.ProviderConf
}

// newResourceReader returns a resourceReader instance.
func newResourceReader(conf *i2gw.ProviderConf) reader {
	return reader{
		conf: conf,
	}
}

func (r *reader) readResourcesFromCluster(ctx context.Context) (*storage, error) {
	// read example-gateway related resources from the cluster.
	return nil, nil
}

func (r *reader) readResourcesFromFile(ctx context.Context, filename string) (*storage, error) {
	// read example-gateway related resources from the file.
	return nil, nil
}