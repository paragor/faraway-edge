package utils

import (
	"context"
	"fmt"
	"log/slog"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/paragor/faraway-edge/pkg/log"
	"google.golang.org/protobuf/encoding/protojson"
)

func DumpSnapshotAsJson(ctx context.Context, snapshot *cache.Snapshot) {
	logger := log.FromContext(ctx)
	opts := protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: false,
	}

	for _, ress := range snapshot.Resources {
		if len(ress.Items) == 0 {
			continue
		}
		fmt.Println("####")
		for name, res := range ress.Items {
			jdata, err := opts.Marshal(res.Resource)
			if err != nil {
				logger.Error("error marshaling resource", slog.String("name", name), log.Error(err))
				continue
			}

			fmt.Println(string(jdata))
		}
	}
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
