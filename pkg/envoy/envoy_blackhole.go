package envoy

import (
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcp_proxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/paragor/faraway-edge/pkg/utils"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

var envoyBlackhole = &EnvoyBlackhole{}

type EnvoyBlackhole struct{}

func (f *EnvoyBlackhole) GenerateFilterChain() *listenerv3.FilterChain {
	return &listenerv3.FilterChain{
		Filters: []*listenerv3.Filter{
			{
				Name: wellknown.TCPProxy,
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: utils.Must(anypb.New(&tcp_proxyv3.TcpProxy{
						StatPrefix: "blackhole.",
						ClusterSpecifier: &tcp_proxyv3.TcpProxy_Cluster{
							Cluster: "blackhole",
						},
					})),
				},
			},
		},
	}
}
func (f *EnvoyBlackhole) GenerateCluster() *clusterv3.Cluster {
	return &clusterv3.Cluster{
		Name:           "blackhole",
		ConnectTimeout: durationpb.New(time.Second),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_STATIC,
		},
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
			ClusterName: "blackhole",
			Endpoints: []*endpointv3.LocalityLbEndpoints{
				{LbEndpoints: []*endpointv3.LbEndpoint{}},
			},
		},
	}
}
