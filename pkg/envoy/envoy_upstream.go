package envoy

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/paragor/faraway-edge/pkg/encodinghelper"
	"google.golang.org/protobuf/types/known/durationpb"
)

type EnvoyUpstream interface {
	GenerateEnvoyCluster(name string) *clusterv3.Cluster
}

type EnvoyUpstreamStaticAddresses struct {
	Port            uint32                  `json:"port"`
	StaticAddresses []string                `json:"static_addresses"`
	ConnectTimeout  encodinghelper.Duration `json:"connect_timeout"`
}

func (u *EnvoyUpstreamStaticAddresses) Validate() error {
	if u.Port == 0 {
		return fmt.Errorf("port is required and must be greater than 0")
	}
	if u.Port > 65535 {
		return fmt.Errorf("port must be less than or equal to 65535")
	}
	if len(u.StaticAddresses) == 0 {
		return fmt.Errorf("static_addresses is required and must contain at least one address")
	}
	for i, addr := range u.StaticAddresses {
		if addr == "" {
			return fmt.Errorf("static_addresses[%d] is empty", i)
		}
	}
	if u.ConnectTimeout.Duration() <= 0 {
		return fmt.Errorf("connect_timeout is required and must be greater than 0")
	}
	return nil
}

func (u *EnvoyUpstreamStaticAddresses) GenerateEnvoyCluster(name string) *clusterv3.Cluster {
	return &clusterv3.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(u.ConnectTimeout.Duration()),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_STATIC,
		},
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpointv3.LocalityLbEndpoints{
				{LbEndpoints: envoyStaticEndpoints(u.StaticAddresses, u.Port)},
			},
			NamedEndpoints: nil,
			Policy:         nil,
		},
	}
}

func envoyStaticEndpoints(addresses []string, port uint32) []*endpointv3.LbEndpoint {
	result := []*endpointv3.LbEndpoint{}
	for _, addr := range addresses {
		result = append(result, &endpointv3.LbEndpoint{
			HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
				Endpoint: &endpointv3.Endpoint{
					Address: &corev3.Address{
						Address: &corev3.Address_SocketAddress{
							SocketAddress: &corev3.SocketAddress{
								Protocol: corev3.SocketAddress_TCP,
								Address:  addr,
								PortSpecifier: &corev3.SocketAddress_PortValue{
									PortValue: port,
								},
							},
						},
					},
				},
			},
		})
	}
	return result
}
