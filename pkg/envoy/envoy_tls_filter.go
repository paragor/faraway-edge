package envoy

import (
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcp_proxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/paragor/faraway-edge/pkg/utils"
	"google.golang.org/protobuf/types/known/anypb"
)

type EnvoyTLSFilter struct {
	Domains             []string
	UpstreamClusterName string
	StatPrefix          string
}

func (f *EnvoyTLSFilter) GenerateFilterChain() *listenerv3.FilterChain {
	return &listenerv3.FilterChain{
		FilterChainMatch: &listenerv3.FilterChainMatch{
			TransportProtocol: "tls",
			ServerNames:       f.Domains,
		},
		Filters: []*listenerv3.Filter{
			{
				Name: wellknown.TCPProxy,
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: utils.Must(anypb.New(&tcp_proxyv3.TcpProxy{
						StatPrefix: f.StatPrefix,
						ClusterSpecifier: &tcp_proxyv3.TcpProxy_Cluster{
							Cluster: f.UpstreamClusterName,
						},
					})),
				},
			},
		},
	}
}
