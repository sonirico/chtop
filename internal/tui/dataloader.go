package tui

import (
	"context"

	"github.com/sonirico/chtop/pkg/ch"
)

// dataLoader is the consumer-side interface the TUI uses to talk to a
// ClickHouse cluster. Kept small and unexported.
type dataLoader interface {
	Clusters(ctx context.Context) ([]ch.ClusterReplica, error)
	ListTables(ctx context.Context, includeSystem bool) ([]ch.TableInfo, error)
	Processes(ctx context.Context) ([]ch.Process, error)
	KillQuery(ctx context.Context, queryID string) error
	Replicas(ctx context.Context) ([]ch.ReplicaStatus, error)
	Merges(ctx context.Context) ([]ch.MergeInfo, error)
	DescribeTable(ctx context.Context, database, table string) (ch.TableDescription, error)
	Columns(ctx context.Context, database, table string) ([]ch.ColumnInfo, error)
	Parts(ctx context.Context, database, table string) ([]ch.PartInfo, error)
	Ping(ctx context.Context) error
	Close()
}
