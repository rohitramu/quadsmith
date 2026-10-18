package ops

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"quadsmith/shared/pkg/cel2sql"

	pb "quadsmith/shared/gen/quadsmith/ops"
)

func tsUnixNano(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.AsTime().UnixNano()
}

var routeMetricFields = map[string]bool{
	"route":                true,
	"method":               true,
	"region":               true,
	"service_name":         true,
	"total_requests":       true,
	"success_count":        true,
	"error_count":          true,
	"availability_percent": true,
	"p50_rtt_ms":           true,
	"p95_rtt_ms":           true,
	"p99_rtt_ms":           true,
}

func filterAndSortRouteMetrics(items []*pb.RouteMetricStat, filter string, sortSpecs []string) ([]*pb.RouteMetricStat, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, routeMetricFields, func(r *pb.RouteMetricStat) map[string]any {
		return map[string]any{
			"route":                r.GetRoute(),
			"method":               r.GetMethod(),
			"region":               r.GetRegion(),
			"service_name":         r.GetServiceName(),
			"total_requests":       r.GetTotalRequests(),
			"success_count":        r.GetSuccessCount(),
			"error_count":          r.GetErrorCount(),
			"availability_percent": r.GetAvailabilityPercent(),
			"p50_rtt_ms":           r.GetP50RttMs(),
			"p95_rtt_ms":           r.GetP95RttMs(),
			"p99_rtt_ms":           r.GetP99RttMs(),
		}
	})
}

var routeRequestFields = map[string]bool{
	"trace_id":     true,
	"region":       true,
	"service_name": true,
	"instance_id":  true,
	"route":        true,
	"method":       true,
	"status_code":  true,
	"success":      true,
	"rtt_ms":       true,
	"occurred_at":  true,
}

func filterAndSortRouteRequests(items []*pb.RouteRequestLog, filter string, sortSpecs []string) ([]*pb.RouteRequestLog, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, routeRequestFields, func(r *pb.RouteRequestLog) map[string]any {
		return map[string]any{
			"trace_id":     r.GetTraceId(),
			"region":       r.GetRegion(),
			"service_name": r.GetServiceName(),
			"instance_id":  r.GetInstanceId(),
			"route":        r.GetRoute(),
			"method":       r.GetMethod(),
			"status_code":  r.GetStatusCode(),
			"success":      r.GetSuccess(),
			"rtt_ms":       r.GetRttMs(),
			"occurred_at":  tsUnixNano(r.GetOccurredAt()),
		}
	})
}

var taskFields = map[string]bool{
	"name":             true,
	"procedure":        true,
	"description":      true,
	"default_interval": true,
}

func filterAndSortTasks(items []*pb.TaskDef, filter string, sortSpecs []string) ([]*pb.TaskDef, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, taskFields, func(t *pb.TaskDef) map[string]any {
		return map[string]any{
			"name":             t.GetName(),
			"procedure":        t.GetProcedure(),
			"description":      t.GetDescription(),
			"default_interval": t.GetDefaultInterval(),
		}
	})
}

var jobFields = map[string]bool{
	"uuid":          true,
	"job_id":        true,
	"trace_id":      true,
	"region":        true,
	"task_name":     true,
	"procedure":     true,
	"caller":        true,
	"caller_type":   true,
	"status":        true,
	"duration_ms":   true,
	"error_message": true,
	"created_at":    true,
}

func filterAndSortJobs(items []*pb.JobRecord, filter string, sortSpecs []string) ([]*pb.JobRecord, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, jobFields, func(j *pb.JobRecord) map[string]any {
		return map[string]any{
			"uuid":          j.GetUuid(),
			"job_id":        j.GetJobId(),
			"trace_id":      j.GetTraceId(),
			"region":        j.GetRegion(),
			"task_name":     j.GetTaskName(),
			"procedure":     j.GetProcedure(),
			"caller":        j.GetCaller(),
			"caller_type":   j.GetCallerType(),
			"status":        j.GetStatus(),
			"duration_ms":   j.GetDurationMs(),
			"error_message": j.GetErrorMessage(),
			"created_at":    tsUnixNano(j.GetCreatedAt()),
		}
	})
}

var tableStatFields = map[string]bool{
	"table_name":       true,
	"row_count":        true,
	"total_size_bytes": true,
}

func filterAndSortTableStats(items []*pb.TableStat, filter string, sortSpecs []string) ([]*pb.TableStat, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, tableStatFields, func(t *pb.TableStat) map[string]any {
		return map[string]any{
			"table_name":       t.GetTableName(),
			"row_count":        t.GetRowCount(),
			"total_size_bytes": t.GetTotalSizeBytes(),
		}
	})
}

var iamUserFields = map[string]bool{
	"operator":       true,
	"type":           true,
	"group_name":     true,
	"has_active_jit": true,
	"last_active_at": true,
}

func filterAndSortIamUsers(items []*pb.IamUserSummary, filter string, sortSpecs []string) ([]*pb.IamUserSummary, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, iamUserFields, func(u *pb.IamUserSummary) map[string]any {
		return map[string]any{
			"operator":       u.GetOperator(),
			"type":           u.GetType(),
			"group_name":     u.GetGroupName(),
			"has_active_jit": u.GetHasActiveJit(),
			"last_active_at": tsUnixNano(u.GetLastActiveAt()),
		}
	})
}

var iamGroupFields = map[string]bool{
	"name":        true,
	"description": true,
}

func filterAndSortIamGroups(items []*pb.IamGroupSummary, filter string, sortSpecs []string) ([]*pb.IamGroupSummary, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, iamGroupFields, func(g *pb.IamGroupSummary) map[string]any {
		return map[string]any{
			"name":        g.GetName(),
			"description": g.GetDescription(),
		}
	})
}

var internalTokenFields = map[string]bool{
	"uuid":       true,
	"id":         true,
	"name":       true,
	"type":       true,
	"group_name": true,
	"status":     true,
	"expires_at": true,
	"created_at": true,
}

func computeTokenStatus(t *pb.InternalTokenRecord) string {
	if t.GetRevokedAt() != nil {
		return "REVOKED"
	}
	if t.GetExpiresAt() != nil && t.GetExpiresAt().AsTime().Before(time.Now()) {
		return "EXPIRED"
	}
	return "ACTIVE"
}

func filterAndSortInternalTokens(items []*pb.InternalTokenRecord, filter string, sortSpecs []string) ([]*pb.InternalTokenRecord, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, internalTokenFields, func(t *pb.InternalTokenRecord) map[string]any {
		return map[string]any{
			"uuid":       t.GetUuid(),
			"id":         t.GetId(),
			"name":       t.GetName(),
			"type":       t.GetType(),
			"group_name": t.GetGroupName(),
			"status":     computeTokenStatus(t),
			"expires_at": tsUnixNano(t.GetExpiresAt()),
			"created_at": tsUnixNano(t.GetCreatedAt()),
		}
	})
}

var internalApiCallFields = map[string]bool{
	"uuid":           true,
	"id":             true,
	"trace_id":       true,
	"region":         true,
	"method":         true,
	"caller":         true,
	"caller_type":    true,
	"group_name":     true,
	"jit_grant_uuid": true,
	"jit_grant_id":   true,
	"status":         true,
	"error_message":  true,
	"duration_ms":    true,
	"created_at":     true,
}

func filterAndSortInternalApiCalls(items []*pb.InternalApiCallRecord, filter string, sortSpecs []string) ([]*pb.InternalApiCallRecord, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, internalApiCallFields, func(c *pb.InternalApiCallRecord) map[string]any {
		return map[string]any{
			"uuid":           c.GetUuid(),
			"id":             c.GetId(),
			"trace_id":       c.GetTraceId(),
			"region":         c.GetRegion(),
			"method":         c.GetMethod(),
			"caller":         c.GetCaller(),
			"caller_type":    c.GetCallerType(),
			"group_name":     c.GetGroupName(),
			"jit_grant_uuid": c.GetJitGrantUuid(),
			"jit_grant_id":   c.GetJitGrantId(),
			"status":         c.GetStatus(),
			"error_message":  c.GetErrorMessage(),
			"duration_ms":    c.GetDurationMs(),
			"created_at":     tsUnixNano(c.GetCreatedAt()),
		}
	})
}

var jitGrantFields = map[string]bool{
	"uuid":          true,
	"id":            true,
	"operator":      true,
	"operator_type": true,
	"reason":        true,
	"operation":     true,
	"status":        true,
	"hours":         true,
	"expires_at":    true,
	"created_at":    true,
}

func filterAndSortJitGrants(items []*pb.JitGrantRecord, filter string, sortSpecs []string) ([]*pb.JitGrantRecord, error) {
	return cel2sql.FilterAndSort(items, filter, sortSpecs, jitGrantFields, func(g *pb.JitGrantRecord) map[string]any {
		return map[string]any{
			"uuid":          g.GetUuid(),
			"id":            g.GetId(),
			"operator":      g.GetOperator(),
			"operator_type": g.GetOperatorType(),
			"reason":        g.GetReason(),
			"operation":     g.GetOperation(),
			"status":        g.GetStatus(),
			"hours":         g.GetHours(),
			"expires_at":    tsUnixNano(g.GetExpiresAt()),
			"created_at":    tsUnixNano(g.GetCreatedAt()),
		}
	})
}
