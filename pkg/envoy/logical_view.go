package envoy

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	tls_inspectorv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	http_connection_managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/paragor/faraway-edge/pkg/utils"
	"google.golang.org/protobuf/types/known/anypb"
)

type LogicalView struct {
	LogicalClusters []*LogicalCluster `json:"logical_clusters"`
	HttpPort        uint32            `json:"http_port"`
	HttpsPort       uint32            `json:"https_port"`
}

func (v *LogicalView) Validate() error {
	if v.HttpPort == 0 {
		return fmt.Errorf("http_port is required and must be greater than 0")
	}
	if v.HttpPort > 65535 {
		return fmt.Errorf("http_port must be less than or equal to 65535")
	}
	if v.HttpsPort == 0 {
		return fmt.Errorf("https_port is required and must be greater than 0")
	}
	if v.HttpsPort > 65535 {
		return fmt.Errorf("https_port must be less than or equal to 65535")
	}
	if len(v.LogicalClusters) == 0 {
		return fmt.Errorf("logical_clusters is required and must contain at least one cluster")
	}
	for i, cluster := range v.LogicalClusters {
		if cluster == nil {
			return fmt.Errorf("logical_clusters[%d] is nil", i)
		}
		if err := cluster.Validate(); err != nil {
			return fmt.Errorf("logical_clusters[%d]: %w", i, err)
		}
	}
	return nil
}

func (s *LogicalView) Listeners() []*listenerv3.Listener {
	return []*listenerv3.Listener{
		s.generateHttpListener(),
		s.generateHttpsListener(),
	}
}

func (s *LogicalView) Clusters() []*clusterv3.Cluster {
	result := []*clusterv3.Cluster{}
	for _, cluster := range s.LogicalClusters {
		result = append(result, cluster.Clusters()...)
	}
	return result
}

func (s *LogicalView) generateHttpsListener() *listenerv3.Listener {
	filters := []*listenerv3.FilterChain{}
	for _, cluster := range s.LogicalClusters {
		filters = append(filters, cluster.TLSFilters()...)
	}

	return &listenerv3.Listener{
		Name: "https_listener",
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: s.HttpsPort,
					},
				},
			},
		},
		StatPrefix: "https",
		ListenerFilters: []*listenerv3.ListenerFilter{
			{
				Name: wellknown.TLSInspector,
				ConfigType: &listenerv3.ListenerFilter_TypedConfig{
					TypedConfig: utils.Must(anypb.New(&tls_inspectorv3.TlsInspector{})),
				},
			},
		},
		FilterChains: filters,
	}
}

func (s *LogicalView) generateHttpListener() *listenerv3.Listener {
	vhosts := []*routev3.VirtualHost{}
	for _, cluster := range s.LogicalClusters {
		vhosts = append(vhosts, cluster.VirtualHosts()...)
	}

	return &listenerv3.Listener{
		Name: "http_listener",
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: s.HttpPort,
					},
				},
			},
		},
		StatPrefix: "http",
		FilterChains: []*listenerv3.FilterChain{{
			Filters: []*listenerv3.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: utils.Must(anypb.New(
						&http_connection_managerv3.HttpConnectionManager{
							StatPrefix: "ingress_http",
							RouteSpecifier: &http_connection_managerv3.HttpConnectionManager_RouteConfig{
								RouteConfig: &routev3.RouteConfiguration{
									Name:         "local_route",
									VirtualHosts: vhosts,
								},
							},
							HttpFilters: []*http_connection_managerv3.HttpFilter{{
								Name: wellknown.Router,
								ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
									TypedConfig: utils.Must(anypb.New(&routerv3.Router{})),
								},
							}},
						})),
				},
			}},
		}},
	}
}
