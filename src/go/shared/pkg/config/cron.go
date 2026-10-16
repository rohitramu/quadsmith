package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ValidateCronExpression verifies that expr is a valid 5-field standard cron expression.
func ValidateCronExpression(expr string) error {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}
	_, err := cronParser.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return nil
}

// DurationPbToCron converts a google.protobuf.Duration to a valid cron string anchored at Unix epoch.
func DurationPbToCron(pbDur *durationpb.Duration) (string, error) {
	if pbDur == nil {
		return "", fmt.Errorf("duration is nil")
	}
	return DurationToCron(pbDur.AsDuration())
}

// DurationToCron converts a positive time.Duration into a standard 5-field cron expression,
// anchored at the Unix epoch: January 1, 1970 at 12:00 AM UTC (a Thursday).
//
// Because standard 5-field cron requires recurring intervals to align with standard unit
// cycles without drift, supported durations are:
// - Whole minutes that evenly divide 60 minutes: 1m, 2m, 3m, 4m, 5m, 6m, 10m, 12m, 15m, 20m, 30m
// - Whole hours that evenly divide 24 hours: 1h, 2h, 3h, 4h, 6h, 8h, 12h, 24h
// - 7 days (weekly anchored to Thursday): 7d / 168h
// For irregular intervals or other anchors, an explicit cron expression should be used.
func DurationToCron(d time.Duration) (string, error) {
	if d <= 0 {
		return "", fmt.Errorf("duration must be strictly positive, got %v", d)
	}

	totalSeconds := int64(d.Seconds())
	if totalSeconds < 60 {
		return "", fmt.Errorf("duration must be at least 1 minute, got %v", d)
	}

	var cronExpr string

	if totalSeconds < 3600 {
		// Case 1: Sub-hourly minute intervals (must evenly divide 60 minutes)
		if totalSeconds%60 != 0 || 60%(totalSeconds/60) != 0 {
			return "", fmt.Errorf("minute duration %v does not evenly divide 60 minutes; supported intervals are 1m, 2m, 3m, 4m, 5m, 6m, 10m, 12m, 15m, 20m, 30m, or use a cron expression", d)
		}
		mins := totalSeconds / 60
		if mins == 1 {
			cronExpr = "* * * * *"
		} else {
			cronExpr = fmt.Sprintf("*/%d * * * *", mins)
		}
	} else if totalSeconds <= 86400 {
		// Case 2: Sub-daily hourly intervals (must evenly divide 24 hours)
		if totalSeconds%3600 != 0 || 24%(totalSeconds/3600) != 0 {
			return "", fmt.Errorf("hourly duration %v does not evenly divide 24 hours; supported intervals are 1h, 2h, 3h, 4h, 6h, 8h, 12h, 24h, or use a cron expression", d)
		}
		hours := totalSeconds / 3600
		if hours == 1 {
			cronExpr = "0 * * * *"
		} else if hours == 24 {
			cronExpr = "0 0 * * *"
		} else {
			var hourList []string
			for h := int64(0); h < 24; h += hours {
				hourList = append(hourList, fmt.Sprintf("%d", h))
			}
			cronExpr = fmt.Sprintf("0 %s * * *", strings.Join(hourList, ","))
		}
	} else {
		// Case 3: Multi-day intervals (Jan 1 1970 was Thursday, day 4)
		if totalSeconds%86400 != 0 || totalSeconds/86400 != 7 {
			return "", fmt.Errorf("multi-day duration %v cannot be accurately expressed in standard cron; standard cron only supports 7d (weekly), or use an explicit cron expression", d)
		}
		cronExpr = "0 0 * * 4"
	}

	if err := ValidateCronExpression(cronExpr); err != nil {
		return "", fmt.Errorf("generated cron %q is invalid: %w", cronExpr, err)
	}
	return cronExpr, nil
}
