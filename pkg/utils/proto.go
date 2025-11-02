package utils

import (
	"fmt"
	"io"
	"sort"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/protojson"
)

func DumpSnapshotAsJson(snapshot cache.ResourceSnapshot, writer io.Writer) error {
	opts := protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: false,
	}

	resourcesTypes := []string{
		resource.EndpointType,
		resource.ClusterType,
		resource.RouteType,
		resource.ScopedRouteType,
		resource.VirtualHostType,
		resource.ListenerType,
		resource.SecretType,
		resource.RuntimeType,
		resource.ExtensionConfigType,
		resource.RateLimitConfigType,
	}
	sort.Strings(resourcesTypes)

	for _, key := range resourcesTypes {
		ress := snapshot.GetResources(key)
		if len(ress) == 0 {
			continue
		}
		for name, res := range ress {
			jdata, err := opts.Marshal(res)
			if err != nil {
				return fmt.Errorf("cant marshal %s: %w", name, err)
			}

			if _, err := fmt.Fprintln(writer, string(jdata)); err != nil {
				return err
			}
		}
	}
	return nil
}

//func HashSnapshot(resources map[resource.Type][]types.Resource) string {
//	mOpts := proto.MarshalOptions{Deterministic: true}
//	mOpts.Marshal()
//	cache.NewSnapshot("", res)
//	for i := 0; i < int(types.UnknownType); i++ {
//		for _, res := range resources[cache.GetResponseTypeURL(types.ResponseType(i))] {
//		}
//
//	}
//}

func CastListeners(rr []*listenerv3.Listener) []types.Resource {
	res := make([]types.Resource, 0, len(rr))
	for _, r := range rr {
		res = append(res, r)
	}
	return res
}

func CastClusters(rr []*clusterv3.Cluster) []types.Resource {
	res := make([]types.Resource, 0, len(rr))
	for _, r := range rr {
		res = append(res, r)
	}
	return res
}
