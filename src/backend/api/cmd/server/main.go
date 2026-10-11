package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/engines/compatibility"
	"quadsmith/api/internal/engines/evaluator"
	"quadsmith/api/internal/engines/linkpreview"
	"quadsmith/api/internal/engines/search"
	"quadsmith/api/internal/static"
	"quadsmith/api/pkg/auth"
	"quadsmith/api/pkg/social"
)

func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/quadsmith"
	}

	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Ensure pg_trgm extension is available for fuzzy text completion search
	_, _ = pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS pg_trgm;")

	mux := http.NewServeMux()

	handlerOpts := []connect.HandlerOption{
		connect.WithInterceptors(auth.NewAuthInterceptor(pool)),
		connect.WithReadMaxBytes(1 << 20), // 1MB limit for RPC payloads
	}

	path_NewAuthServiceHandler, h_NewAuthServiceHandler := quadsmithconnect.NewAuthServiceHandler(auth.NewAuthService(pool), handlerOpts...)
	mux.Handle(path_NewAuthServiceHandler, h_NewAuthServiceHandler)

	path_NewUserServiceHandler, h_NewUserServiceHandler := quadsmithconnect.NewUserServiceHandler(auth.NewUserService(pool), handlerOpts...)
	mux.Handle(path_NewUserServiceHandler, h_NewUserServiceHandler)

	path_NewMotorServiceHandler, h_NewMotorServiceHandler := quadsmithconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewMotorServiceHandler, h_NewMotorServiceHandler)
	path_NewFrameServiceHandler, h_NewFrameServiceHandler := quadsmithconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFrameServiceHandler, h_NewFrameServiceHandler)
	path_NewBatteryServiceHandler, h_NewBatteryServiceHandler := quadsmithconnect.NewBatteryServiceHandler(pb.NewBatteryServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewBatteryServiceHandler, h_NewBatteryServiceHandler)
	path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler := quadsmithconnect.NewElectronicSpeedControllerServiceHandler(pb.NewElectronicSpeedControllerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler)
	path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler := quadsmithconnect.NewFlightControllerServiceHandler(pb.NewFlightControllerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler)
	path_NewReceiverServiceHandler, h_NewReceiverServiceHandler := quadsmithconnect.NewReceiverServiceHandler(pb.NewReceiverServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewReceiverServiceHandler, h_NewReceiverServiceHandler)
	path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler := quadsmithconnect.NewVideoTransmitterServiceHandler(pb.NewVideoTransmitterServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler)
	path_NewAntennaServiceHandler, h_NewAntennaServiceHandler := quadsmithconnect.NewAntennaServiceHandler(pb.NewAntennaServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewAntennaServiceHandler, h_NewAntennaServiceHandler)
	path_NewCameraServiceHandler, h_NewCameraServiceHandler := quadsmithconnect.NewCameraServiceHandler(pb.NewCameraServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewCameraServiceHandler, h_NewCameraServiceHandler)
	path_NewPropellerServiceHandler, h_NewPropellerServiceHandler := quadsmithconnect.NewPropellerServiceHandler(pb.NewPropellerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewPropellerServiceHandler, h_NewPropellerServiceHandler)
	path_NewBuildServiceHandler, h_NewBuildServiceHandler := quadsmithconnect.NewBuildServiceHandler(pb.NewBuildServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewBuildServiceHandler, h_NewBuildServiceHandler)
	path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler := quadsmithconnect.NewGpsReceiverServiceHandler(pb.NewGpsReceiverServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler)

	path_NewFcFirmwareServiceHandler, h_NewFcFirmwareServiceHandler := quadsmithconnect.NewFcFirmwareServiceHandler(pb.NewFcFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFcFirmwareServiceHandler, h_NewFcFirmwareServiceHandler)
	path_NewEscFirmwareServiceHandler, h_NewEscFirmwareServiceHandler := quadsmithconnect.NewEscFirmwareServiceHandler(pb.NewEscFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewEscFirmwareServiceHandler, h_NewEscFirmwareServiceHandler)
	path_NewVtxFirmwareServiceHandler, h_NewVtxFirmwareServiceHandler := quadsmithconnect.NewVtxFirmwareServiceHandler(pb.NewVtxFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewVtxFirmwareServiceHandler, h_NewVtxFirmwareServiceHandler)
	path_NewRadioFirmwareServiceHandler, h_NewRadioFirmwareServiceHandler := quadsmithconnect.NewRadioFirmwareServiceHandler(pb.NewRadioFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewRadioFirmwareServiceHandler, h_NewRadioFirmwareServiceHandler)
	path_NewSoftwareToolServiceHandler, h_NewSoftwareToolServiceHandler := quadsmithconnect.NewSoftwareToolServiceHandler(pb.NewSoftwareToolServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewSoftwareToolServiceHandler, h_NewSoftwareToolServiceHandler)

	path_NewTransmitterServiceHandler, h_NewTransmitterServiceHandler := quadsmithconnect.NewTransmitterServiceHandler(pb.NewTransmitterServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewTransmitterServiceHandler, h_NewTransmitterServiceHandler)
	path_NewGogglesServiceHandler, h_NewGogglesServiceHandler := quadsmithconnect.NewGogglesServiceHandler(pb.NewGogglesServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewGogglesServiceHandler, h_NewGogglesServiceHandler)
	path_NewChargerServiceHandler, h_NewChargerServiceHandler := quadsmithconnect.NewChargerServiceHandler(pb.NewChargerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewChargerServiceHandler, h_NewChargerServiceHandler)
	path_NewFieldToolServiceHandler, h_NewFieldToolServiceHandler := quadsmithconnect.NewFieldToolServiceHandler(pb.NewFieldToolServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFieldToolServiceHandler, h_NewFieldToolServiceHandler)

	path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler := quadsmithconnect.NewEvaluatorServiceHandler(evaluator.NewEvaluatorServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler)

	path_NewSearchServiceHandler, h_NewSearchServiceHandler := quadsmithconnect.NewSearchServiceHandler(search.NewSearchServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewSearchServiceHandler, h_NewSearchServiceHandler)

	path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler := quadsmithconnect.NewLinkPreviewServiceHandler(linkpreview.NewLinkPreviewServiceHandler(), handlerOpts...)
	mux.Handle(path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler)

	path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler := quadsmithconnect.NewCompatibilityServiceHandler(compatibility.NewCompatibilityServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler)

	path_NewSocialServiceHandler, h_NewSocialServiceHandler := quadsmithconnect.NewSocialServiceHandler(social.NewSocialService(pool), handlerOpts...)
	mux.Handle(path_NewSocialServiceHandler, h_NewSocialServiceHandler)

	// Serve the React SPA for any unmatched paths
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "/app/web/dist"
	}
	mux.Handle("/", static.ServeSPA(staticDir, pool))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ensurePreSeededUsers(pool)
	fmt.Printf("Starting Quadsmith API Server on port %s...\n", port)
	printPreSeededUserEmails(pool)

	// Wrap root handler with security headers and request body size limit (2 MB)
	rootHandler := securityHeadersMiddleware(http.MaxBytesHandler(mux, 2<<20))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(rootHandler, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval' https://www.gstatic.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; frame-src https://www.youtube.com https://www.youtube-nocookie.com; frame-ancestors 'none'; object-src 'none'; base-uri 'self';")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func ensurePreSeededUsers(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = pool.Exec(ctx, `
		INSERT INTO users (uuid, id, email, role, created_at, updated_at)
		VALUES 
			('018f0000-0000-7000-8000-000000000001', 'test_pilot', 'pilot@example.com', 'user', NOW() - INTERVAL '2 days', NOW()),
			('018f0000-0000-7000-8000-000000000010', 'alex', 'alex@quadsmith.dev', 'user', NOW() - INTERVAL '2 days', NOW())
		ON CONFLICT (uuid) DO NOTHING;

		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES 
			('018f0000-0000-7000-8000-000000000002', '018f0000-0000-7000-8000-000000000001', '', 'E2E Test Account', NOW()),
			('018f0000-0000-7000-8000-000000000011', '018f0000-0000-7000-8000-000000000010', '', 'Community Builder & Pilot', NOW())
		ON CONFLICT (user_uuid) DO NOTHING;

		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
		VALUES 
			('018f0000-0000-7000-8000-000000000003', '018f0000-0000-7000-8000-000000000001', 'pilot@example.com', true, NOW() - INTERVAL '2 days'),
			('018f0000-0000-7000-8000-000000000012', '018f0000-0000-7000-8000-000000000010', 'alex@quadsmith.dev', true, NOW() - INTERVAL '2 days')
		ON CONFLICT (email) DO NOTHING;

		INSERT INTO user_reputations (uuid, user_uuid, score, updated_at)
		VALUES
			('018f0000-0000-7000-8000-000000000004', '018f0000-0000-7000-8000-000000000001', 150, NOW()),
			('018f0000-0000-7000-8000-000000000013', '018f0000-0000-7000-8000-000000000010', 250, NOW())
		ON CONFLICT (user_uuid) DO NOTHING;
	`)
}

func printPreSeededUserEmails(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, `
		SELECT DISTINCT email FROM (
			SELECT email FROM user_emails
			UNION
			SELECT email FROM users WHERE deleted_at IS NULL
		) sub
		WHERE email IS NOT NULL AND email != ''
		ORDER BY email
	`)
	if err != nil {
		log.Printf("Warning: failed to query pre-seeded user emails: %v\n", err)
		return
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil && email != "" {
			emails = append(emails, email)
		}
	}

	if len(emails) > 0 {
		fmt.Println("Pre-seeded user emails:")
		for _, email := range emails {
			fmt.Printf("  - %s\n", email)
		}
	} else {
		fmt.Println("Pre-seeded user emails: (none)")
	}
}
