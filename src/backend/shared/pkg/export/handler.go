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
	users, err := pb.ExportUsers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	profiles, err := pb.ExportUserProfiles(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	emails, err := pb.ExportUserEmails(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	oauths, err := pb.ExportOAuthAccounts(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	emailOauths, err := pb.ExportUserEmailOAuthAccounts(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	histories, err := pb.ExportUsernameHistories(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	reputations, err := pb.ExportUserReputations(ctx, pool, userUUID)
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
		UserReputations        []json.RawMessage `json:"user_reputations"`
	}{
		Users:                  mustProtoSlice(users),
		UserProfiles:           mustProtoSlice(profiles),
		UserEmails:             mustProtoSlice(emails),
		OAuthAccounts:          mustProtoSlice(oauths),
		UserEmailOAuthAccounts: mustProtoSlice(emailOauths),
		UsernameHistories:      mustProtoSlice(histories),
		UserReputations:        mustProtoSlice(reputations),
	}
	if err := writeJSONToZip(zw, "account.json", accArchive); err != nil {
		return err
	}

	// 2. security.json
	refreshTokens, err := pb.ExportRefreshTokens(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	emailOtps, err := pb.ExportEmailOtps(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	secArchive := struct {
		RefreshTokens []json.RawMessage `json:"refresh_tokens"`
		EmailOtps     []json.RawMessage `json:"email_otps"`
	}{
		RefreshTokens: mustProtoSlice(refreshTokens),
		EmailOtps:     mustProtoSlice(emailOtps),
	}
	if err := writeJSONToZip(zw, "security.json", secArchive); err != nil {
		return err
	}

	// 3. builds.json
	allEntities, err := pb.ExportEntities(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	var buildEntities []*pb.Entity
	var compEntities []*pb.Entity
	for _, e := range allEntities {
		if e.Collection == "builds" {
			buildEntities = append(buildEntities, e)
		} else {
			compEntities = append(compEntities, e)
		}
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

	// 4. components.json (all 20 component tables)
	antennas, err := pb.ExportAntennas(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	batteries, err := pb.ExportBatteries(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	cameras, err := pb.ExportCameras(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	chargers, err := pb.ExportChargers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	escs, err := pb.ExportElectronicSpeedControllers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	escFirmwares, err := pb.ExportEscFirmwares(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	fcFirmwares, err := pb.ExportFcFirmwares(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	fieldTools, err := pb.ExportFieldTools(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	fcs, err := pb.ExportFlightControllers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	frames, err := pb.ExportFrames(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	goggles, err := pb.ExportGoggles(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	gps, err := pb.ExportGpsReceivers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	motors, err := pb.ExportMotors(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	props, err := pb.ExportPropellers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	radioFirmwares, err := pb.ExportRadioFirmwares(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	receivers, err := pb.ExportReceivers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	softwareTools, err := pb.ExportSoftwareTools(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	transmitters, err := pb.ExportTransmitters(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	vtxs, err := pb.ExportVideoTransmitters(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	vtxFirmwares, err := pb.ExportVtxFirmwares(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	compsArchive := struct {
		Entities                   []json.RawMessage `json:"entities"`
		Antennas                   []json.RawMessage `json:"antennas"`
		Batteries                  []json.RawMessage `json:"batteries"`
		Cameras                    []json.RawMessage `json:"cameras"`
		Chargers                   []json.RawMessage `json:"chargers"`
		ElectronicSpeedControllers []json.RawMessage `json:"electronic_speed_controllers"`
		EscFirmwares               []json.RawMessage `json:"esc_firmwares"`
		FcFirmwares                []json.RawMessage `json:"fc_firmwares"`
		FieldTools                 []json.RawMessage `json:"field_tools"`
		FlightControllers          []json.RawMessage `json:"flight_controllers"`
		Frames                     []json.RawMessage `json:"frames"`
		Goggles                    []json.RawMessage `json:"goggles"`
		GpsReceivers               []json.RawMessage `json:"gps_receivers"`
		Motors                     []json.RawMessage `json:"motors"`
		Propellers                 []json.RawMessage `json:"propellers"`
		RadioFirmwares             []json.RawMessage `json:"radio_firmwares"`
		Receivers                  []json.RawMessage `json:"receivers"`
		SoftwareTools              []json.RawMessage `json:"software_tools"`
		Transmitters               []json.RawMessage `json:"transmitters"`
		VideoTransmitters          []json.RawMessage `json:"video_transmitters"`
		VtxFirmwares               []json.RawMessage `json:"vtx_firmwares"`
	}{
		Entities:                   mustProtoSlice(compEntities),
		Antennas:                   mustProtoSlice(antennas),
		Batteries:                  mustProtoSlice(batteries),
		Cameras:                    mustProtoSlice(cameras),
		Chargers:                   mustProtoSlice(chargers),
		ElectronicSpeedControllers: mustProtoSlice(escs),
		EscFirmwares:               mustProtoSlice(escFirmwares),
		FcFirmwares:                mustProtoSlice(fcFirmwares),
		FieldTools:                 mustProtoSlice(fieldTools),
		FlightControllers:          mustProtoSlice(fcs),
		Frames:                     mustProtoSlice(frames),
		Goggles:                    mustProtoSlice(goggles),
		GpsReceivers:               mustProtoSlice(gps),
		Motors:                     mustProtoSlice(motors),
		Propellers:                 mustProtoSlice(props),
		RadioFirmwares:             mustProtoSlice(radioFirmwares),
		Receivers:                  mustProtoSlice(receivers),
		SoftwareTools:              mustProtoSlice(softwareTools),
		Transmitters:               mustProtoSlice(transmitters),
		VideoTransmitters:          mustProtoSlice(vtxs),
		VtxFirmwares:               mustProtoSlice(vtxFirmwares),
	}
	if err := writeJSONToZip(zw, "components.json", compsArchive); err != nil {
		return err
	}

	// 5. collections.json
	collections, err := pb.ExportCollections(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	collItems, err := pb.ExportCollectionItems(ctx, pool, userUUID)
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
	reviews, err := pb.ExportReviews(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	reviewVotes, err := pb.ExportReviewVotes(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	reviewComments, err := pb.ExportReviewComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questions, err := pb.ExportQuestions(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questionComments, err := pb.ExportQuestionComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answers, err := pb.ExportAnswers(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answerComments, err := pb.ExportAnswerComments(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	questionVotes, err := pb.ExportQuestionVotes(ctx, pool, userUUID)
	if err != nil {
		return err
	}
	answerVotes, err := pb.ExportAnswerVotes(ctx, pool, userUUID)
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
	blobs, err := pb.ExportBlobs(ctx, pool, userUUID)
	if err != nil {
		return err
	}

	// Separate blob metadata (with Bytes cleared) and write raw files to zip
	blobsForJSON := make([]*pb.Blob, 0, len(blobs))
	for _, b := range blobs {
		rawBytes := b.Bytes

		// Write metadata entry without raw payload
		bCopy := proto.Clone(b).(*pb.Blob)
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
