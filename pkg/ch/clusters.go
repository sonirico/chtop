package ch

import (
	"context"
	"fmt"
)

// ClusterReplica is one row from system.clusters: cluster name, shard,
// replica, host details and errors counter.
type ClusterReplica struct {
	Cluster              string
	ShardNum             uint32
	ShardWeight          uint32
	ReplicaNum           uint32
	HostName             string
	HostAddress          string
	Port                 uint16
	IsLocal              bool
	User                 string
	DefaultDatabase      string
	ErrorsCount          uint32
	EstimatedRecoveryTime uint32
}

// Clusters lists every row in system.clusters across all configured
// clusters. Sort is by cluster name, shard, replica.
func (c *Client) Clusters(ctx context.Context) ([]ClusterReplica, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			cluster, shard_num, shard_weight, replica_num,
			host_name, host_address, port, is_local,
			user, default_database, errors_count, estimated_recovery_time
		FROM system.clusters
		ORDER BY cluster, shard_num, replica_num
	`)
	if err != nil {
		return nil, fmt.Errorf("query system.clusters: %w", err)
	}
	defer rows.Close()

	var out []ClusterReplica
	for rows.Next() {
		var r ClusterReplica
		var isLocal uint8
		if err := rows.Scan(
			&r.Cluster, &r.ShardNum, &r.ShardWeight, &r.ReplicaNum,
			&r.HostName, &r.HostAddress, &r.Port, &isLocal,
			&r.User, &r.DefaultDatabase, &r.ErrorsCount, &r.EstimatedRecoveryTime,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		r.IsLocal = isLocal == 1
		out = append(out, r)
	}
	return out, rows.Err()
}
