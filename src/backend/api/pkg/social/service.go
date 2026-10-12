package social

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/authcontext"
)

type SocialService struct {
	quadsmithconnect.UnimplementedSocialServiceHandler
	db *pgxpool.Pool
}

func NewSocialService(db *pgxpool.Pool) *SocialService {
	return &SocialService{db: db}
}

// ---------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------

func slugify(text string) string {
	reg := regexp.MustCompile("[^a-z0-9]+")
	s := strings.Trim(reg.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func resolveEntity(ctx context.Context, db *pgxpool.Pool, entityID string, collection *string) (entityUUID string, resolvedCol string, name string, image string, authorOrManufacturer string, isArchived bool, err error) {
	trimmedID := strings.TrimSpace(entityID)
	if trimmedID == "" {
		return "", "", "", "", "", false, connect.NewError(connect.CodeInvalidArgument, errors.New("entity_id is required"))
	}
	col := ""
	if collection != nil {
		col = strings.TrimSpace(*collection)
		if col == "build" {
			col = "builds"
		}
	}

	if col != "" {
		err = db.QueryRow(ctx, `
			SELECT uuid, collection, name, COALESCE(primary_display_image, ''), COALESCE(author_or_manufacturer, ''), is_archived
			FROM entities
			WHERE collection = $1 AND (id = $2 OR uuid::text = $2)
			LIMIT 1
		`, col, trimmedID).Scan(&entityUUID, &resolvedCol, &name, &image, &authorOrManufacturer, &isArchived)
	} else {
		err = db.QueryRow(ctx, `
			SELECT uuid, collection, name, COALESCE(primary_display_image, ''), COALESCE(author_or_manufacturer, ''), is_archived
			FROM entities
			WHERE id = $1 OR uuid::text = $1
			LIMIT 1
		`, trimmedID).Scan(&entityUUID, &resolvedCol, &name, &image, &authorOrManufacturer, &isArchived)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", "", false, connect.NewError(connect.CodeNotFound, fmt.Errorf("entity '%s' not found", trimmedID))
		}
		return "", "", "", "", "", false, connect.NewError(connect.CodeInternal, err)
	}
	return entityUUID, resolvedCol, name, image, authorOrManufacturer, isArchived, nil
}

func checkProbationaryPeriod(ctx context.Context, db *pgxpool.Pool, userUUID string) error {
	var createdAt time.Time
	err := db.QueryRow(ctx, "SELECT created_at FROM users WHERE uuid = $1", userUUID).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	if time.Since(createdAt) < 24*time.Hour {
		return connect.NewError(connect.CodePermissionDenied, errors.New("new accounts have a 24-hour read-only probationary period before interacting with the community"))
	}
	return nil
}

func getUserDisplay(ctx context.Context, db *pgxpool.Pool, userUUID string) (username string, avatarURL string) {
	_ = db.QueryRow(ctx, `
		SELECT u.id, COALESCE(p.avatar_url, '')
		FROM users u
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		WHERE u.uuid = $1
	`, userUUID).Scan(&username, &avatarURL)
	return username, avatarURL
}

func adjustReputation(ctx context.Context, db *pgxpool.Pool, userUUID string, delta int32) error {
	if delta == 0 {
		return nil
	}
	repUUID := authcontext.NewUUIDv7()
	_, err := db.Exec(ctx, `
		INSERT INTO user_reputations (uuid, user_uuid, score, updated_at)
		VALUES ($1, $2, GREATEST(0, $3), NOW())
		ON CONFLICT (user_uuid) DO UPDATE
		SET score = GREATEST(0, user_reputations.score + $3), updated_at = NOW()
	`, repUUID, userUUID, delta)
	return err
}

func checkDownvoteEligibility(ctx context.Context, db *pgxpool.Pool, userUUID, targetAuthorUUID string) error {
	var currentScore int32
	_ = db.QueryRow(ctx, "SELECT score FROM user_reputations WHERE user_uuid = $1", userUUID).Scan(&currentScore)
	if currentScore < 100 {
		return connect.NewError(connect.CodePermissionDenied, errors.New("downvoting requires at least 100 reputation points to prevent griefing"))
	}

	var downvoteCount int64
	err := db.QueryRow(ctx, `
		SELECT (
			(SELECT count(*) FROM question_votes WHERE user_uuid = $1 AND is_upvote = false AND created_at > NOW() - INTERVAL '24 hours') +
			(SELECT count(*) FROM answer_votes WHERE user_uuid = $1 AND is_upvote = false AND created_at > NOW() - INTERVAL '24 hours') +
			(SELECT count(*) FROM review_votes WHERE user_uuid = $1 AND is_helpful = false AND created_at > NOW() - INTERVAL '24 hours')
		)
	`, userUUID).Scan(&downvoteCount)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if downvoteCount >= 3 {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("maximum of 3 downvotes per rolling 24-hour window reached"))
	}

	if targetAuthorUUID != "" {
		var targetWeeklyCount int64
		err = db.QueryRow(ctx, `
			SELECT (
				(SELECT count(*) FROM question_votes qv JOIN questions q ON q.uuid = qv.question_uuid WHERE qv.user_uuid = $1 AND q.user_uuid = $2 AND qv.is_upvote = false AND qv.created_at > NOW() - INTERVAL '7 days') +
				(SELECT count(*) FROM answer_votes av JOIN answers a ON a.uuid = av.answer_uuid WHERE av.user_uuid = $1 AND a.user_uuid = $2 AND av.is_upvote = false AND av.created_at > NOW() - INTERVAL '7 days') +
				(SELECT count(*) FROM review_votes rv JOIN reviews r ON r.uuid = rv.review_uuid WHERE rv.user_uuid = $1 AND r.user_uuid = $2 AND rv.is_helpful = false AND rv.created_at > NOW() - INTERVAL '7 days')
			)
		`, userUUID, targetAuthorUUID).Scan(&targetWeeklyCount)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		if targetWeeklyCount >= 3 {
			return connect.NewError(connect.CodeResourceExhausted, errors.New("maximum of 3 downvotes against the same author per 7-day window reached"))
		}
	}

	return nil
}

func checkUpvoteEligibility(ctx context.Context, db *pgxpool.Pool, userUUID string) error {
	var upvoteCount int64
	err := db.QueryRow(ctx, `
		SELECT (
			(SELECT count(*) FROM question_votes WHERE user_uuid = $1 AND is_upvote = true AND created_at > NOW() - INTERVAL '24 hours') +
			(SELECT count(*) FROM answer_votes WHERE user_uuid = $1 AND is_upvote = true AND created_at > NOW() - INTERVAL '24 hours') +
			(SELECT count(*) FROM review_votes WHERE user_uuid = $1 AND is_helpful = true AND created_at > NOW() - INTERVAL '24 hours')
		)
	`, userUUID).Scan(&upvoteCount)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if upvoteCount >= 20 {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("maximum of 20 upvotes per rolling 24-hour window reached"))
	}
	return nil
}

func calculateReviewSummary(ctx context.Context, db *pgxpool.Pool, entityUUID string) (*pb.ReviewSummary, error) {
	var total, recCount int32
	err := db.QueryRow(ctx, `
		SELECT count(*), COALESCE(count(*) FILTER (WHERE is_recommended = true), 0)
		FROM reviews
		WHERE entity_uuid = $1 AND deleted_at IS NULL
	`, entityUUID).Scan(&total, &recCount)
	if err != nil {
		return nil, err
	}

	summary := &pb.ReviewSummary{
		TotalReviews:     total,
		RecommendedCount: recCount,
	}

	if total == 0 {
		summary.ScorePercentage = 0
		summary.Tier = pb.ReviewTier_REVIEW_TIER_FEW_REVIEWS
		return summary, nil
	}

	pct := (float32(recCount) / float32(total)) * 100.0
	summary.ScorePercentage = float32(math.Round(float64(pct)*10) / 10)

	if total < 5 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_FEW_REVIEWS
	} else if pct >= 95.0 && total >= 50 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_POSITIVE
	} else if pct >= 80.0 && total >= 10 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_VERY_POSITIVE
	} else if pct >= 70.0 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_POSITIVE
	} else if pct >= 40.0 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_MIXED
	} else if pct >= 20.0 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_MOSTLY_NEGATIVE
	} else if total >= 50 {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_OVERWHELMINGLY_NEGATIVE
	} else {
		summary.Tier = pb.ReviewTier_REVIEW_TIER_MOSTLY_NEGATIVE
	}

	return summary, nil
}

// ---------------------------------------------------------
// Review RPCs
// ---------------------------------------------------------

func (s *SocialService) CreateReview(ctx context.Context, req *connect.Request[pb.CreateReviewRequest]) (*connect.Response[pb.Review], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	entityUUID, resolvedCol, _, _, _, isArchived, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	if isArchived {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot submit a review for an archived entity"))
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("review writeup cannot be empty"))
	}

	// 1. Self-Review Prevention: Build authors cannot review their own builds
	if resolvedCol == "build" || resolvedCol == "builds" {
		var authorUUID sql.NullString
		err := s.db.QueryRow(ctx, "SELECT user_uuid FROM builds WHERE uuid = $1", entityUUID).Scan(&authorUUID)
		if err == nil && authorUUID.Valid && authorUUID.String == u.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("authors cannot submit a review for their own build"))
		}
	}

	// 2. Exactly 1 review per user per target entity
	var existingUUID string
	_ = s.db.QueryRow(ctx, "SELECT uuid FROM reviews WHERE user_uuid = $1 AND entity_uuid = $2 AND deleted_at IS NULL", u.UserUUID, entityUUID).Scan(&existingUUID)
	if existingUUID != "" {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("you have already reviewed this item; you can update your existing review"))
	}

	reviewUUID := authcontext.NewUUIDv7()
	reviewID := "rev-" + reviewUUID[:8]
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)

	_, err = s.db.Exec(ctx, `
		INSERT INTO reviews (
			uuid, id, user_uuid, entity_uuid, is_recommended, body, is_edited,
			author_username, author_avatar_url, helpful_count, unhelpful_count,
			comment_count, current_user_vote, is_author_verified_owner,
			collection, entity_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, 0, 0, 0, 0, false, $9, $10, NOW(), NOW())
	`, reviewUUID, reviewID, u.UserUUID, entityUUID, req.Msg.GetIsRecommended(), body, username, avatar, resolvedCol, req.Msg.GetEntityId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Reward review author with +3 reputation points
	_ = adjustReputation(ctx, s.db, u.UserUUID, +3)

	res := &pb.Review{
		Uuid:                  reviewUUID,
		Id:                    reviewID,
		UserUuid:              u.UserUUID,
		EntityUuid:            entityUUID,
		Collection:            resolvedCol,
		EntityId:              req.Msg.GetEntityId(),
		IsRecommended:         req.Msg.GetIsRecommended(),
		Body:                  body,
		IsEdited:              false,
		CreatedAt:             timestamppb.Now(),
		UpdatedAt:             timestamppb.Now(),
		AuthorUsername:        username,
		AuthorAvatarUrl:       avatar,
		HelpfulCount:          0,
		UnhelpfulCount:        0,
		CommentCount:          0,
		CurrentUserVote:       pb.ReviewVoteType_REVIEW_VOTE_TYPE_UNSPECIFIED,
		IsAuthorVerifiedOwner: false,
	}

	return connect.NewResponse(res), nil
}

func (s *SocialService) UpdateReview(ctx context.Context, req *connect.Request[pb.UpdateReviewRequest]) (*connect.Response[pb.Review], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	revID := strings.TrimSpace(req.Msg.GetReviewId())
	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("review body cannot be empty"))
	}

	var revUUID, userUUID, entityUUID, col, entityID string
	var createdAt time.Time
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid, entity_uuid, collection, entity_id, created_at FROM reviews WHERE id = $1 OR uuid::text = $1", revID).Scan(&revUUID, &userUUID, &entityUUID, &col, &entityID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot edit another user's review"))
	}

	_, err = s.db.Exec(ctx, `
		UPDATE reviews
		SET is_recommended = $1, body = $2, is_edited = true, updated_at = NOW()
		WHERE uuid = $3
	`, req.Msg.GetIsRecommended(), body, revUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	username, avatar := getUserDisplay(ctx, s.db, userUUID)

	return connect.NewResponse(&pb.Review{
		Uuid:            revUUID,
		Id:              revID,
		UserUuid:        userUUID,
		EntityUuid:      entityUUID,
		Collection:      col,
		EntityId:        entityID,
		IsRecommended:   req.Msg.GetIsRecommended(),
		Body:            body,
		IsEdited:        true,
		CreatedAt:       timestamppb.New(createdAt),
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteReview(ctx context.Context, req *connect.Request[pb.DeleteReviewRequest]) (*connect.Response[pb.ReviewSummary], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	revID := strings.TrimSpace(req.Msg.GetReviewId())
	var revUUID, userUUID, entityUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid, entity_uuid FROM reviews WHERE id = $1 OR uuid::text = $1", revID).Scan(&revUUID, &userUUID, &entityUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's review"))
	}

	// Soft-delete review and remove from calculation
	_, err = s.db.Exec(ctx, "UPDATE reviews SET deleted_at = NOW() WHERE uuid = $1", revUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Revert review author +3 reputation reward
	_ = adjustReputation(ctx, s.db, userUUID, -3)

	summary, err := calculateReviewSummary(ctx, s.db, entityUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(summary), nil
}

func (s *SocialService) ListReviews(ctx context.Context, req *connect.Request[pb.ListReviewsRequest]) (*connect.Response[pb.ListReviewsResponse], error) {
	entityUUID, _, _, _, _, _, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	currentCallerUUID := ""
	if u := authcontext.UserContextFromContext(ctx); u != nil {
		currentCallerUUID = u.UserUUID
	}

	orderClause := "ORDER BY (helpful_count - unhelpful_count) DESC, r.created_at DESC"
	switch req.Msg.GetSortBy() {
	case "recent":
		orderClause = "ORDER BY r.created_at DESC"
	case "positive":
		orderClause = "ORDER BY r.is_recommended DESC, (helpful_count - unhelpful_count) DESC"
	case "negative":
		orderClause = "ORDER BY r.is_recommended ASC, (helpful_count - unhelpful_count) DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			r.uuid, r.id, r.user_uuid, r.entity_uuid, r.collection, r.entity_id,
			r.is_recommended, r.body, r.is_edited, r.created_at, r.updated_at,
			u.id as author_username, COALESCE(p.avatar_url, '') as author_avatar_url,
			COALESCE(SUM(CASE WHEN v.is_helpful = true THEN 1 ELSE 0 END), 0) as helpful_count,
			COALESCE(SUM(CASE WHEN v.is_helpful = false THEN 1 ELSE 0 END), 0) as unhelpful_count,
			COALESCE(c.comment_count, 0) as comment_count,
			COALESCE(MAX(CASE WHEN v.user_uuid = $2 THEN (CASE WHEN v.is_helpful THEN 1 ELSE 2 END) ELSE 0 END), 0) as user_vote
		FROM reviews r
		JOIN users u ON u.uuid = r.user_uuid
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		LEFT JOIN review_votes v ON v.review_uuid = r.uuid
		LEFT JOIN (
			SELECT review_uuid, count(*) as comment_count
			FROM review_comments
			WHERE deleted_at IS NULL
			GROUP BY review_uuid
		) c ON c.review_uuid = r.uuid
		WHERE r.entity_uuid = $1 AND r.deleted_at IS NULL
		GROUP BY r.uuid, r.id, r.user_uuid, r.entity_uuid, r.collection, r.entity_id, r.is_recommended, r.body, r.is_edited, r.created_at, r.updated_at, u.id, p.avatar_url, c.comment_count
		%s
		LIMIT 50
	`, orderClause)

	rows, err := s.db.Query(ctx, query, entityUUID, currentCallerUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var reviews []*pb.Review
	for rows.Next() {
		var rev pb.Review
		var createdAt, updatedAt time.Time
		var userVoteInt int32

		err := rows.Scan(
			&rev.Uuid, &rev.Id, &rev.UserUuid, &rev.EntityUuid, &rev.Collection, &rev.EntityId,
			&rev.IsRecommended, &rev.Body, &rev.IsEdited, &createdAt, &updatedAt,
			&rev.AuthorUsername, &rev.AuthorAvatarUrl, &rev.HelpfulCount, &rev.UnhelpfulCount, &rev.CommentCount, &userVoteInt,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		rev.CreatedAt = timestamppb.New(createdAt)
		rev.UpdatedAt = timestamppb.New(updatedAt)
		rev.CurrentUserVote = pb.ReviewVoteType(userVoteInt)

		reviews = append(reviews, &rev)
	}

	summary, err := calculateReviewSummary(ctx, s.db, entityUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ListReviewsResponse{
		Reviews: reviews,
		Summary: summary,
	}), nil
}

func (s *SocialService) VoteReview(ctx context.Context, req *connect.Request[pb.VoteReviewRequest]) (*connect.Response[pb.VoteReviewResponse], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	revID := strings.TrimSpace(req.Msg.GetReviewId())
	var revUUID, authorUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM reviews WHERE id = $1 OR uuid::text = $1", revID).Scan(&revUUID, &authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	if authorUUID == u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot vote on your own review"))
	}

	// If casting a helpful vote, enforce upvote eligibility cap (20/24h)
	if req.Msg.GetVoteType() == pb.ReviewVoteType_REVIEW_VOTE_TYPE_HELPFUL {
		if err := checkUpvoteEligibility(ctx, s.db, u.UserUUID); err != nil {
			return nil, err
		}
	}
	// If casting an unhelpful vote, enforce downvote unlock threshold (100 rep) and rate limits (3/24h global, 3/week per author)
	if req.Msg.GetVoteType() == pb.ReviewVoteType_REVIEW_VOTE_TYPE_UNHELPFUL {
		if err := checkDownvoteEligibility(ctx, s.db, u.UserUUID, authorUUID); err != nil {
			return nil, err
		}
	}

	var existingIsHelpful sql.NullBool
	var existingCreatedAt sql.NullTime
	_ = s.db.QueryRow(ctx, "SELECT is_helpful, created_at FROM review_votes WHERE review_uuid = $1 AND user_uuid = $2", revUUID, u.UserUUID).Scan(&existingIsHelpful, &existingCreatedAt)

	// Reverse previous vote reputation if existed
	if existingIsHelpful.Valid {
		if existingIsHelpful.Bool {
			_ = adjustReputation(ctx, s.db, authorUUID, -1)
		} else {
			_ = adjustReputation(ctx, s.db, authorUUID, +2)
			if existingCreatedAt.Valid && time.Since(existingCreatedAt.Time) <= 24*time.Hour {
				_ = adjustReputation(ctx, s.db, u.UserUUID, +2)
			}
		}
	}

	switch req.Msg.GetVoteType() {
	case pb.ReviewVoteType_REVIEW_VOTE_TYPE_UNSPECIFIED:
		// Retract vote
		_, _ = s.db.Exec(ctx, "DELETE FROM review_votes WHERE review_uuid = $1 AND user_uuid = $2", revUUID, u.UserUUID)

	case pb.ReviewVoteType_REVIEW_VOTE_TYPE_HELPFUL:
		voteUUID := authcontext.NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO review_votes (uuid, review_uuid, user_uuid, is_helpful, created_at)
			VALUES ($1, $2, $3, true, NOW())
			ON CONFLICT (user_uuid, review_uuid) DO UPDATE SET is_helpful = true, created_at = NOW()
		`, voteUUID, revUUID, u.UserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = adjustReputation(ctx, s.db, authorUUID, +1)

	case pb.ReviewVoteType_REVIEW_VOTE_TYPE_UNHELPFUL:
		voteUUID := authcontext.NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO review_votes (uuid, review_uuid, user_uuid, is_helpful, created_at)
			VALUES ($1, $2, $3, false, NOW())
			ON CONFLICT (user_uuid, review_uuid) DO UPDATE SET is_helpful = false, created_at = NOW()
		`, voteUUID, revUUID, u.UserUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = adjustReputation(ctx, s.db, authorUUID, -2)
		_ = adjustReputation(ctx, s.db, u.UserUUID, -2) // Downvote costs voter -2 rep
	}

	var helpful, unhelpful int32
	_ = s.db.QueryRow(ctx, `
		SELECT
			COALESCE(count(*) FILTER (WHERE is_helpful = true), 0),
			COALESCE(count(*) FILTER (WHERE is_helpful = false), 0)
		FROM review_votes
		WHERE review_uuid = $1
	`, revUUID).Scan(&helpful, &unhelpful)

	return connect.NewResponse(&pb.VoteReviewResponse{
		HelpfulCount:    helpful,
		UnhelpfulCount:  unhelpful,
		CurrentUserVote: req.Msg.GetVoteType(),
	}), nil
}

func (s *SocialService) CreateReviewComment(ctx context.Context, req *connect.Request[pb.CreateReviewCommentRequest]) (*connect.Response[pb.ReviewComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("comment body cannot be empty"))
	}

	var revUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM reviews WHERE id = $1 OR uuid::text = $1", req.Msg.GetReviewId()).Scan(&revUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	commentUUID := authcontext.NewUUIDv7()
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)
	_, err = s.db.Exec(ctx, `
		INSERT INTO review_comments (uuid, review_uuid, user_uuid, body, author_username, author_avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, commentUUID, revUUID, u.UserUUID, body, username, avatar)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ReviewComment{
		Uuid:            commentUUID,
		ReviewUuid:      revUUID,
		UserUuid:        u.UserUUID,
		Body:            body,
		CreatedAt:       timestamppb.Now(),
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteReviewComment(ctx context.Context, req *connect.Request[pb.DeleteReviewCommentRequest]) (*connect.Response[pb.ReviewComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var commentUUID, userUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM review_comments WHERE uuid::text = $1", req.Msg.GetCommentUuid()).Scan(&commentUUID, &userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("comment not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's comment"))
	}

	_, err = s.db.Exec(ctx, "UPDATE review_comments SET deleted_at = NOW(), body = '[Comment deleted]' WHERE uuid = $1", commentUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ReviewComment{
		Uuid:      commentUUID,
		Body:      "[Comment deleted]",
		DeletedAt: timestamppb.Now(),
	}), nil
}

func (s *SocialService) ListReviewComments(ctx context.Context, req *connect.Request[pb.ListReviewCommentsRequest]) (*connect.Response[pb.ListReviewCommentsResponse], error) {
	var revUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM reviews WHERE id = $1 OR uuid::text = $1", req.Msg.GetReviewId()).Scan(&revUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	rows, err := s.db.Query(ctx, `
		SELECT
			c.uuid, c.review_uuid, c.user_uuid, c.body, c.deleted_at, c.created_at, c.updated_at,
			CASE WHEN c.deleted_at IS NOT NULL THEN '[Deleted User]' ELSE u.id END as author_username,
			CASE WHEN c.deleted_at IS NOT NULL THEN '' ELSE COALESCE(p.avatar_url, '') END as author_avatar_url
		FROM review_comments c
		JOIN users u ON u.uuid = c.user_uuid
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		WHERE c.review_uuid = $1
		ORDER BY c.created_at ASC
	`, revUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var comments []*pb.ReviewComment
	for rows.Next() {
		var c pb.ReviewComment
		var delAt *time.Time
		var createdAt, updatedAt time.Time
		err := rows.Scan(&c.Uuid, &c.ReviewUuid, &c.UserUuid, &c.Body, &delAt, &createdAt, &updatedAt, &c.AuthorUsername, &c.AuthorAvatarUrl)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		c.CreatedAt = timestamppb.New(createdAt)
		c.UpdatedAt = timestamppb.New(updatedAt)
		if delAt != nil {
			c.DeletedAt = timestamppb.New(*delAt)
		}
		comments = append(comments, &c)
	}

	return connect.NewResponse(&pb.ListReviewCommentsResponse{Comments: comments}), nil
}

// ---------------------------------------------------------
// Technical Q&A RPCs (StackOverflow-Style)
// ---------------------------------------------------------

func (s *SocialService) CreateQuestion(ctx context.Context, req *connect.Request[pb.CreateQuestionRequest]) (*connect.Response[pb.Question], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	entityUUID, resolvedCol, _, _, _, isArchived, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	if isArchived {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot post a question on an archived entity"))
	}

	// Self-Question Block: Users cannot post questions on their own builds
	if resolvedCol == "build" || resolvedCol == "builds" {
		var authorUUID sql.NullString
		err := s.db.QueryRow(ctx, "SELECT user_uuid FROM builds WHERE uuid = $1", entityUUID).Scan(&authorUUID)
		if err == nil && authorUUID.Valid && authorUUID.String == u.UserUUID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("authors cannot ask questions on their own builds"))
		}
	}

	title := strings.TrimSpace(req.Msg.GetTitle())
	body := strings.TrimSpace(req.Msg.GetBody())
	if title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("question title cannot be empty"))
	}

	qUUID := authcontext.NewUUIDv7()
	qID := "q-" + qUUID[:8]
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)

	_, err = s.db.Exec(ctx, `
		INSERT INTO questions (
			uuid, id, user_uuid, entity_uuid, title, body, net_score, is_edited,
			author_username, author_avatar_url, upvote_count, downvote_count,
			answer_count, has_accepted_answer, current_user_vote,
			collection, entity_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, false, $7, $8, 0, 0, 0, false, 0, $9, $10, NOW(), NOW())
	`, qUUID, qID, u.UserUUID, entityUUID, title, body, username, avatar, resolvedCol, req.Msg.GetEntityId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.Question{
		Uuid:              qUUID,
		Id:                qID,
		UserUuid:          u.UserUUID,
		EntityUuid:        entityUUID,
		Collection:        resolvedCol,
		EntityId:          req.Msg.GetEntityId(),
		Title:             title,
		Body:              body,
		NetScore:          0,
		IsEdited:          false,
		CreatedAt:         timestamppb.Now(),
		UpdatedAt:         timestamppb.Now(),
		AuthorUsername:    username,
		AuthorAvatarUrl:   avatar,
		UpvoteCount:       0,
		DownvoteCount:     0,
		AnswerCount:       0,
		HasAcceptedAnswer: false,
		CurrentUserVote:   0,
	}), nil
}

func (s *SocialService) UpdateQuestion(ctx context.Context, req *connect.Request[pb.UpdateQuestionRequest]) (*connect.Response[pb.Question], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	qID := strings.TrimSpace(req.Msg.GetQuestionId())
	title := strings.TrimSpace(req.Msg.GetTitle())
	body := strings.TrimSpace(req.Msg.GetBody())
	if title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title cannot be empty"))
	}

	var qUUID, authorUUID, entityUUID, col, entityID string
	var createdAt time.Time
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid, entity_uuid, collection, entity_id, created_at FROM questions WHERE id = $1 OR uuid::text = $1", qID).Scan(&qUUID, &authorUUID, &entityUUID, &col, &entityID, &createdAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	if authorUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot edit another user's question"))
	}

	_, err = s.db.Exec(ctx, `
		UPDATE questions
		SET title = $1, body = $2, is_edited = true, updated_at = NOW()
		WHERE uuid = $3
	`, title, body, qUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	username, avatar := getUserDisplay(ctx, s.db, authorUUID)

	return connect.NewResponse(&pb.Question{
		Uuid:            qUUID,
		Id:              qID,
		UserUuid:        authorUUID,
		EntityUuid:      entityUUID,
		Collection:      col,
		EntityId:        entityID,
		Title:           title,
		Body:            body,
		IsEdited:        true,
		CreatedAt:       timestamppb.New(createdAt),
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteQuestion(ctx context.Context, req *connect.Request[pb.DeleteQuestionRequest]) (*connect.Response[pb.Question], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	qID := strings.TrimSpace(req.Msg.GetQuestionId())
	var qUUID, authorUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM questions WHERE id = $1 OR uuid::text = $1", qID).Scan(&qUUID, &authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	if authorUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's question"))
	}

	// Knowledge preservation rule: check if question has accepted answers or upvoted answers
	var hasAcceptedOrUpvoted bool
	_ = s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM answers
			WHERE question_uuid = $1 AND (is_accepted_by_asker = true OR net_score >= 1) AND deleted_at IS NULL
		)
	`, qUUID).Scan(&hasAcceptedOrUpvoted)

	if hasAcceptedOrUpvoted {
		// Mask user and preserve knowledge thread for community
		_, err = s.db.Exec(ctx, "UPDATE questions SET deleted_at = NOW() WHERE uuid = $1", qUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else {
		// Hard delete when no answers exist
		_, err = s.db.Exec(ctx, "DELETE FROM questions WHERE uuid = $1", qUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&pb.Question{
		Uuid:      qUUID,
		DeletedAt: timestamppb.Now(),
	}), nil
}

func (s *SocialService) ListQuestions(ctx context.Context, req *connect.Request[pb.ListQuestionsRequest]) (*connect.Response[pb.ListQuestionsResponse], error) {
	entityUUID, _, _, _, _, _, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	currentCallerUUID := ""
	if u := authcontext.UserContextFromContext(ctx); u != nil {
		currentCallerUUID = u.UserUUID
	}

	orderClause := "ORDER BY q.net_score DESC, q.created_at DESC"
	if req.Msg.GetSortBy() == "recent" {
		orderClause = "ORDER BY q.created_at DESC"
	}

	searchClause := ""
	if qSearch := strings.TrimSpace(req.Msg.GetQuery()); qSearch != "" {
		searchClause = fmt.Sprintf("AND (q.title ILIKE '%%%s%%' OR q.body ILIKE '%%%s%%')", qSearch, qSearch)
	}

	query := fmt.Sprintf(`
		SELECT
			q.uuid, q.id, q.user_uuid, q.entity_uuid, q.collection, q.entity_id,
			q.title, q.body, q.net_score, q.is_edited, q.deleted_at, q.created_at, q.updated_at,
			CASE WHEN q.deleted_at IS NOT NULL THEN '[Deleted User]' ELSE u.id END as author_username,
			CASE WHEN q.deleted_at IS NOT NULL THEN '' ELSE COALESCE(p.avatar_url, '') END as author_avatar_url,
			COALESCE(ans.ans_count, 0) as answer_count,
			COALESCE(ans.has_accepted, false) as has_accepted_answer,
			COALESCE(MAX(CASE WHEN v.user_uuid = $2 THEN (CASE WHEN v.is_upvote THEN 1 ELSE -1 END) ELSE 0 END), 0) as user_vote,
			COALESCE(SUM(CASE WHEN v.is_upvote = true THEN 1 ELSE 0 END), 0) as upvotes,
			COALESCE(SUM(CASE WHEN v.is_upvote = false THEN 1 ELSE 0 END), 0) as downvotes
		FROM questions q
		JOIN users u ON u.uuid = q.user_uuid
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		LEFT JOIN question_votes v ON v.question_uuid = q.uuid
		LEFT JOIN (
			SELECT question_uuid, count(*) as ans_count, bool_or(is_accepted_by_asker) as has_accepted
			FROM answers
			WHERE deleted_at IS NULL
			GROUP BY question_uuid
		) ans ON ans.question_uuid = q.uuid
		WHERE q.entity_uuid = $1 %s
		GROUP BY q.uuid, q.id, q.user_uuid, q.entity_uuid, q.collection, q.entity_id, q.title, q.body, q.net_score, q.is_edited, q.deleted_at, q.created_at, q.updated_at, u.id, p.avatar_url, ans.ans_count, ans.has_accepted
		%s
		LIMIT 50
	`, searchClause, orderClause)

	rows, err := s.db.Query(ctx, query, entityUUID, currentCallerUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var questions []*pb.Question
	for rows.Next() {
		var q pb.Question
		var delAt *time.Time
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&q.Uuid, &q.Id, &q.UserUuid, &q.EntityUuid, &q.Collection, &q.EntityId,
			&q.Title, &q.Body, &q.NetScore, &q.IsEdited, &delAt, &createdAt, &updatedAt,
			&q.AuthorUsername, &q.AuthorAvatarUrl, &q.AnswerCount, &q.HasAcceptedAnswer, &q.CurrentUserVote,
			&q.UpvoteCount, &q.DownvoteCount,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		q.CreatedAt = timestamppb.New(createdAt)
		q.UpdatedAt = timestamppb.New(updatedAt)
		if delAt != nil {
			q.DeletedAt = timestamppb.New(*delAt)
		}
		questions = append(questions, &q)
	}

	return connect.NewResponse(&pb.ListQuestionsResponse{Questions: questions}), nil
}

func (s *SocialService) VoteQuestion(ctx context.Context, req *connect.Request[pb.VoteQuestionRequest]) (*connect.Response[pb.VoteQuestionResponse], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	qID := strings.TrimSpace(req.Msg.GetQuestionId())
	var qUUID, authorUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM questions WHERE id = $1 OR uuid::text = $1", qID).Scan(&qUUID, &authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	if authorUUID == u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot vote on your own question"))
	}

	vote := req.Msg.GetVote() // +1, -1, or 0
	if vote > 0 {
		if err := checkUpvoteEligibility(ctx, s.db, u.UserUUID); err != nil {
			return nil, err
		}
	} else if vote < 0 {
		if err := checkDownvoteEligibility(ctx, s.db, u.UserUUID, authorUUID); err != nil {
			return nil, err
		}
	}

	var existingIsUpvote sql.NullBool
	var existingCreatedAt sql.NullTime
	_ = s.db.QueryRow(ctx, "SELECT is_upvote, created_at FROM question_votes WHERE question_uuid = $1 AND user_uuid = $2", qUUID, u.UserUUID).Scan(&existingIsUpvote, &existingCreatedAt)

	// Revert previous vote impact
	if existingIsUpvote.Valid {
		if existingIsUpvote.Bool {
			_ = adjustReputation(ctx, s.db, authorUUID, -1)
		} else {
			_ = adjustReputation(ctx, s.db, authorUUID, +2)
			if existingCreatedAt.Valid && time.Since(existingCreatedAt.Time) <= 24*time.Hour {
				_ = adjustReputation(ctx, s.db, u.UserUUID, +2)
			}
		}
	}

	if vote == 0 {
		_, _ = s.db.Exec(ctx, "DELETE FROM question_votes WHERE question_uuid = $1 AND user_uuid = $2", qUUID, u.UserUUID)
	} else {
		isUp := vote > 0
		vUUID := authcontext.NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO question_votes (uuid, question_uuid, user_uuid, is_upvote, created_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_uuid, question_uuid) DO UPDATE SET is_upvote = $4, created_at = NOW()
		`, vUUID, qUUID, u.UserUUID, isUp)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if isUp {
			_ = adjustReputation(ctx, s.db, authorUUID, +1)
		} else {
			_ = adjustReputation(ctx, s.db, authorUUID, -2)
			_ = adjustReputation(ctx, s.db, u.UserUUID, -2)
		}
	}

	// Update denormalized net_score
	_, _ = s.db.Exec(ctx, `
		UPDATE questions
		SET net_score = (
			SELECT COALESCE(SUM(CASE WHEN is_upvote THEN 1 ELSE -1 END), 0)
			FROM question_votes
			WHERE question_uuid = $1
		)
		WHERE uuid = $1
	`, qUUID)

	var netScore, upvotes, downvotes int32
	_ = s.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN is_upvote THEN 1 ELSE -1 END), 0),
			COALESCE(count(*) FILTER (WHERE is_upvote = true), 0),
			COALESCE(count(*) FILTER (WHERE is_upvote = false), 0)
		FROM question_votes
		WHERE question_uuid = $1
	`, qUUID).Scan(&netScore, &upvotes, &downvotes)

	return connect.NewResponse(&pb.VoteQuestionResponse{
		NetScore:        netScore,
		UpvoteCount:     upvotes,
		DownvoteCount:   downvotes,
		CurrentUserVote: vote,
	}), nil
}

func (s *SocialService) CreateAnswer(ctx context.Context, req *connect.Request[pb.CreateAnswerRequest]) (*connect.Response[pb.Answer], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("answer body cannot be empty"))
	}

	var qUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM questions WHERE id = $1 OR uuid::text = $1", req.Msg.GetQuestionId()).Scan(&qUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	ansUUID := authcontext.NewUUIDv7()
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)
	_, err = s.db.Exec(ctx, `
		INSERT INTO answers (
			uuid, question_uuid, user_uuid, body, net_score,
			is_accepted_by_asker, is_author_endorsed, is_edited,
			author_username, author_avatar_url, upvote_count,
			downvote_count, current_user_vote, comment_count,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 0, false, false, false, $5, $6, 0, 0, 0, 0, NOW(), NOW())
	`, ansUUID, qUUID, u.UserUUID, body, username, avatar)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.Answer{
		Uuid:              ansUUID,
		QuestionUuid:      qUUID,
		UserUuid:          u.UserUUID,
		Body:              body,
		NetScore:          0,
		IsAcceptedByAsker: false,
		IsAuthorEndorsed:  false,
		IsEdited:          false,
		CreatedAt:         timestamppb.Now(),
		UpdatedAt:         timestamppb.Now(),
		AuthorUsername:    username,
		AuthorAvatarUrl:   avatar,
		UpvoteCount:       0,
		DownvoteCount:     0,
		CurrentUserVote:   0,
		CommentCount:      0,
	}), nil
}

func (s *SocialService) UpdateAnswer(ctx context.Context, req *connect.Request[pb.UpdateAnswerRequest]) (*connect.Response[pb.Answer], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("answer body cannot be empty"))
	}

	var authorUUID string
	err := s.db.QueryRow(ctx, "SELECT user_uuid FROM answers WHERE uuid::text = $1", req.Msg.GetAnswerUuid()).Scan(&authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	if authorUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot edit another user's answer"))
	}

	_, err = s.db.Exec(ctx, "UPDATE answers SET body = $1, is_edited = true, updated_at = NOW() WHERE uuid::text = $2", body, req.Msg.GetAnswerUuid())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	username, avatar := getUserDisplay(ctx, s.db, authorUUID)

	return connect.NewResponse(&pb.Answer{
		Uuid:            req.Msg.GetAnswerUuid(),
		Body:            body,
		IsEdited:        true,
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteAnswer(ctx context.Context, req *connect.Request[pb.DeleteAnswerRequest]) (*connect.Response[pb.Answer], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var authorUUID string
	err := s.db.QueryRow(ctx, "SELECT user_uuid FROM answers WHERE uuid::text = $1", req.Msg.GetAnswerUuid()).Scan(&authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	if authorUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's answer"))
	}

	_, err = s.db.Exec(ctx, "DELETE FROM answers WHERE uuid::text = $1", req.Msg.GetAnswerUuid())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.Answer{
		Uuid:      req.Msg.GetAnswerUuid(),
		DeletedAt: timestamppb.Now(),
	}), nil
}

func (s *SocialService) ListAnswers(ctx context.Context, req *connect.Request[pb.ListAnswersRequest]) (*connect.Response[pb.ListAnswersResponse], error) {
	var qUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM questions WHERE id = $1 OR uuid::text = $1", req.Msg.GetQuestionId()).Scan(&qUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	currentCallerUUID := ""
	if u := authcontext.UserContextFromContext(ctx); u != nil {
		currentCallerUUID = u.UserUUID
	}

	rows, err := s.db.Query(ctx, `
		SELECT
			a.uuid, a.question_uuid, a.user_uuid, a.body, a.net_score, a.is_accepted_by_asker, a.is_author_endorsed,
			a.is_edited, a.deleted_at, a.created_at, a.updated_at,
			CASE WHEN a.deleted_at IS NOT NULL THEN '[Deleted User]' ELSE u.id END as author_username,
			CASE WHEN a.deleted_at IS NOT NULL THEN '' ELSE COALESCE(p.avatar_url, '') END as author_avatar_url,
			COALESCE(SUM(CASE WHEN v.is_upvote = true THEN 1 ELSE 0 END), 0) as upvotes,
			COALESCE(SUM(CASE WHEN v.is_upvote = false THEN 1 ELSE 0 END), 0) as downvotes,
			COALESCE(MAX(CASE WHEN v.user_uuid = $2 THEN (CASE WHEN v.is_upvote THEN 1 ELSE -1 END) ELSE 0 END), 0) as user_vote,
			COALESCE(comm.comment_count, 0) as comment_count
		FROM answers a
		JOIN users u ON u.uuid = a.user_uuid
		LEFT JOIN user_profiles p ON p.user_uuid = u.uuid
		LEFT JOIN answer_votes v ON v.answer_uuid = a.uuid
		LEFT JOIN (
			SELECT answer_uuid, count(*) as comment_count
			FROM answer_comments
			WHERE deleted_at IS NULL
			GROUP BY answer_uuid
		) comm ON comm.answer_uuid = a.uuid
		WHERE a.question_uuid = $1 AND a.deleted_at IS NULL
		GROUP BY a.uuid, a.question_uuid, a.user_uuid, a.body, a.net_score, a.is_accepted_by_asker, a.is_author_endorsed, a.is_edited, a.deleted_at, a.created_at, a.updated_at, u.id, p.avatar_url, comm.comment_count
		ORDER BY a.is_accepted_by_asker DESC, a.is_author_endorsed DESC, a.net_score DESC, a.created_at ASC
	`, qUUID, currentCallerUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var answers []*pb.Answer
	for rows.Next() {
		var a pb.Answer
		var delAt *time.Time
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&a.Uuid, &a.QuestionUuid, &a.UserUuid, &a.Body, &a.NetScore, &a.IsAcceptedByAsker, &a.IsAuthorEndorsed,
			&a.IsEdited, &delAt, &createdAt, &updatedAt, &a.AuthorUsername, &a.AuthorAvatarUrl,
			&a.UpvoteCount, &a.DownvoteCount, &a.CurrentUserVote, &a.CommentCount,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		a.CreatedAt = timestamppb.New(createdAt)
		a.UpdatedAt = timestamppb.New(updatedAt)
		if delAt != nil {
			a.DeletedAt = timestamppb.New(*delAt)
		}
		answers = append(answers, &a)
	}

	return connect.NewResponse(&pb.ListAnswersResponse{Answers: answers}), nil
}

func (s *SocialService) VoteAnswer(ctx context.Context, req *connect.Request[pb.VoteAnswerRequest]) (*connect.Response[pb.VoteAnswerResponse], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	var ansUUID, authorUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM answers WHERE uuid::text = $1", req.Msg.GetAnswerUuid()).Scan(&ansUUID, &authorUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	if authorUUID == u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot vote on your own answer"))
	}

	vote := req.Msg.GetVote()
	if vote > 0 {
		if err := checkUpvoteEligibility(ctx, s.db, u.UserUUID); err != nil {
			return nil, err
		}
	} else if vote < 0 {
		if err := checkDownvoteEligibility(ctx, s.db, u.UserUUID, authorUUID); err != nil {
			return nil, err
		}
	}

	var existingIsUpvote sql.NullBool
	var existingCreatedAt sql.NullTime
	_ = s.db.QueryRow(ctx, "SELECT is_upvote, created_at FROM answer_votes WHERE answer_uuid = $1 AND user_uuid = $2", ansUUID, u.UserUUID).Scan(&existingIsUpvote, &existingCreatedAt)

	if existingIsUpvote.Valid {
		if existingIsUpvote.Bool {
			_ = adjustReputation(ctx, s.db, authorUUID, -1)
		} else {
			_ = adjustReputation(ctx, s.db, authorUUID, +2)
			if existingCreatedAt.Valid && time.Since(existingCreatedAt.Time) <= 24*time.Hour {
				_ = adjustReputation(ctx, s.db, u.UserUUID, +2)
			}
		}
	}

	if vote == 0 {
		_, _ = s.db.Exec(ctx, "DELETE FROM answer_votes WHERE answer_uuid = $1 AND user_uuid = $2", ansUUID, u.UserUUID)
	} else {
		isUp := vote > 0
		vUUID := authcontext.NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO answer_votes (uuid, answer_uuid, user_uuid, is_upvote, created_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_uuid, answer_uuid) DO UPDATE SET is_upvote = $4, created_at = NOW()
		`, vUUID, ansUUID, u.UserUUID, isUp)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if isUp {
			_ = adjustReputation(ctx, s.db, authorUUID, +1)
		} else {
			_ = adjustReputation(ctx, s.db, authorUUID, -2)
			_ = adjustReputation(ctx, s.db, u.UserUUID, -2)
		}
	}

	_, _ = s.db.Exec(ctx, `
		UPDATE answers
		SET net_score = (
			SELECT COALESCE(SUM(CASE WHEN is_upvote THEN 1 ELSE -1 END), 0)
			FROM answer_votes
			WHERE answer_uuid = $1
		)
		WHERE uuid = $1
	`, ansUUID)

	var netScore, upvotes, downvotes int32
	_ = s.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN is_upvote THEN 1 ELSE -1 END), 0),
			COALESCE(count(*) FILTER (WHERE is_upvote = true), 0),
			COALESCE(count(*) FILTER (WHERE is_upvote = false), 0)
		FROM answer_votes
		WHERE answer_uuid = $1
	`, ansUUID).Scan(&netScore, &upvotes, &downvotes)

	return connect.NewResponse(&pb.VoteAnswerResponse{
		NetScore:        netScore,
		UpvoteCount:     upvotes,
		DownvoteCount:   downvotes,
		CurrentUserVote: vote,
	}), nil
}

func (s *SocialService) AcceptAnswer(ctx context.Context, req *connect.Request[pb.AcceptAnswerRequest]) (*connect.Response[pb.Answer], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	var ansUUID, qUUID, answererUUID string
	var currentlyAccepted bool
	var acceptedAt sql.NullTime
	err := s.db.QueryRow(ctx, `
		SELECT uuid, question_uuid, user_uuid, is_accepted_by_asker, accepted_at
		FROM answers
		WHERE uuid::text = $1
	`, req.Msg.GetAnswerUuid()).Scan(&ansUUID, &qUUID, &answererUUID, &currentlyAccepted, &acceptedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	// Dual-Authority: ONLY the Question Asker can confer "Accepted Answer" checkmark
	var askerUUID string
	err = s.db.QueryRow(ctx, "SELECT user_uuid FROM questions WHERE uuid = $1", qUUID).Scan(&askerUUID)
	if err != nil || askerUUID != u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("only the author of the question can accept an answer"))
	}

	// Self-Accept Block: Asker cannot accept their own answer
	if answererUUID == u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot accept your own answer"))
	}

	newStatus := !currentlyAccepted

	// Unset any previous accepted answer for this question
	var prevAcceptedAnswererUUID string
	var prevAcceptedAt sql.NullTime
	err = s.db.QueryRow(ctx, `
		UPDATE answers SET is_accepted_by_asker = false
		WHERE question_uuid = $1 AND is_accepted_by_asker = true AND uuid != $2
		RETURNING user_uuid, accepted_at
	`, qUUID, ansUUID).Scan(&prevAcceptedAnswererUUID, &prevAcceptedAt)
	if err == nil && prevAcceptedAnswererUUID != "" {
		// Rug-Pull protection: only claw back reputation if revoked within 24 hours
		if prevAcceptedAt.Valid && time.Since(prevAcceptedAt.Time) <= 24*time.Hour {
			_ = adjustReputation(ctx, s.db, prevAcceptedAnswererUUID, -30)
			_ = adjustReputation(ctx, s.db, askerUUID, -10)
		}
	}

	if newStatus {
		_, err = s.db.Exec(ctx, "UPDATE answers SET is_accepted_by_asker = true, accepted_at = NOW(), updated_at = NOW() WHERE uuid = $1", ansUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = adjustReputation(ctx, s.db, answererUUID, +30) // +30 rep to answerer
		_ = adjustReputation(ctx, s.db, askerUUID, +10)    // +10 rep bonus to asker
	} else {
		_, err = s.db.Exec(ctx, "UPDATE answers SET is_accepted_by_asker = false, updated_at = NOW() WHERE uuid = $1", ansUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// Rug-Pull protection: only claw back reputation if revoked within 24 hours
		if acceptedAt.Valid && time.Since(acceptedAt.Time) <= 24*time.Hour {
			_ = adjustReputation(ctx, s.db, answererUUID, -30)
			_ = adjustReputation(ctx, s.db, askerUUID, -10)
		}
	}

	username, avatar := getUserDisplay(ctx, s.db, answererUUID)

	return connect.NewResponse(&pb.Answer{
		Uuid:              ansUUID,
		QuestionUuid:      qUUID,
		UserUuid:          answererUUID,
		IsAcceptedByAsker: newStatus,
		AuthorUsername:    username,
		AuthorAvatarUrl:   avatar,
	}), nil
}

func (s *SocialService) EndorseAnswer(ctx context.Context, req *connect.Request[pb.EndorseAnswerRequest]) (*connect.Response[pb.Answer], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	var ansUUID, qUUID, answererUUID string
	var currentlyEndorsed bool
	var endorsedAt sql.NullTime
	err := s.db.QueryRow(ctx, `
		SELECT uuid, question_uuid, user_uuid, is_author_endorsed, endorsed_at
		FROM answers
		WHERE uuid::text = $1
	`, req.Msg.GetAnswerUuid()).Scan(&ansUUID, &qUUID, &answererUUID, &currentlyEndorsed, &endorsedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	// Dual-Authority: ONLY the Build Author can confer "Author Verified" star badge
	var buildAuthorUUID sql.NullString
	err = s.db.QueryRow(ctx, `
		SELECT b.user_uuid
		FROM questions q
		JOIN builds b ON b.uuid = q.entity_uuid
		WHERE q.uuid = $1
	`, qUUID).Scan(&buildAuthorUUID)

	if err != nil || !buildAuthorUUID.Valid || (buildAuthorUUID.String != u.UserUUID && u.Role != "ADMIN") {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("only the build author (or admin) can endorse an answer"))
	}

	// Self-Endorse Block: Build author cannot endorse their own answers
	if answererUUID == u.UserUUID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("build authors cannot endorse their own answers"))
	}

	newStatus := !currentlyEndorsed

	// Unset previous endorsement on this question
	var prevEndorsedAnswerer string
	var prevEndorsedAt sql.NullTime
	err = s.db.QueryRow(ctx, `
		UPDATE answers SET is_author_endorsed = false
		WHERE question_uuid = $1 AND is_author_endorsed = true AND uuid != $2
		RETURNING user_uuid, endorsed_at
	`, qUUID, ansUUID).Scan(&prevEndorsedAnswerer, &prevEndorsedAt)
	if err == nil && prevEndorsedAnswerer != "" {
		if prevEndorsedAt.Valid && time.Since(prevEndorsedAt.Time) <= 24*time.Hour {
			_ = adjustReputation(ctx, s.db, prevEndorsedAnswerer, -20)
		}
	}

	if newStatus {
		_, err = s.db.Exec(ctx, "UPDATE answers SET is_author_endorsed = true, endorsed_at = NOW(), updated_at = NOW() WHERE uuid = $1", ansUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = adjustReputation(ctx, s.db, answererUUID, +20) // +20 rep for author endorsement
	} else {
		_, err = s.db.Exec(ctx, "UPDATE answers SET is_author_endorsed = false, updated_at = NOW() WHERE uuid = $1", ansUUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if endorsedAt.Valid && time.Since(endorsedAt.Time) <= 24*time.Hour {
			_ = adjustReputation(ctx, s.db, answererUUID, -20)
		}
	}

	username, avatar := getUserDisplay(ctx, s.db, answererUUID)

	return connect.NewResponse(&pb.Answer{
		Uuid:             ansUUID,
		QuestionUuid:     qUUID,
		UserUuid:         answererUUID,
		IsAuthorEndorsed: newStatus,
		AuthorUsername:   username,
		AuthorAvatarUrl:  avatar,
	}), nil
}

func (s *SocialService) CreateQuestionComment(ctx context.Context, req *connect.Request[pb.CreateQuestionCommentRequest]) (*connect.Response[pb.QuestionComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("comment body cannot be empty"))
	}

	var qUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM questions WHERE id = $1 OR uuid::text = $1", req.Msg.GetQuestionId()).Scan(&qUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("question not found"))
	}

	commentUUID := authcontext.NewUUIDv7()
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)
	_, err = s.db.Exec(ctx, `
		INSERT INTO question_comments (uuid, question_uuid, user_uuid, body, author_username, author_avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, commentUUID, qUUID, u.UserUUID, body, username, avatar)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.QuestionComment{
		Uuid:            commentUUID,
		QuestionUuid:    qUUID,
		UserUuid:        u.UserUUID,
		Body:            body,
		CreatedAt:       timestamppb.Now(),
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteQuestionComment(ctx context.Context, req *connect.Request[pb.DeleteQuestionCommentRequest]) (*connect.Response[pb.QuestionComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var userUUID string
	err := s.db.QueryRow(ctx, "SELECT user_uuid FROM question_comments WHERE uuid::text = $1", req.Msg.GetCommentUuid()).Scan(&userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("comment not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's comment"))
	}

	_, err = s.db.Exec(ctx, "UPDATE question_comments SET deleted_at = NOW(), body = '[Comment deleted]' WHERE uuid::text = $1", req.Msg.GetCommentUuid())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.QuestionComment{
		Uuid:      req.Msg.GetCommentUuid(),
		Body:      "[Comment deleted]",
		DeletedAt: timestamppb.Now(),
	}), nil
}

func (s *SocialService) CreateAnswerComment(ctx context.Context, req *connect.Request[pb.CreateAnswerCommentRequest]) (*connect.Response[pb.AnswerComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	body := strings.TrimSpace(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("comment body cannot be empty"))
	}

	var ansUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM answers WHERE uuid::text = $1", req.Msg.GetAnswerUuid()).Scan(&ansUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("answer not found"))
	}

	commentUUID := authcontext.NewUUIDv7()
	username, avatar := getUserDisplay(ctx, s.db, u.UserUUID)
	_, err = s.db.Exec(ctx, `
		INSERT INTO answer_comments (uuid, answer_uuid, user_uuid, body, author_username, author_avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, commentUUID, ansUUID, u.UserUUID, body, username, avatar)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.AnswerComment{
		Uuid:            commentUUID,
		AnswerUuid:      ansUUID,
		UserUuid:        u.UserUUID,
		Body:            body,
		CreatedAt:       timestamppb.Now(),
		UpdatedAt:       timestamppb.Now(),
		AuthorUsername:  username,
		AuthorAvatarUrl: avatar,
	}), nil
}

func (s *SocialService) DeleteAnswerComment(ctx context.Context, req *connect.Request[pb.DeleteAnswerCommentRequest]) (*connect.Response[pb.AnswerComment], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var userUUID string
	err := s.db.QueryRow(ctx, "SELECT user_uuid FROM answer_comments WHERE uuid::text = $1", req.Msg.GetCommentUuid()).Scan(&userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("comment not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's comment"))
	}

	_, err = s.db.Exec(ctx, "UPDATE answer_comments SET deleted_at = NOW(), body = '[Comment deleted]' WHERE uuid::text = $1", req.Msg.GetCommentUuid())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.AnswerComment{
		Uuid:      req.Msg.GetCommentUuid(),
		Body:      "[Comment deleted]",
		DeletedAt: timestamppb.Now(),
	}), nil
}

// ---------------------------------------------------------
// Favorites & Collections RPCs
// ---------------------------------------------------------

func (s *SocialService) getOrCreateDefaultLikedCollection(ctx context.Context, userUUID string) (string, error) {
	var colUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM collections WHERE user_uuid = $1 AND is_default_liked = true LIMIT 1", userUUID).Scan(&colUUID)
	if err == nil && colUUID != "" {
		return colUUID, nil
	}

	colUUID = authcontext.NewUUIDv7()
	colID := "liked-" + colUUID[:8]
	username, _ := getUserDisplay(ctx, s.db, userUUID)
	_, err = s.db.Exec(ctx, `
		INSERT INTO collections (uuid, id, user_uuid, name, description, is_public, is_default_liked, author_username, item_count, created_at, updated_at)
		VALUES ($1, $2, $3, 'Liked', 'Default collection of liked builds and components', true, true, $4, 0, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, colUUID, colID, userUUID, username)
	if err != nil {
		return "", err
	}
	return colUUID, nil
}

func (s *SocialService) ToggleLikeItem(ctx context.Context, req *connect.Request[pb.ToggleLikeItemRequest]) (*connect.Response[pb.ToggleLikeItemResponse], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	entityUUID, resolvedCol, name, image, authorOrMfg, _, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	likedColUUID, err := s.getOrCreateDefaultLikedCollection(ctx, u.UserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var existingItemUUID string
	_ = s.db.QueryRow(ctx, "SELECT uuid FROM collection_items WHERE collection_uuid = $1 AND entity_uuid = $2", likedColUUID, entityUUID).Scan(&existingItemUUID)

	isLiked := false
	if existingItemUUID != "" {
		// Unlike
		_, _ = s.db.Exec(ctx, "DELETE FROM collection_items WHERE uuid = $1", existingItemUUID)
		isLiked = false
	} else {
		// Like
		newItemUUID := authcontext.NewUUIDv7()
		_, err = s.db.Exec(ctx, `
			INSERT INTO collection_items (uuid, collection_uuid, entity_uuid, entity_id, collection, name, primary_display_image, author_or_manufacturer, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, newItemUUID, likedColUUID, entityUUID, req.Msg.GetEntityId(), resolvedCol, name, image, authorOrMfg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		isLiked = true
	}

	// Count total likes across all users' "Liked" default collections
	var likeCount int32
	_ = s.db.QueryRow(ctx, `
		SELECT count(*)
		FROM collection_items ci
		JOIN collections c ON c.uuid = ci.collection_uuid
		WHERE c.is_default_liked = true AND ci.entity_uuid = $1
	`, entityUUID).Scan(&likeCount)

	return connect.NewResponse(&pb.ToggleLikeItemResponse{
		IsLiked:   isLiked,
		LikeCount: likeCount,
	}), nil
}

func (s *SocialService) CreateCollection(ctx context.Context, req *connect.Request[pb.CreateCollectionRequest]) (*connect.Response[pb.Collection], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("collection name cannot be empty"))
	}

	colUUID := authcontext.NewUUIDv7()
	slug := slugify(name)
	if slug == "" {
		slug = "collection"
	}
	colID := fmt.Sprintf("%s-%s", slug, colUUID[:6])
	username, _ := getUserDisplay(ctx, s.db, u.UserUUID)

	_, err := s.db.Exec(ctx, `
		INSERT INTO collections (uuid, id, user_uuid, name, description, is_public, is_default_liked, author_username, item_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, 0, NOW(), NOW())
	`, colUUID, colID, u.UserUUID, name, req.Msg.GetDescription(), req.Msg.GetIsPublic(), username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.Collection{
		Uuid:           colUUID,
		Id:             colID,
		UserUuid:       u.UserUUID,
		AuthorUsername: username,
		Name:           name,
		Description:    req.Msg.GetDescription(),
		IsPublic:       req.Msg.GetIsPublic(),
		IsDefaultLiked: false,
		ItemCount:      0,
		CreatedAt:      timestamppb.Now(),
		UpdatedAt:      timestamppb.Now(),
	}), nil
}

func (s *SocialService) UpdateCollection(ctx context.Context, req *connect.Request[pb.UpdateCollectionRequest]) (*connect.Response[pb.Collection], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	var colUUID, colID, userUUID, currentName, currentDesc string
	var isDefault bool
	err := s.db.QueryRow(ctx, "SELECT uuid, id, user_uuid, name, description, is_default_liked FROM collections WHERE id = $1 OR uuid::text = $1", req.Msg.GetCollectionId()).Scan(&colUUID, &colID, &userUUID, &currentName, &currentDesc, &isDefault)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot update another user's collection"))
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		name = currentName
	}
	if isDefault {
		name = "Liked"
	}
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("collection name cannot be empty"))
	}

	desc := req.Msg.GetDescription()
	if req.Msg.GetName() == "" && desc == "" && currentDesc != "" {
		desc = currentDesc
	}

	_, err = s.db.Exec(ctx, `
		UPDATE collections
		SET name = $1, description = $2, is_public = $3, updated_at = NOW()
		WHERE uuid = $4
	`, name, desc, req.Msg.GetIsPublic(), colUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	username, _ := getUserDisplay(ctx, s.db, userUUID)

	return connect.NewResponse(&pb.Collection{
		Uuid:           colUUID,
		Id:             colID,
		UserUuid:       userUUID,
		AuthorUsername: username,
		Name:           name,
		Description:    desc,
		IsPublic:       req.Msg.GetIsPublic(),
		IsDefaultLiked: isDefault,
		UpdatedAt:      timestamppb.Now(),
	}), nil
}

func (s *SocialService) DeleteCollection(ctx context.Context, req *connect.Request[pb.DeleteCollectionRequest]) (*connect.Response[pb.Collection], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var colUUID, userUUID string
	var isDefault bool
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid, is_default_liked FROM collections WHERE id = $1 OR uuid::text = $1", req.Msg.GetCollectionId()).Scan(&colUUID, &userUUID, &isDefault)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot delete another user's collection"))
	}

	if isDefault {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("the system Liked collection cannot be deleted"))
	}

	_, err = s.db.Exec(ctx, "DELETE FROM collections WHERE uuid = $1", colUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.Collection{Uuid: colUUID}), nil
}

func (s *SocialService) GetCollection(ctx context.Context, req *connect.Request[pb.GetCollectionRequest]) (*connect.Response[pb.Collection], error) {
	var c pb.Collection
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT
			c.uuid, c.id, c.user_uuid, c.name, c.description, c.is_public, c.is_default_liked,
			c.created_at, c.updated_at, u.id as author_username,
			(SELECT count(*) FROM collection_items WHERE collection_uuid = c.uuid) as item_count
		FROM collections c
		JOIN users u ON u.uuid = c.user_uuid
		WHERE c.id = $1 OR c.uuid::text = $1
	`, req.Msg.GetId()).Scan(
		&c.Uuid, &c.Id, &c.UserUuid, &c.Name, &c.Description, &c.IsPublic, &c.IsDefaultLiked,
		&createdAt, &updatedAt, &c.AuthorUsername, &c.ItemCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Privacy check
	caller := authcontext.UserContextFromContext(ctx)
	if !c.IsPublic && (caller == nil || (caller.UserUUID != c.UserUuid && caller.Role != "ADMIN")) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this collection is private"))
	}

	c.CreatedAt = timestamppb.New(createdAt)
	c.UpdatedAt = timestamppb.New(updatedAt)

	return connect.NewResponse(&c), nil
}

func (s *SocialService) ListUserCollections(ctx context.Context, req *connect.Request[pb.ListUserCollectionsRequest]) (*connect.Response[pb.ListUserCollectionsResponse], error) {
	targetUserID := strings.TrimSpace(req.Msg.GetUserId())
	var targetUserUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid FROM users WHERE id = $1 OR uuid::text = $1", targetUserID).Scan(&targetUserUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	caller := authcontext.UserContextFromContext(ctx)
	isOwner := caller != nil && caller.UserUUID == targetUserUUID

	// If owner, list all collections; otherwise, only public ones
	privacyFilter := "AND c.is_public = true"
	if isOwner {
		privacyFilter = ""
	}

	var filterEntityUUID string
	if req.Msg.ContainsEntityId != nil && strings.TrimSpace(*req.Msg.ContainsEntityId) != "" {
		entityUUID, _, _, _, _, _, err := resolveEntity(ctx, s.db, *req.Msg.ContainsEntityId, req.Msg.ContainsCollection)
		if err == nil {
			filterEntityUUID = entityUUID
		}
	}

	var args []any
	args = append(args, targetUserUUID)

	containsItemSelect := "false as contains_item"
	if filterEntityUUID != "" {
		args = append(args, filterEntityUUID)
		containsItemSelect = fmt.Sprintf(`EXISTS (
			SELECT 1 FROM collection_items ci
			WHERE ci.collection_uuid = c.uuid AND ci.entity_uuid = $%d
		) as contains_item`, len(args))
	}

	query := fmt.Sprintf(`
		SELECT
			c.uuid, c.id, c.user_uuid, c.name, c.description, c.is_public, c.is_default_liked,
			c.created_at, c.updated_at, u.id as author_username,
			(SELECT count(*) FROM collection_items WHERE collection_uuid = c.uuid) as item_count,
			%s
		FROM collections c
		JOIN users u ON u.uuid = c.user_uuid
		WHERE c.user_uuid = $1 %s
		ORDER BY c.is_default_liked DESC, c.created_at DESC
	`, containsItemSelect, privacyFilter)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var collections []*pb.Collection
	for rows.Next() {
		var c pb.Collection
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&c.Uuid, &c.Id, &c.UserUuid, &c.Name, &c.Description, &c.IsPublic, &c.IsDefaultLiked,
			&createdAt, &updatedAt, &c.AuthorUsername, &c.ItemCount, &c.ContainsItem,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		c.CreatedAt = timestamppb.New(createdAt)
		c.UpdatedAt = timestamppb.New(updatedAt)
		collections = append(collections, &c)
	}

	return connect.NewResponse(&pb.ListUserCollectionsResponse{Collections: collections}), nil
}

func (s *SocialService) AddCollectionItem(ctx context.Context, req *connect.Request[pb.AddCollectionItemRequest]) (*connect.Response[pb.CollectionItem], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	if err := checkProbationaryPeriod(ctx, s.db, u.UserUUID); err != nil {
		return nil, err
	}

	var colUUID, userUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM collections WHERE id = $1 OR uuid::text = $1", req.Msg.GetCollectionId()).Scan(&colUUID, &userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot add items to another user's collection"))
	}

	entityUUID, resolvedCol, name, image, authorOrMfg, _, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	itemUUID := authcontext.NewUUIDv7()
	_, err = s.db.Exec(ctx, `
		INSERT INTO collection_items (uuid, collection_uuid, entity_uuid, entity_id, collection, name, primary_display_image, author_or_manufacturer, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT DO NOTHING
	`, itemUUID, colUUID, entityUUID, req.Msg.GetEntityId(), resolvedCol, name, image, authorOrMfg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CollectionItem{
		Uuid:                 itemUUID,
		CollectionUuid:       colUUID,
		EntityUuid:           entityUUID,
		EntityId:             req.Msg.GetEntityId(),
		Collection:           resolvedCol,
		Name:                 name,
		PrimaryDisplayImage:  &image,
		AuthorOrManufacturer: &authorOrMfg,
		CreatedAt:            timestamppb.Now(),
	}), nil
}

func (s *SocialService) RemoveCollectionItem(ctx context.Context, req *connect.Request[pb.RemoveCollectionItemRequest]) (*connect.Response[pb.CollectionItem], error) {
	u := authcontext.UserContextFromContext(ctx)
	if u == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var colUUID, userUUID string
	err := s.db.QueryRow(ctx, "SELECT uuid, user_uuid FROM collections WHERE id = $1 OR uuid::text = $1", req.Msg.GetCollectionId()).Scan(&colUUID, &userUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
	}

	if userUUID != u.UserUUID && u.Role != "ADMIN" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cannot remove items from another user's collection"))
	}

	entityUUID, _, _, _, _, _, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.Exec(ctx, "DELETE FROM collection_items WHERE collection_uuid = $1 AND entity_uuid = $2", colUUID, entityUUID)

	return connect.NewResponse(&pb.CollectionItem{CollectionUuid: colUUID, EntityUuid: entityUUID}), nil
}

func (s *SocialService) ListCollectionItems(ctx context.Context, req *connect.Request[pb.ListCollectionItemsRequest]) (*connect.Response[pb.ListCollectionItemsResponse], error) {
	var c pb.Collection
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT
			c.uuid, c.id, c.user_uuid, c.name, c.description, c.is_public, c.is_default_liked,
			c.created_at, c.updated_at, u.id as author_username
		FROM collections c
		JOIN users u ON u.uuid = c.user_uuid
		WHERE c.id = $1 OR c.uuid::text = $1
	`, req.Msg.GetCollectionId()).Scan(
		&c.Uuid, &c.Id, &c.UserUuid, &c.Name, &c.Description, &c.IsPublic, &c.IsDefaultLiked,
		&createdAt, &updatedAt, &c.AuthorUsername,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("collection not found"))
	}

	caller := authcontext.UserContextFromContext(ctx)
	if !c.IsPublic && (caller == nil || (caller.UserUUID != c.UserUuid && caller.Role != "ADMIN")) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this collection is private"))
	}

	c.CreatedAt = timestamppb.New(createdAt)
	c.UpdatedAt = timestamppb.New(updatedAt)

	rows, err := s.db.Query(ctx, `
		SELECT
			ci.uuid, ci.collection_uuid, ci.entity_uuid, ci.created_at,
			e.id as entity_id, e.collection, e.name,
			COALESCE(e.primary_display_image, '') as primary_display_image,
			COALESCE(e.author_or_manufacturer, '') as author_or_manufacturer
		FROM collection_items ci
		JOIN entities e ON e.uuid = ci.entity_uuid
		WHERE ci.collection_uuid = $1
		ORDER BY ci.created_at DESC
		LIMIT 100
	`, c.Uuid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var items []*pb.CollectionItem
	for rows.Next() {
		var item pb.CollectionItem
		var itemCreatedAt time.Time
		var img, authorMfg string
		err := rows.Scan(
			&item.Uuid, &item.CollectionUuid, &item.EntityUuid, &itemCreatedAt,
			&item.EntityId, &item.Collection, &item.Name, &img, &authorMfg,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if img != "" {
			item.PrimaryDisplayImage = &img
		}
		if authorMfg != "" {
			item.AuthorOrManufacturer = &authorMfg
		}
		item.CreatedAt = timestamppb.New(itemCreatedAt)
		items = append(items, &item)
	}

	return connect.NewResponse(&pb.ListCollectionItemsResponse{
		Collection: &c,
		Items:      items,
	}), nil
}

func (s *SocialService) GetEntitySocialSummary(ctx context.Context, req *connect.Request[pb.GetEntitySocialSummaryRequest]) (*connect.Response[pb.EntitySocialSummary], error) {
	entityUUID, _, _, _, _, isArchived, err := resolveEntity(ctx, s.db, req.Msg.GetEntityId(), req.Msg.Collection)
	if err != nil {
		return nil, err
	}

	currentCallerUUID := ""
	if u := authcontext.UserContextFromContext(ctx); u != nil {
		currentCallerUUID = u.UserUUID
	}

	var isLiked bool
	var likeCount, qCount int32

	_ = s.db.QueryRow(ctx, `
		SELECT count(*)
		FROM collection_items ci
		JOIN collections c ON c.uuid = ci.collection_uuid
		WHERE c.is_default_liked = true AND ci.entity_uuid = $1
	`, entityUUID).Scan(&likeCount)

	if currentCallerUUID != "" {
		_ = s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM collection_items ci
				JOIN collections c ON c.uuid = ci.collection_uuid
				WHERE c.user_uuid = $1 AND c.is_default_liked = true AND ci.entity_uuid = $2
			)
		`, currentCallerUUID, entityUUID).Scan(&isLiked)
	}

	_ = s.db.QueryRow(ctx, "SELECT count(*) FROM questions WHERE entity_uuid = $1 AND deleted_at IS NULL", entityUUID).Scan(&qCount)

	reviewSummary, err := calculateReviewSummary(ctx, s.db, entityUUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.EntitySocialSummary{
		IsLiked:       isLiked,
		LikeCount:     likeCount,
		ReviewSummary: reviewSummary,
		QuestionCount: qCount,
		IsArchived:    isArchived,
	}), nil
}
