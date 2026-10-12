package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/authcontext"
	"quadsmith/api/internal/engines/compatibility"
	"quadsmith/api/internal/engines/evaluator"
	"quadsmith/api/internal/engines/linkpreview"
	"quadsmith/api/internal/engines/search"
	"quadsmith/api/internal/static"
	"quadsmith/api/pkg/auth"
	"quadsmith/api/pkg/blob"
	"quadsmith/api/pkg/social"
	bconfig "quadsmith/backend/config"
	fconfig "quadsmith/frontend/config"
)

func main() {
	var configPath string
	var envName string
	flag.StringVar(&configPath, "config", "", "Path to backend configuration YAML file")
	flag.StringVar(&envName, "env", "", "Active environment (sandbox, test, staging, prod)")
	flag.Parse()

	if configPath == "" && envName != "" {
		resolved, err := bconfig.FindBackendConfigFile(envName)
		if err != nil {
			log.Fatalf("Failed to locate config for environment %q: %v", envName, err)
		}
		configPath = resolved
	}

	cfg, err := bconfig.Load(configPath)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := cfg.ValidateServer(); err != nil {
		log.Fatalf("%v", err)
	}

	fcfg, _ := fconfig.Load(cfg.Environment)

	pool, err := pgxpool.New(context.Background(), cfg.Database.URL)
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

	path_NewAuthServiceHandler, h_NewAuthServiceHandler := quadsmithconnect.NewAuthServiceHandler(auth.NewAuthService(pool, &cfg.Auth), handlerOpts...)
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

	// Blob storage & streaming endpoints
	blobHandler := blob.NewHandler(pool, func(ctx context.Context, r *http.Request) *authcontext.UserContext {
		return auth.AuthenticateHTTPRequest(ctx, pool, r)
	})
	mux.Handle("/api/blobs", blobHandler)
	mux.Handle("/api/blobs/", blobHandler)
	mux.Handle("/blobs/", blobHandler)

	// Public config endpoint
	mux.HandleFunc("/api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"environment": cfg.Environment,
			"ui":          fcfg.UI,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Serve the React SPA for any unmatched paths
	mux.Handle("/", static.ServeSPA(cfg.Server.StaticDir, pool))

	port := strconv.Itoa(cfg.Server.Port)
	fmt.Printf("Starting Quadsmith API Server on port %s...\n", port)

	maxBytes := cfg.Server.MaxRequestBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}

	// Wrap root handler with security headers and route-aware request body size limit
	rootHandler := securityHeadersMiddleware(dynamicMaxBytesHandler(mux, maxBytes))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(rootHandler, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func dynamicMaxBytesHandler(next http.Handler, defaultLimit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := defaultLimit
		if strings.HasPrefix(r.URL.Path, "/api/blobs") {
			limit = 100 << 20 // 100MB for blob uploads
		}
		http.MaxBytesHandler(next, limit).ServeHTTP(w, r)
	})
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
