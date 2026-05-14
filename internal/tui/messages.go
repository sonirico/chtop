package tui

import "github.com/sonirico/chtop/pkg/ch"

type clustersLoadedMsg struct {
	clusters []ch.ClusterReplica
}

type tablesLoadedMsg struct {
	tables []ch.TableInfo
}

type processesLoadedMsg struct {
	procs []ch.Process
}

type replicasLoadedMsg struct {
	replicas []ch.ReplicaStatus
}

type mergesLoadedMsg struct {
	merges []ch.MergeInfo
}

type tableDescribeLoadedMsg struct {
	desc ch.TableDescription
}

type columnsLoadedMsg struct {
	columns []ch.ColumnInfo
}

type partsLoadedMsg struct {
	parts []ch.PartInfo
}

type tableMutationsLoadedMsg struct {
	mutations []ch.MutationInfo
}

type queryLogLoadedMsg struct {
	entries []ch.QueryLogInfo
}

type explainLoadedMsg struct {
	tab  int
	body string
}

type metricsLoadedMsg struct {
	snapshot ch.MetricsSnapshot
}

type errorsLoadedMsg struct {
	errors []ch.ErrorInfo
}

type killDoneMsg struct {
	queryID string
	err     error
}

type errorMsg struct {
	err error
}

type switchViewMsg struct {
	view viewID
}

type tickMsg struct{}
