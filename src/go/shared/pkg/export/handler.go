package export

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"quadsmith/shared/pkg/auth"

	pb "quadsmith/shared/gen/quadsmith/api"
	storepb "quadsmith/shared/gen/quadsmith/store"
)

// NewExportHandler creates an HTTP handler for streaming GDPR data archive ZIP files.
func NewExportHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Support token from ?token= query parameter or Authorization: Bearer <token>
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			}
		}
		if token == "" {
			http.Error(w, "missing export token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.VerifyExportToken(token)
		if err != nil {
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		var username string
		err = pool.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1 AND deleted_at IS NULL`, claims.UserUUID).Scan(&username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		dateStr := time.Now().UTC().Format("2006-01-02")
		filename := fmt.Sprintf("quadsmith-data-%s-%s.zip", username, dateStr)

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Header().Set("Cache-Control", "no-store")

		zw := zip.NewWriter(w)
		defer zw.Close()

		if err := streamArchive(ctx, pool, zw, claims.UserUUID); err != nil {
			// Once headers are sent, logging error is the standard approach
			return
		}
	})
}

func streamArchive(ctx context.Context, pool *pgxpool.Pool, zw *zip.Writer, userUUID string) error {
	// 1. account.json
	users, err := storepb.ExportUsers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	profiles, err := storepb.ExportUserProfiles(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	emails, err := storepb.ExportUserEmails(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	oauths, err := storepb.ExportOAuthAccounts(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	emailOauths, err := storepb.ExportUserEmailOAuthAccounts(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	histories, err := storepb.ExportUsernameHistories(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	accArchive := struct {
		Users                  []json.RawMessage `json:"users"`
		UserProfiles           []json.RawMessage `json:"user_profiles"`
		UserEmails             []json.RawMessage `json:"user_emails"`
		OAuthAccounts          []json.RawMessage `json:"oauth_accounts"`
		UserEmailOAuthAccounts []json.RawMessage `json:"user_email_oauth_accounts"`
		UsernameHistories      []json.RawMessage `json:"username_histories"`
	}{
		Users:                  mustProtoSlice(users),
		UserProfiles:           mustProtoSlice(profiles),
		UserEmails:             mustProtoSlice(emails),
		OAuthAccounts:          mustProtoSlice(oauths),
		UserEmailOAuthAccounts: mustProtoSlice(emailOauths),
		UsernameHistories:      mustProtoSlice(histories),
	}
	if err := writeJSONToZip(zw, "account.json", accArchive); err != nil {
		return err
	}

	// 2. security.json
	refreshTokens, err := storepb.ExportRefreshTokens(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	secArchive := struct {
		RefreshTokens []json.RawMessage `json:"refresh_tokens"`
	}{
		RefreshTokens: mustProtoSlice(refreshTokens),
	}
	if err := writeJSONToZip(zw, "security.json", secArchive); err != nil {
		return err
	}

	// 3. builds.json
	buildEntities, err := storepb.ExportEntities(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	builds, err := pb.ExportBuilds(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	buildsArchive := struct {
		Entities []json.RawMessage `json:"entities"`
		Builds   []json.RawMessage `json:"builds"`
	}{
		Entities: mustProtoSlice(buildEntities),
		Builds:   mustProtoSlice(builds),
	}
	if err := writeJSONToZip(zw, "builds.json", buildsArchive); err != nil {
		return err
	}

	// 4. collections.json
	collections, err := storepb.ExportCollections(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	collItems, err := storepb.ExportCollectionItems(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	collsArchive := struct {
		Collections     []json.RawMessage `json:"collections"`
		CollectionItems []json.RawMessage `json:"collection_items"`
	}{
		Collections:     mustProtoSlice(collections),
		CollectionItems: mustProtoSlice(collItems),
	}
	if err := writeJSONToZip(zw, "collections.json", collsArchive); err != nil {
		return err
	}

	// 6. social.json
	reviews, err := storepb.ExportReviews(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	reviewVotes, err := storepb.ExportReviewVotes(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	reviewComments, err := storepb.ExportReviewComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questions, err := storepb.ExportQuestions(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questionComments, err := storepb.ExportQuestionComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answers, err := storepb.ExportAnswers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answerComments, err := storepb.ExportAnswerComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questionVotes, err := storepb.ExportQuestionVotes(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answerVotes, err := storepb.ExportAnswerVotes(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	socArchive := struct {
		Reviews          []json.RawMessage `json:"reviews"`
		ReviewVotes      []json.RawMessage `json:"review_votes"`
		ReviewComments   []json.RawMessage `json:"review_comments"`
		Questions        []json.RawMessage `json:"questions"`
		QuestionComments []json.RawMessage `json:"question_comments"`
		Answers          []json.RawMessage `json:"answers"`
		AnswerComments   []json.RawMessage `json:"answer_comments"`
		QuestionVotes    []json.RawMessage `json:"question_votes"`
		AnswerVotes      []json.RawMessage `json:"answer_votes"`
	}{
		Reviews:          mustProtoSlice(reviews),
		ReviewVotes:      mustProtoSlice(reviewVotes),
		ReviewComments:   mustProtoSlice(reviewComments),
		Questions:        mustProtoSlice(questions),
		QuestionComments: mustProtoSlice(questionComments),
		Answers:          mustProtoSlice(answers),
		AnswerComments:   mustProtoSlice(answerComments),
		QuestionVotes:    mustProtoSlice(questionVotes),
		AnswerVotes:      mustProtoSlice(answerVotes),
	}
	if err := writeJSONToZip(zw, "social.json", socArchive); err != nil {
		return err
	}

	// 7. blobs.json + blobs/<id>_<filename>
	blobs, err := storepb.ExportBlobs(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	// Separate blob metadata (with Bytes cleared) and write raw files to zip
	blobsForJSON := make([]*storepb.Blob, 0, len(blobs))
	for _, b := range blobs {
		rawBytes := b.Bytes

		// Write metadata entry without raw payload
		bCopy := proto.Clone(b).(*storepb.Blob)
		bCopy.Bytes = nil
		blobsForJSON = append(blobsForJSON, bCopy)

		if len(rawBytes) > 0 {
			sanitizedFilename := filepath.Base(b.Filename)
			sanitizedFilename = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
					return r
				}
				return '_'
			}, sanitizedFilename)
			if sanitizedFilename == "" || sanitizedFilename == "." {
				sanitizedFilename = "file"
			}

			zipPath := fmt.Sprintf("blobs/%s_%s", b.Id, sanitizedFilename)
			fw, err := zw.Create(zipPath)
			if err != nil {
				return fmt.Errorf("failed to create zip file %s: %w", zipPath, err)
			}
			if _, err := fw.Write(rawBytes); err != nil {
				return fmt.Errorf("failed to write raw blob %s: %w", zipPath, err)
			}
		}
	}

	blobsArchive := struct {
		Blobs []json.RawMessage `json:"blobs"`
	}{
		Blobs: mustProtoSlice(blobsForJSON),
	}
	if err := writeJSONToZip(zw, "blobs.json", blobsArchive); err != nil {
		return err
	}

	return nil
}

func mustProtoSlice[T proto.Message](slice []T) []json.RawMessage {
	opts := protojson.MarshalOptions{
		UseProtoNames:   true,
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: false,
	}
	raws := make([]json.RawMessage, 0, len(slice))
	for _, item := range slice {
		b, err := opts.Marshal(item)
		if err != nil {
			continue
		}
		raws = append(raws, json.RawMessage(b))
	}
	return raws
}

func writeJSONToZip(zw *zip.Writer, filename string, v any) error {
	w, err := zw.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s in zip: %w", filename, err)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
