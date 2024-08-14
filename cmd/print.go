/*
Copyright 2023 The Kubernetes Authors.

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

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd"

	// Call init function for the providers
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/apisix"
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/gce"
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/ingressnginx"
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/istio"
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/kong"
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/providers/openapi3"

	// Call init for notifications
	_ "github.com/kubernetes-sigs/ingress2gateway/pkg/i2gw/notifications"
)

type PrintRunner struct {
	// outputFormat contains currently set output format. Value assigned via --output/-o flag.
	// Defaults to YAML.
	outputFormat string

	// The path to the input yaml config file. Value assigned via --input-file flag
	inputFile string

	// The namespace used to query Gateway API objects. Value assigned via
	// --namespace/-n flag.
	// On absence, the current user active namespace is used.
	namespace string

	// allNamespaces indicates whether all namespaces should be used. Value assigned via
	// --all-namespaces/-A flag.
	allNamespaces bool

	// resourcePrinter determines how resource objects are printed out
	resourcePrinter printers.ResourcePrinter

	// Only resources that matches this filter will be processed.
	namespaceFilter string

	// providers indicates which providers are used to execute convert action.
	providers []string

	// Provider specific flags --<provider>-<flag>.
	providerSpecificFlags map[string]*string
}

// PrintGatewayAPIObjects performs necessary steps to digest and print
// converted Gateway API objects. The steps include reading from the source,
// construct ingresses and provider-specific resources, convert them, then print
// the Gateway API objects out.
func (pr *PrintRunner) PrintGatewayAPIObjects(cmd *cobra.Command, _ []string) error {
	err := pr.initializeResourcePrinter()
	if err != nil {
		return fmt.Errorf("failed to initialize resrouce printer: %w", err)
	}
	err = pr.initializeNamespaceFilter()
	if err != nil {
		return fmt.Errorf("failed to initialize namespace filter: %w", err)
	}

	gatewayResources, notificationTablesMap, err := i2gw.ToGatewayAPIResources(cmd.Context(), pr.namespaceFilter, pr.inputFile, pr.providers, pr.getProviderSpecificFlags())
	if err != nil {
		return err
	}

	for _, table := range notificationTablesMap {
		fmt.Println(table)
	}

	pr.outputResult(gatewayResources)

	return nil
}

func (pr *PrintRunner) outputResult(gatewayResources []i2gw.GatewayResources) {
	resourceCount := 0

	for _, r := range gatewayResources {
		resourceCount += len(r.GatewayClasses)
		for _, gatewayClass := range r.GatewayClasses {
			gatewayClass := gatewayClass
			err := pr.resourcePrinter.PrintObj(&gatewayClass, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s GatewayClass: %v\n", gatewayClass.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.Gateways)
		for _, gatewayContext := range r.Gateways {
			gatewayContext := gatewayContext
			err := pr.resourcePrinter.PrintObj(&gatewayContext.Gateway, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s Gateway: %v\n", gatewayContext.Gateway.Name, err)
			}
			if gatewayContext.Extension == nil {
				continue
			}
			if gatewayContext.Extension.GCE != nil && gatewayContext.Extension.GCE.GCPGatewayPolicy != nil {
				resourceCount += 1
				err := pr.resourcePrinter.PrintObj(gatewayContext.Extension.GCE.GCPGatewayPolicy, os.Stdout)
				if err != nil {
					fmt.Printf("# Error printing %s GCPGatewayPolicy: %v\n", gatewayContext.Extension.GCE.GCPGatewayPolicy.Name, err)
				}
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.HTTPRoutes)
		for _, routeContext := range r.HTTPRoutes {
			routeContext := routeContext
			err := pr.resourcePrinter.PrintObj(&routeContext.HTTPRoute, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s HTTPRoute: %v\n", routeContext.HTTPRoute.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.TLSRoutes)
		for _, tlsRoute := range r.TLSRoutes {
			tlsRoute := tlsRoute
			err := pr.resourcePrinter.PrintObj(&tlsRoute, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s TLSRoute: %v\n", tlsRoute.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.TCPRoutes)
		for _, tcpRoute := range r.TCPRoutes {
			tcpRoute := tcpRoute
			err := pr.resourcePrinter.PrintObj(&tcpRoute, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s TCPRoute: %v\n", tcpRoute.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.UDPRoutes)
		for _, udpRoute := range r.UDPRoutes {
			udpRoute := udpRoute
			err := pr.resourcePrinter.PrintObj(&udpRoute, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s UDPRoute: %v\n", udpRoute.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		resourceCount += len(r.ReferenceGrants)
		for _, referenceGrant := range r.ReferenceGrants {
			referenceGrant := referenceGrant
			err := pr.resourcePrinter.PrintObj(&referenceGrant, os.Stdout)
			if err != nil {
				fmt.Printf("# Error printing %s ReferenceGrant: %v\n", referenceGrant.Name, err)
			}
		}
	}

	for _, r := range gatewayResources {
		for _, serviceExtension := range r.ServiceExtension {
			serviceExtension := serviceExtension
			if serviceExtension.GCE == nil {
				continue
			}
			if serviceExtension.GCE.GCPBackendPolicy != nil {
				resourceCount += 1
				err := pr.resourcePrinter.PrintObj(serviceExtension.GCE.GCPBackendPolicy, os.Stdout)
				if err != nil {
					fmt.Printf("# Error printing %s GCPBackendPolicy: %v\n", serviceExtension.GCE.GCPBackendPolicy.Name, err)
				}
			}
			if serviceExtension.GCE.HealthCheckPolicy != nil {
				resourceCount += 1
				err := pr.resourcePrinter.PrintObj(serviceExtension.GCE.HealthCheckPolicy, os.Stdout)
				if err != nil {
					fmt.Printf("# Error printing %s HealthCheckPolicy: %v\n", serviceExtension.GCE.HealthCheckPolicy.Name, err)
				}
			}
		}
	}

	if resourceCount == 0 {
		msg := "No resources found"
		if pr.namespaceFilter != "" {
			msg = fmt.Sprintf("%s in %s namespace", msg, pr.namespaceFilter)
		}
		fmt.Println(msg)
	}
}

// initializeResourcePrinter assign a specific type of printers.ResourcePrinter
// based on the outputFormat of the printRunner struct.
func (pr *PrintRunner) initializeResourcePrinter() error {
	switch pr.outputFormat {
	case "yaml", "":
		pr.resourcePrinter = &printers.YAMLPrinter{}
		return nil
	case "json":
		pr.resourcePrinter = &printers.JSONPrinter{}
		return nil
	default:
		return fmt.Errorf("%s is not a supported output format", pr.outputFormat)
	}

}

// initializeNamespaceFilter initializes the correct namespace filter for resource processing with these scenarios:
// 1. If the --all-namespaces flag is used, it processes all resources, regardless of whether they are from the cluster or file.
// 2. If namespace is specified, it filters resources based on that namespace.
// 3. If no namespace is specified and reading from the cluster, it attempts to get the namespace from the cluster; if unsuccessful, initialization fails.
// 4. If no namespace is specified and reading from a file, it attempts to get the namespace from the cluster; if unsuccessful, it reads all resources.
func (pr *PrintRunner) initializeNamespaceFilter() error {
	// When we should use all namespaces, empty string is used as the filter.
	if pr.allNamespaces {
		pr.namespaceFilter = ""
		return nil
	}

	// If namespace flag is not specified, try to use the default namespace from the cluster
	if pr.namespace == "" {
		ns, err := getNamespaceInCurrentContext()
		if err != nil && pr.inputFile == "" {
			// When asked to read from the cluster, but getting the current namespace
			// failed for whatever reason - do not process the request.
			return err
		}
		// If err is nil we got the right filtered namespace.
		// If the input file is specified, and we failed to get the namespace, use all namespaces.
		pr.namespaceFilter = ns
		return nil
	}

	pr.namespaceFilter = pr.namespace
	return nil
}

func newPrintCommand() *cobra.Command {
	pr := &PrintRunner{}
	var printFlags genericclioptions.JSONYamlPrintFlags
	allowedFormats := printFlags.AllowedFormats()

	// printCmd represents the print command. It prints HTTPRoutes and Gateways
	// generated from Ingress resources.
	var cmd = &cobra.Command{
		Use:   "print",
		Short: "Prints Gateway API objects generated from ingress and provider-specific resources.",
		RunE:  pr.PrintGatewayAPIObjects,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			openAPIExist := slices.Contains(pr.providers, "openapi3")
			if openAPIExist && len(pr.providers) != 1 {
				return fmt.Errorf("openapi3 must be the only provider when specified")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&pr.outputFormat, "output", "o", "yaml",
		fmt.Sprintf(`Output format. One of: (%s).`, strings.Join(allowedFormats, ", ")))

	cmd.Flags().StringVar(&pr.inputFile, "input-file", "",
		`Path to the manifest file. When set, the tool will read ingresses from the file instead of reading from the cluster. Supported files are yaml and json.`)

	cmd.Flags().StringVarP(&pr.namespace, "namespace", "n", "",
		`If present, the namespace scope for this CLI request.`)

	cmd.Flags().BoolVarP(&pr.allNamespaces, "all-namespaces", "A", false,
		`If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even
if specified with --namespace.`)

	cmd.Flags().StringSliceVar(&pr.providers, "providers", []string{},
		fmt.Sprintf("If present, the tool will try to convert only resources related to the specified providers, supported values are %v.", i2gw.GetSupportedProviders()))

	pr.providerSpecificFlags = make(map[string]*string)
	for provider, flags := range i2gw.GetProviderSpecificFlagDefinitions() {
		for _, flag := range flags {
			flagName := fmt.Sprintf("%s-%s", provider, flag.Name)
			pr.providerSpecificFlags[flagName] = cmd.Flags().String(flagName, flag.DefaultValue, fmt.Sprintf("Provider-specific: %s. %s", provider, flag.Description))
		}
	}

	_ = cmd.MarkFlagRequired("providers")
	cmd.MarkFlagsMutuallyExclusive("namespace", "all-namespaces")
	return cmd
}

// getNamespaceInCurrentContext returns the namespace in the current active context of the user.
func getNamespaceInCurrentContext() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	currentNamespace, _, err := kubeConfig.Namespace()

	return currentNamespace, err
}

// getProviderSpecificFlags returns the provider specific flags input by the user.
// The flags are returned in a map where the key is the provider name and the value is a map of flag name to flag value.
func (pr *PrintRunner) getProviderSpecificFlags() map[string]map[string]string {
	providerSpecificFlags := make(map[string]map[string]string)
	for flagName, value := range pr.providerSpecificFlags {
		provider, found := lo.Find(pr.providers, func(p string) bool { return strings.HasPrefix(flagName, fmt.Sprintf("%s-", p)) })
		if !found {
			continue
		}
		flagNameWithoutProvider := strings.TrimPrefix(flagName, fmt.Sprintf("%s-", provider))
		if providerSpecificFlags[provider] == nil {
			providerSpecificFlags[provider] = make(map[string]string)
		}
		providerSpecificFlags[provider][flagNameWithoutProvider] = *value
	}
	return providerSpecificFlags
}
