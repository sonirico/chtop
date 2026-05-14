package tui

import (
	"testing"
	"time"

	"github.com/sonirico/chtop/pkg/ch"
	"github.com/stretchr/testify/require"
)

func TestKafkaConsumerRow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Second)
	stale := now.Add(-90 * time.Second)

	type testCase struct {
		name string
		k    ch.KafkaConsumerInfo
		want []string
	}
	cases := []testCase{
		{
			name: "healthy consumer polling recently",
			k: ch.KafkaConsumerInfo{
				Database:        "kafka_in",
				Table:           "events_queue",
				ConsumerID:      "consumer-1",
				NumMessagesRead: 12_345_678,
				NumCommits:      1234,
				LastPollTime:    recent,
			},
			want: []string{
				"kafka_in.events_queue", "consumer-1",
				"12.35M", "1.23K", "5.0s ago", "ok",
			},
		},
		{
			name: "stale consumer hasn't polled in 90s",
			k: ch.KafkaConsumerInfo{
				Database:        "kafka_in",
				Table:           "slow_queue",
				ConsumerID:      "consumer-2",
				NumMessagesRead: 100,
				NumCommits:      1,
				LastPollTime:    stale,
			},
			want: []string{
				"kafka_in.slow_queue", "consumer-2",
				"100", "1", "1m30s ago", "stale",
			},
		},
		{
			name: "never polled (zero time)",
			k: ch.KafkaConsumerInfo{
				Database:        "kafka_in",
				Table:           "new_queue",
				ConsumerID:      "consumer-3",
				NumMessagesRead: 0,
				NumCommits:      0,
				LastPollTime:    time.Time{},
			},
			want: []string{
				"kafka_in.new_queue", "consumer-3",
				"0", "0", "never", "stale",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, kafkaConsumerRow(now, tc.k))
		})
	}
}
