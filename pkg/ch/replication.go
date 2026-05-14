package ch

import (
	"context"
	"fmt"
	"time"
)

// ReplicaStatus mirrors a row of system.replicas for one replicated table.
// Use the *Behind fields to spot replicas that are not caught up.
type ReplicaStatus struct {
	Database          string
	Table             string
	IsLeader          bool
	IsReadOnly        bool
	IsSessionExpired  bool
	FutureParts       uint32
	PartsToCheck      uint32
	QueueSize         uint32
	InsertsInQueue    uint32
	MergesInQueue     uint32
	LogMaxIndex       uint64
	LogPointer        uint64
	TotalReplicas     uint8
	ActiveReplicas    uint8
	AbsoluteDelay     time.Duration
	LastQueueUpdate   time.Time
}

// Replicas returns one row per replicated table on this replica.
func (c *Client) Replicas(ctx context.Context) ([]ReplicaStatus, error) {
	rows, err := c.conn.Query(ctx, `
		SELECT
			database, table, is_leader, is_readonly, is_session_expired,
			future_parts, parts_to_check, queue_size, inserts_in_queue,
			merges_in_queue, log_max_index, log_pointer,
			total_replicas, active_replicas, absolute_delay, last_queue_update
		FROM system.replicas
		ORDER BY absolute_delay DESC, database, table
	`)
	if err != nil {
		return nil, fmt.Errorf("query system.replicas: %w", err)
	}
	defer rows.Close()

	var out []ReplicaStatus
	for rows.Next() {
		var r ReplicaStatus
		var isLeader, isRO, isExpired uint8
		var absoluteDelay uint64
		if err := rows.Scan(
			&r.Database, &r.Table, &isLeader, &isRO, &isExpired,
			&r.FutureParts, &r.PartsToCheck, &r.QueueSize, &r.InsertsInQueue,
			&r.MergesInQueue, &r.LogMaxIndex, &r.LogPointer,
			&r.TotalReplicas, &r.ActiveReplicas, &absoluteDelay, &r.LastQueueUpdate,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		r.IsLeader = isLeader == 1
		r.IsReadOnly = isRO == 1
		r.IsSessionExpired = isExpired == 1
		r.AbsoluteDelay = time.Duration(absoluteDelay) * time.Second
		out = append(out, r)
	}
	return out, rows.Err()
}
