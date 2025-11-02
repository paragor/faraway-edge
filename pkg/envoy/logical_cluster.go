package envoy

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type LogicalCluster struct {
	Name      string                   `json:"name"`
	Ingresses []*LogicalClusterIngress `json:"ingresses"`
}

func (c *LogicalCluster) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	for i, ingress := range c.Ingresses {
		if ingress == nil {
			return fmt.Errorf("cluster %q: ingresses[%d] is nil", c.Name, i)
		}
		if err := ingress.Validate(); err != nil {
			return fmt.Errorf("cluster %q: ingresses[%d]: %w", c.Name, i, err)
		}
	}
	return nil
}

func (c *LogicalCluster) TLSFilters() []*listenerv3.FilterChain {
	result := []*listenerv3.FilterChain{}
	for _, upstream := range c.Ingresses {
		result = append(result, upstream.TLSFilter(c.Name))
	}
	return result
}

func (c *LogicalCluster) VirtualHosts() []*routev3.VirtualHost {
	result := []*routev3.VirtualHost{}
	for _, upstream := range c.Ingresses {
		result = append(result, upstream.VirtualHost(c.Name))
	}
	return result
}

func (c *LogicalCluster) Clusters() []*clusterv3.Cluster {
	result := []*clusterv3.Cluster{}
	for _, upstream := range c.Ingresses {
		result = append(result, upstream.Clusters(c.Name)...)
	}
	return result
}
