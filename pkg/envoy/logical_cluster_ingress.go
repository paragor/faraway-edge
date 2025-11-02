package envoy

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type IngressConfig struct {
	Domain string `json:"domain"`
}

func (ic *IngressConfig) Validate() error {
	if ic.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

type LogicalClusterIngress struct {
	Name          string                        `json:"name"`
	HttpUpstream  *EnvoyUpstreamStaticAddresses `json:"http_upstream"`
	HttpsUpstream *EnvoyUpstreamStaticAddresses `json:"https_upstream"`

	Frontends []*IngressConfig `json:"frontends"`
}

func (li *LogicalClusterIngress) Validate() error {
	if li.Name == "" {
		return fmt.Errorf("ingress name is required")
	}
	if li.HttpUpstream == nil {
		return fmt.Errorf("ingress %q: http_upstream is required", li.Name)
	}
	if err := li.HttpUpstream.Validate(); err != nil {
		return fmt.Errorf("ingress %q: http_upstream: %w", li.Name, err)
	}
	if li.HttpsUpstream == nil {
		return fmt.Errorf("ingress %q: https_upstream is required", li.Name)
	}
	if err := li.HttpsUpstream.Validate(); err != nil {
		return fmt.Errorf("ingress %q: https_upstream: %w", li.Name, err)
	}
	if len(li.Frontends) == 0 {
		return fmt.Errorf("ingress %q: frontends is required and must contain at least one frontend", li.Name)
	}
	for i, frontend := range li.Frontends {
		if frontend == nil {
			return fmt.Errorf("ingress %q: frontends[%d] is nil", li.Name, i)
		}
		if err := frontend.Validate(); err != nil {
			return fmt.Errorf("ingress %q: frontends[%d]: %w", li.Name, i, err)
		}
	}
	return nil
}

func (li *LogicalClusterIngress) VirtualHost(logicalClusterName string) *routev3.VirtualHost {
	upstreamClusterName := li.getHttpClusterName(logicalClusterName)
	domains := []string{}
	for _, front := range li.Frontends {
		domains = append(domains, front.Domain)
	}
	return &routev3.VirtualHost{
		Name:    upstreamClusterName,
		Domains: domains,
		Routes: []*routev3.Route{
			{
				Name: upstreamClusterName,
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: upstreamClusterName,
						},
					},
				},
				StatPrefix: upstreamClusterName + ".",
			},
		},
	}
}

func (li *LogicalClusterIngress) TLSFilter(logicalClusterName string) *listenerv3.FilterChain {
	domains := []string{}
	for _, frontend := range li.Frontends {
		domains = append(domains, frontend.Domain)
	}
	filter := &EnvoyTLSFilter{
		Domains:             domains,
		UpstreamClusterName: li.getHttpsClusterName(logicalClusterName),
		StatPrefix:          li.getHttpsClusterName(logicalClusterName) + ".",
	}
	return filter.GenerateFilterChain()
}

func (li *LogicalClusterIngress) Clusters(logicalClusterName string) []*clusterv3.Cluster {
	return []*clusterv3.Cluster{
		li.HttpUpstream.GenerateEnvoyCluster(li.getHttpClusterName(logicalClusterName)),
		li.HttpsUpstream.GenerateEnvoyCluster(li.getHttpsClusterName(logicalClusterName)),
	}
}

func (li *LogicalClusterIngress) getHttpClusterName(logicalClusterName string) string {
	return logicalClusterName + ".http." + li.Name
}
func (li *LogicalClusterIngress) getHttpsClusterName(logicalClusterName string) string {
	return logicalClusterName + ".https." + li.Name
}
