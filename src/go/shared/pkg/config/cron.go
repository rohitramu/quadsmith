package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"google.golang.org/protobuf/types/known/durationpb"
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
// anchored at the Unix epoch: January 1, 1970 at 12:00 AM UTC (time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)).
func DurationToCron(d time.Duration) (string, error) {
	if d <= 0 {
		return "", fmt.Errorf("duration must be strictly positive, got %v", d)
	}

	totalSeconds := int64(d.Seconds())
	if totalSeconds < 60 {
		return "", fmt.Errorf("duration must be at least 1 minute, got %v", d)
	}

	var cronExpr string

	// Case 1: Sub-hourly intervals (minutes dividing 60: 1m, 2m, 3m, 4m, 5m, 6m, 10m, 12m, 15m, 20m, 30m)
	if totalSeconds < 3600 {
		mins := totalSeconds / 60
		if 60%mins == 0 && totalSeconds%60 == 0 {
			if mins == 1 {
				cronExpr = "* * * * *"
			} else {
				cronExpr = fmt.Sprintf("*/%d * * * *", mins)
			}
		} else {
			var minutes []string
			for m := int64(0); m < 60; m += mins {
				minutes = append(minutes, fmt.Sprintf("%d", m))
			}
			cronExpr = fmt.Sprintf("%s * * * *", strings.Join(minutes, ","))
		}
	} else if totalSeconds <= 86400 {
		// Case 2: Sub-daily hourly intervals (hours dividing 24: 1h, 2h, 3h, 4h, 6h, 8h, 12h)
		if totalSeconds%3600 == 0 {
			hours := totalSeconds / 3600
			if 24%hours == 0 {
				if hours == 1 {
					cronExpr = "0 * * * *"
				} else {
					var hourList []string
					for h := int64(0); h < 24; h += hours {
						hourList = append(hourList, fmt.Sprintf("%d", h))
					}
					cronExpr = fmt.Sprintf("0 %s * * *", strings.Join(hourList, ","))
				}
			} else {
				var hourList []string
				for h := int64(0); h < 24; h += hours {
					hourList = append(hourList, fmt.Sprintf("%d", h))
				}
				cronExpr = fmt.Sprintf("0 %s * * *", strings.Join(hourList, ","))
			}
		} else {
			minPart := (totalSeconds % 3600) / 60
			hourStep := totalSeconds / 3600
			if hourStep == 0 {
				hourStep = 1
			}
			var hourList []string
			for h := int64(0); h < 24; h += hourStep {
				hourList = append(hourList, fmt.Sprintf("%d", h))
			}
			cronExpr = fmt.Sprintf("%d %s * * *", minPart, strings.Join(hourList, ","))
		}
	} else if totalSeconds == 86400 {
		// Case 3: Exactly 24 hours (1 day)
		cronExpr = "0 0 * * *"
	} else if totalSeconds%86400 == 0 {
		// Case 4: Multi-day intervals
		// Jan 1 1970 was Thursday (day 4).
		days := totalSeconds / 86400
		if 7%days == 0 {
			cronExpr = "0 0 * * 4"
		} else {
			var daysOfWeek []string
			for dIdx := int64(0); dIdx < 7; dIdx += days {
				dow := (4 + dIdx) % 7
				daysOfWeek = append(daysOfWeek, fmt.Sprintf("%d", dow))
			}
			cronExpr = fmt.Sprintf("0 0 * * %s", strings.Join(daysOfWeek, ","))
		}
	} else {
		return "", fmt.Errorf("unsupported interval duration %v: must be in minutes, hours, or days", d)
	}

	if err := ValidateCronExpression(cronExpr); err != nil {
		return "", fmt.Errorf("generated cron %q is invalid: %w", cronExpr, err)
	}
	return cronExpr, nil
}
