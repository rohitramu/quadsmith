package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/authcontext"
	"quadsmith/shared/pkg/blob"
	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/engines/compatibility"
	"quadsmith/shared/pkg/engines/evaluator"
	"quadsmith/shared/pkg/engines/linkpreview"
	"quadsmith/shared/pkg/engines/search"
	"quadsmith/shared/pkg/export"
	"quadsmith/shared/pkg/social"
	"quadsmith/shared/pkg/telemetry"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to API server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-api: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadAPIServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-api: configuration validation failed: %v", err)
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.Database.GetUrl())
	if err != nil {
		log.Fatalf("Invalid database URL: %v\n", err)
	}
	if cfg.Database.GetMaxConnections() > 0 {
		poolCfg.MaxConns = cfg.Database.GetMaxConnections()
	}
	if cfg.Database.GetMinConnections() > 0 {
		poolCfg.MinConns = cfg.Database.GetMinConnections()
	}
	poolCfg.ConnConfig.Tracer = telemetry.NewDBQueryTracer("quadsmith_api")
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Ensure pg_trgm extension is available for fuzzy text completion search
	_, _ = pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS pg_trgm;")

	reg := telemetry.NewRegistry("server-api", cfg.GetEnvironment(), true)
	reg.AddCheck("postgres:quadsmith_api", telemetry.PostgresPingCheck(pool))
	reg.SetDBPools(pool)

	mux := http.NewServeMux()

	handlerOpts := []connect.HandlerOption{
		connect.WithInterceptors(auth.NewAuthInterceptor(pool)),
		connect.WithReadMaxBytes(1 << 20), // 1MB limit for RPC payloads
	}

	path_NewAuthServiceHandler, h_NewAuthServiceHandler := apiconnect.NewAuthServiceHandler(auth.NewAuthService(pool, cfg.Auth), handlerOpts...)
	mux.Handle(path_NewAuthServiceHandler, h_NewAuthServiceHandler)

	path_NewUserServiceHandler, h_NewUserServiceHandler := apiconnect.NewUserServiceHandler(auth.NewUserService(pool), handlerOpts...)
	mux.Handle(path_NewUserServiceHandler, h_NewUserServiceHandler)

	path_NewMotorServiceHandler, h_NewMotorServiceHandler := apiconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewMotorServiceHandler, h_NewMotorServiceHandler)
	path_NewFrameServiceHandler, h_NewFrameServiceHandler := apiconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFrameServiceHandler, h_NewFrameServiceHandler)
	path_NewBatteryServiceHandler, h_NewBatteryServiceHandler := apiconnect.NewBatteryServiceHandler(pb.NewBatteryServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewBatteryServiceHandler, h_NewBatteryServiceHandler)
	path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler := apiconnect.NewElectronicSpeedControllerServiceHandler(pb.NewElectronicSpeedControllerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler)
	path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler := apiconnect.NewFlightControllerServiceHandler(pb.NewFlightControllerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler)
	path_NewReceiverServiceHandler, h_NewReceiverServiceHandler := apiconnect.NewReceiverServiceHandler(pb.NewReceiverServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewReceiverServiceHandler, h_NewReceiverServiceHandler)
	path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler := apiconnect.NewVideoTransmitterServiceHandler(pb.NewVideoTransmitterServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler)
	path_NewAntennaServiceHandler, h_NewAntennaServiceHandler := apiconnect.NewAntennaServiceHandler(pb.NewAntennaServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewAntennaServiceHandler, h_NewAntennaServiceHandler)
	path_NewCameraServiceHandler, h_NewCameraServiceHandler := apiconnect.NewCameraServiceHandler(pb.NewCameraServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewCameraServiceHandler, h_NewCameraServiceHandler)
	path_NewPropellerServiceHandler, h_NewPropellerServiceHandler := apiconnect.NewPropellerServiceHandler(pb.NewPropellerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewPropellerServiceHandler, h_NewPropellerServiceHandler)
	path_NewBuildServiceHandler, h_NewBuildServiceHandler := apiconnect.NewBuildServiceHandler(pb.NewBuildServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewBuildServiceHandler, h_NewBuildServiceHandler)
	path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler := apiconnect.NewGpsReceiverServiceHandler(pb.NewGpsReceiverServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler)

	path_NewFcFirmwareServiceHandler, h_NewFcFirmwareServiceHandler := apiconnect.NewFcFirmwareServiceHandler(pb.NewFcFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFcFirmwareServiceHandler, h_NewFcFirmwareServiceHandler)
	path_NewEscFirmwareServiceHandler, h_NewEscFirmwareServiceHandler := apiconnect.NewEscFirmwareServiceHandler(pb.NewEscFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewEscFirmwareServiceHandler, h_NewEscFirmwareServiceHandler)
	path_NewVtxFirmwareServiceHandler, h_NewVtxFirmwareServiceHandler := apiconnect.NewVtxFirmwareServiceHandler(pb.NewVtxFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewVtxFirmwareServiceHandler, h_NewVtxFirmwareServiceHandler)
	path_NewRadioFirmwareServiceHandler, h_NewRadioFirmwareServiceHandler := apiconnect.NewRadioFirmwareServiceHandler(pb.NewRadioFirmwareServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewRadioFirmwareServiceHandler, h_NewRadioFirmwareServiceHandler)
	path_NewSoftwareToolServiceHandler, h_NewSoftwareToolServiceHandler := apiconnect.NewSoftwareToolServiceHandler(pb.NewSoftwareToolServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewSoftwareToolServiceHandler, h_NewSoftwareToolServiceHandler)

	path_NewTransmitterServiceHandler, h_NewTransmitterServiceHandler := apiconnect.NewTransmitterServiceHandler(pb.NewTransmitterServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewTransmitterServiceHandler, h_NewTransmitterServiceHandler)
	path_NewGogglesServiceHandler, h_NewGogglesServiceHandler := apiconnect.NewGogglesServiceHandler(pb.NewGogglesServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewGogglesServiceHandler, h_NewGogglesServiceHandler)
	path_NewChargerServiceHandler, h_NewChargerServiceHandler := apiconnect.NewChargerServiceHandler(pb.NewChargerServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewChargerServiceHandler, h_NewChargerServiceHandler)
	path_NewFieldToolServiceHandler, h_NewFieldToolServiceHandler := apiconnect.NewFieldToolServiceHandler(pb.NewFieldToolServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewFieldToolServiceHandler, h_NewFieldToolServiceHandler)

	path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler := apiconnect.NewEvaluatorServiceHandler(evaluator.NewEvaluatorServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler)

	path_NewSearchServiceHandler, h_NewSearchServiceHandler := apiconnect.NewSearchServiceHandler(search.NewSearchServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewSearchServiceHandler, h_NewSearchServiceHandler)

	path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler := apiconnect.NewLinkPreviewServiceHandler(linkpreview.NewLinkPreviewServiceHandler(), handlerOpts...)
	mux.Handle(path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler)

	path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler := apiconnect.NewCompatibilityServiceHandler(compatibility.NewCompatibilityServiceHandler(pool), handlerOpts...)
	mux.Handle(path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler)

	path_NewSocialServiceHandler, h_NewSocialServiceHandler := apiconnect.NewSocialServiceHandler(social.NewSocialService(pool), handlerOpts...)
	mux.Handle(path_NewSocialServiceHandler, h_NewSocialServiceHandler)

	// Blob storage & streaming endpoints
	blobHandler := blob.NewHandler(pool, func(ctx context.Context, r *http.Request) *authcontext.UserContext {
		return auth.AuthenticateHTTPRequest(ctx, pool, r)
	})
	mux.Handle("/api/blobs", blobHandler)
	mux.Handle("/api/blobs/", blobHandler)
	mux.Handle("/blobs/", blobHandler)

	// GDPR account data export streaming endpoint
	mux.Handle("/api/account/export", export.NewExportHandler(pool))

	// Deep health check and Prometheus metrics endpoints
	mux.HandleFunc("/healthz", reg.HealthzHandler())
	mux.HandleFunc("/metrics", reg.MetricsHandler())

	// Root path handler for API server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Quadsmith API Server\n"))
	})

	port := strconv.Itoa(int(cfg.Server.GetPort()))
	fmt.Printf("Starting Quadsmith API Server on port %s...\n", port)
	if cfg.GetEnvironment() != "prod" {
		printPreSeededUsers(os.Stdout, pool)
	}

	maxBytes := cfg.Server.GetMaxRequestBytes()
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}

	// Wrap root handler with security headers, route-aware request body size limit, and telemetry middleware
	rootHandler := reg.Middleware(securityHeadersMiddleware(dynamicMaxBytesHandler(mux, maxBytes)))

	readTimeout := 30 * time.Second
	if cfg.Server.ReadTimeout != nil && cfg.Server.ReadTimeout.AsDuration() > 0 {
		readTimeout = cfg.Server.ReadTimeout.AsDuration()
	}
	writeTimeout := 60 * time.Second
	if cfg.Server.WriteTimeout != nil && cfg.Server.WriteTimeout.AsDuration() > 0 {
		writeTimeout = cfg.Server.WriteTimeout.AsDuration()
	}
	idleTimeout := 120 * time.Second
	if cfg.Server.IdleTimeout != nil && cfg.Server.IdleTimeout.AsDuration() > 0 {
		idleTimeout = cfg.Server.IdleTimeout.AsDuration()
	}

	corsMiddleware := newCORSMiddleware()

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(corsMiddleware.Handler(rootHandler), &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
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

func newCORSMiddleware() *cors.Cors {
	return cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := u.Hostname()
			if host == "localhost" || host == "127.0.0.1" {
				return true
			}
			if host == "quadsmith.com" || strings.HasSuffix(host, ".quadsmith.com") {
				return true
			}
			return false
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{
			"Grpc-Status",
			"Grpc-Message",
			"Grpc-Status-Details-Bin",
			"Connect-End-Stream",
			"X-Trace-Id",
		},
		AllowCredentials: true,
	})
}

func printPreSeededUsers(w io.Writer, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, `
		SELECT COALESCE(ue.email, ''), u.id
		FROM users u
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE u.deleted_at IS NULL
		ORDER BY ue.email, u.id
	`)
	if err != nil {
		log.Printf("Warning: failed to query pre-seeded users: %v\n", err)
		return
	}
	defer rows.Close()

	type userEntry struct {
		email string
		id    string
	}
	var users []userEntry
	for rows.Next() {
		var u userEntry
		if err := rows.Scan(&u.email, &u.id); err == nil && (u.email != "" || u.id != "") {
			users = append(users, u)
		}
	}

	if len(users) > 0 {
		fmt.Fprintln(w, "Pre-seeded users:")
		for _, u := range users {
			if u.email != "" {
				fmt.Fprintf(w, "  - %s (@%s)\n", u.email, u.id)
			} else {
				fmt.Fprintf(w, "  - @%s\n", u.id)
			}
		}
	} else {
		fmt.Fprintln(w, "Pre-seeded users: (none)")
	}
}
