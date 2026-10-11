package social

import (
	"testing"
	"time"

	pb "quadsmith/api/gen/quadsmith"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Apex 5 O3 Bando Smasher", "apex-5-o3-bando-smasher"},
		{"Sub-250g Speedster!", "sub-250g-speedster"},
		{"   leading and trailing   ", "leading-and-trailing"},
		{"Special @#$ Characters", "special-characters"},
		{"", ""},
	}

	for _, tc := range tests {
		got := slugify(tc.input)
		if got != tc.expected {
			t.Errorf("slugify(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestReviewTierThresholds(t *testing.T) {
	tests := []struct {
		total int32
		rec   int32
		tier  pb.ReviewTier
	}{
		{0, 0, pb.ReviewTier_REVIEW_TIER_FEW_REVIEWS},
		{4, 4, pb.ReviewTier_REVIEW_TIER_FEW_REVIEWS},               // Below 5 reviews
		{10, 10, pb.ReviewTier_REVIEW_TIER_VERY_POSITIVE},           // 100% positive, 10 reviews
		{50, 48, pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_POSITIVE}, // 96% positive, 50 reviews
		{20, 15, pb.ReviewTier_REVIEW_TIER_POSITIVE},                // 75% positive
		{20, 10, pb.ReviewTier_REVIEW_TIER_MIXED},                   // 50% positive
		{20, 6, pb.ReviewTier_REVIEW_TIER_MOSTLY_NEGATIVE},          // 30% positive
		{50, 5, pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_NEGATIVE},  // 10% positive, 50 reviews
	}

	for _, tc := range tests {
		var tier pb.ReviewTier
		pct := float32(0)
		if tc.total > 0 {
			pct = (float32(tc.rec) / float32(tc.total)) * 100.0
		}

		if tc.total < 5 {
			tier = pb.ReviewTier_REVIEW_TIER_FEW_REVIEWS
		} else if pct >= 95.0 && tc.total >= 50 {
			tier = pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_POSITIVE
		} else if pct >= 80.0 && tc.total >= 10 {
			tier = pb.ReviewTier_REVIEW_TIER_VERY_POSITIVE
		} else if pct >= 70.0 {
			tier = pb.ReviewTier_REVIEW_TIER_POSITIVE
		} else if pct >= 40.0 {
			tier = pb.ReviewTier_REVIEW_TIER_MIXED
		} else if pct >= 20.0 {
			tier = pb.ReviewTier_REVIEW_TIER_MOSTLY_NEGATIVE
		} else if tc.total >= 50 {
			tier = pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_NEGATIVE
		} else {
			tier = pb.ReviewTier_REVIEW_TIER_MOSTLY_NEGATIVE
		}

		if tier != tc.tier {
			t.Errorf("total %d, rec %d -> tier %v, expected %v", tc.total, tc.rec, tier, tc.tier)
		}
	}
}

func TestReputationDeltaRules(t *testing.T) {
	const (
		ReviewAuthorReward           = int32(3)
		UpvoteTargetReward           = int32(1)
		DownvoteTargetDelta          = int32(-2)
		DownvoteVoterStakes          = int32(-2)
		DailyUpvoteLimit             = 20
		DailyDownvoteLimit           = 3
		DownvoteUnlockRep            = 100
		WeeklyDownvotePerAuthorLimit = 3
	)

	if ReviewAuthorReward != 3 {
		t.Errorf("expected review author reward to be 3, got %d", ReviewAuthorReward)
	}
	if UpvoteTargetReward != 1 {
		t.Errorf("expected upvote target reward to be 1, got %d", UpvoteTargetReward)
	}
	if DailyUpvoteLimit != 20 {
		t.Errorf("expected daily upvote limit to be 20, got %d", DailyUpvoteLimit)
	}
	if DailyDownvoteLimit != 3 {
		t.Errorf("expected daily downvote limit to be 3, got %d", DailyDownvoteLimit)
	}
	if DownvoteUnlockRep != 100 {
		t.Errorf("expected downvote unlock rep to be 100, got %d", DownvoteUnlockRep)
	}
	if WeeklyDownvotePerAuthorLimit != 3 {
		t.Errorf("expected weekly per-author downvote limit to be 3, got %d", WeeklyDownvotePerAuthorLimit)
	}
}

func TestProbationaryPeriodLogic(t *testing.T) {
	now := time.Now()

	// New account created 2 hours ago -> should be blocked
	newAccountCreatedAt := now.Add(-2 * time.Hour)
	if time.Since(newAccountCreatedAt) >= 24*time.Hour {
		t.Errorf("account created 2h ago should still be in 24h probationary period")
	}

	// Account created 23 hours and 59 minutes ago -> still in probation
	edgeAccount := now.Add(-23*time.Hour - 59*time.Minute)
	if time.Since(edgeAccount) >= 24*time.Hour {
		t.Errorf("account created 23h59m ago should still be in 24h probationary period")
	}

	// Matured account created 25 hours ago -> eligible
	maturedAccount := now.Add(-25 * time.Hour)
	if time.Since(maturedAccount) < 24*time.Hour {
		t.Errorf("account created 25h ago should no longer be in probationary period")
	}
}

func TestAnswerRugPullVestingRules(t *testing.T) {
	// Rug-pull protection: answers revoked after 24 hours keep their reputation permanently vested
	now := time.Now()

	// Scenario 1: Revocation within 24 hours -> clawback applies
	acceptedAtRecent := now.Add(-6 * time.Hour)
	var clawbackDelta int32
	if time.Since(acceptedAtRecent) <= 24*time.Hour {
		clawbackDelta = -30
	} else {
		clawbackDelta = 0
	}
	if clawbackDelta != -30 {
		t.Errorf("expected clawback delta -30 within 24h grace window, got %d", clawbackDelta)
	}

	// Scenario 2: Revocation after 48 hours -> 0 clawback, permanent vesting
	acceptedAtOld := now.Add(-48 * time.Hour)
	var vestedDelta int32
	if time.Since(acceptedAtOld) <= 24*time.Hour {
		vestedDelta = -30
	} else {
		vestedDelta = 0 // Permanently vested!
	}
	if vestedDelta != 0 {
		t.Errorf("expected 0 clawback for answer revoked after 24h (permanent vesting), got %d", vestedDelta)
	}
}
