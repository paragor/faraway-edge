package envoy

import "context"

type LogicalClusterProvider interface {
	GetLogicaCluster(ctx context.Context) (*LogicalCluster, error)
}

type StaticLogicalClusterProvider struct {
	cluster *LogicalCluster
}

func (p *StaticLogicalClusterProvider) GetLogicaCluster(ctx context.Context) (*LogicalCluster, error) {
	return p.cluster, nil
}

func NewStaticLogicalClusterProvider(cluster *LogicalCluster) *StaticLogicalClusterProvider {
	return &StaticLogicalClusterProvider{cluster: cluster}
}
