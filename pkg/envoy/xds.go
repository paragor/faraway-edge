package envoy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sort"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/sotw/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/paragor/faraway-edge/pkg/log"
	"github.com/paragor/faraway-edge/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type XDS struct {
	cacheManager cache.SnapshotCache
	providers    []LogicalClusterProvider
	server       server.Server
	port         int
	lastHash     string
	token        string
}

func NewXDS(xdsPort int, providers []LogicalClusterProvider, token string) *XDS {
	return &XDS{
		cacheManager: cache.NewSnapshotCache(true, AllCache{}, nil),
		port:         xdsPort,
		providers:    providers,
		token:        token,
	}
}

func (xds *XDS) takeView(ctx context.Context) (*LogicalView, error) {
	view := &LogicalView{
		HttpPort:  80,
		HttpsPort: 443,
	}
	for _, provider := range xds.providers {
		cluster, err := provider.GetLogicaCluster(ctx)
		if err != nil {
			return nil, err
		}
		view.LogicalClusters = append(view.LogicalClusters, cluster)
	}
	if err := view.Validate(); err != nil {
		return nil, fmt.Errorf("logical view validation failed: %w", err)
	}
	return view, nil
}

func (xds *XDS) initProviders(ctx context.Context) error {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		view, err := xds.takeView(ctx)
		if err != nil {
			logger.Error("Error taking view", log.Error(err))
			time.Sleep(1 * time.Second)
			continue
		}
		if err := xds.updateView(ctx, view); err != nil {
			return err
		}
		return nil
	}
}

func (xds *XDS) DumpCurrentSnapshot(writer io.Writer) error {
	snap, err := xds.cacheManager.GetSnapshot("all")
	if err != nil {
		return err
	}

	return utils.DumpSnapshotAsJson(snap, writer)
}

func (xds *XDS) RunServer(ctx context.Context, providerStartupTimeout time.Duration, onReady func()) error {
	logger := log.FromContext(ctx)
	startupCtx, cancel := context.WithTimeout(ctx, providerStartupTimeout)
	defer cancel()
	if err := xds.initProviders(startupCtx); err != nil {
		return fmt.Errorf("error initializing providers: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			time.Sleep(15 * time.Second)

			view, err := xds.takeView(ctx)
			if err != nil {
				logger.Error("Error taking view", log.Error(err))
				continue
			}
			if err := xds.updateView(ctx, view); err != nil {
				logger.Error("Error updating view", log.Error(err))
			}
			onReady()
		}
	}()

	cb := NewXDSCallbacks(log.PutIntoContext(ctx, logger.With(slog.String("component", "envoy-xds"))))
	xds.server = server.NewServer(ctx, xds.cacheManager, cb, sotw.WithOrderedADS())

	// Create gRPC server with auth interceptors
	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(TokenAuthInterceptor(xds.token, logger)),
		grpc.StreamInterceptor(TokenAuthStreamInterceptor(xds.token, logger)),
	}
	grpcServer := grpc.NewServer(grpcOpts...)
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, xds.server)
	lis, err := net.Listen("tcp", ":18000")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve failed: %w", err)
	}
	return nil
}

func (xds *XDS) calculateResourcesHash(resources map[resource.Type][]types.Resource) (string, error) {
	hasher := sha256.New()

	// Sort resource types for deterministic ordering
	var resourceTypes []resource.Type
	for resType := range resources {
		resourceTypes = append(resourceTypes, resType)
	}
	sort.Slice(resourceTypes, func(i, j int) bool {
		return resourceTypes[i] < resourceTypes[j]
	})

	// Hash each resource deterministically
	for _, resType := range resourceTypes {
		resList := resources[resType]

		// Marshal each resource and collect with its hash for sorting
		type resourceWithHash struct {
			hash string
			data []byte
		}
		var marshaledResources []resourceWithHash

		for _, res := range resList {
			// Marshal each resource deterministically
			data, err := proto.MarshalOptions{Deterministic: true}.Marshal(res)
			if err != nil {
				return "", fmt.Errorf("failed to marshal resource: %w", err)
			}
			// Calculate individual resource hash for sorting
			resHasher := sha256.New()
			resHasher.Write(data)
			marshaledResources = append(marshaledResources, resourceWithHash{
				hash: hex.EncodeToString(resHasher.Sum(nil)),
				data: data,
			})
		}

		// Sort by hash for deterministic ordering
		sort.Slice(marshaledResources, func(i, j int) bool {
			return marshaledResources[i].hash < marshaledResources[j].hash
		})

		// Add sorted resources to overall hash
		for _, mr := range marshaledResources {
			hasher.Write(mr.data)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (xds *XDS) updateView(ctx context.Context, view *LogicalView) error {
	logger := log.FromContext(ctx)

	resources := map[resource.Type][]types.Resource{
		resource.ListenerType: utils.CastListeners(view.Listeners()),
		resource.ClusterType:  utils.CastClusters(view.Clusters()),
	}

	// Calculate hash of resources
	newHash, err := xds.calculateResourcesHash(resources)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Skip update if hash hasn't changed
	if xds.lastHash == newHash {
		logger.Info("Configuration unchanged, skipping snapshot update", slog.String("hash", newHash))
		return nil
	}

	// Create snapshot with hash as version
	snap, err := cache.NewSnapshot(
		newHash,
		resources,
	)

	if err != nil {
		return err
	}

	if err := snap.Consistent(); err != nil {
		return err
	}

	if err := xds.cacheManager.SetSnapshot(ctx, "all", snap); err != nil {
		return err
	}

	logger.Info("Snapshot updated", slog.String("version", newHash), slog.String("previous_version", xds.lastHash))
	xds.lastHash = newHash
	return nil
}

type AllCache struct {
}

func (c AllCache) ID(node *corev3.Node) string {
	return "all"
}
