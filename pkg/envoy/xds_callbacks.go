package envoy

import (
	"context"
	"log/slog"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/paragor/faraway-edge/pkg/log"
)

// XDSCallbacks implements server.Callbacks with structured logging
type XDSCallbacks struct {
	ctx context.Context
}

func NewXDSCallbacks(ctx context.Context) *XDSCallbacks {
	return &XDSCallbacks{ctx: ctx}
}

func (cb *XDSCallbacks) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	logger := log.FromContext(cb.ctx)
	node := extractNodeFromContext(ctx)
	logger.Info("stream opened",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", typeURL),
		slog.String("node_id", node.Id),
		slog.String("node_cluster", node.Cluster))
	return nil
}

func extractNodeFromContext(ctx context.Context) *corev3.Node {
	// Try to extract node from context value
	if node, ok := ctx.Value("node").(*corev3.Node); ok && node != nil {
		return node
	}
	// Return empty node if not found
	return &corev3.Node{}
}

func (cb *XDSCallbacks) OnStreamClosed(streamID int64, node *corev3.Node) {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if node != nil {
		nodeID = node.Id
		nodeCluster = node.Cluster
	}
	logger.Info("stream closed",
		slog.Int64("stream_id", streamID),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
}

func (cb *XDSCallbacks) OnStreamRequest(streamID int64, req *discoveryv3.DiscoveryRequest) error {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("received request",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", req.TypeUrl),
		slog.String("version", req.VersionInfo),
		slog.Int("resource_names_count", len(req.ResourceNames)),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
	return nil
}

func (cb *XDSCallbacks) OnStreamResponse(ctx context.Context, streamID int64, req *discoveryv3.DiscoveryRequest, resp *discoveryv3.DiscoveryResponse) {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("responding to request",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", resp.TypeUrl),
		slog.String("version", resp.VersionInfo),
		slog.Int("resources_count", len(resp.Resources)),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
}

func (cb *XDSCallbacks) OnFetchRequest(ctx context.Context, req *discoveryv3.DiscoveryRequest) error {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("fetch request received",
		slog.String("type_url", req.TypeUrl),
		slog.String("version", req.VersionInfo),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
	return nil
}

func (cb *XDSCallbacks) OnFetchResponse(req *discoveryv3.DiscoveryRequest, resp *discoveryv3.DiscoveryResponse) {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("fetch response sent",
		slog.String("type_url", resp.TypeUrl),
		slog.String("version", resp.VersionInfo),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
}

func (cb *XDSCallbacks) OnDeltaStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	logger := log.FromContext(cb.ctx)
	node := extractNodeFromContext(ctx)
	logger.Info("delta stream opened",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", typeURL),
		slog.String("node_id", node.Id),
		slog.String("node_cluster", node.Cluster))
	return nil
}

func (cb *XDSCallbacks) OnDeltaStreamClosed(streamID int64, node *corev3.Node) {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if node != nil {
		nodeID = node.Id
		nodeCluster = node.Cluster
	}
	logger.Info("delta stream closed",
		slog.Int64("stream_id", streamID),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
}

func (cb *XDSCallbacks) OnStreamDeltaRequest(streamID int64, req *discoveryv3.DeltaDiscoveryRequest) error {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("delta request received",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", req.TypeUrl),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
	return nil
}

func (cb *XDSCallbacks) OnStreamDeltaResponse(streamID int64, req *discoveryv3.DeltaDiscoveryRequest, resp *discoveryv3.DeltaDiscoveryResponse) {
	logger := log.FromContext(cb.ctx)
	nodeID := ""
	nodeCluster := ""
	if req.Node != nil {
		nodeID = req.Node.Id
		nodeCluster = req.Node.Cluster
	}
	logger.Info("delta response sent",
		slog.Int64("stream_id", streamID),
		slog.String("type_url", resp.TypeUrl),
		slog.String("node_id", nodeID),
		slog.String("node_cluster", nodeCluster))
}