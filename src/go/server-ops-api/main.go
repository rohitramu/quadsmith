package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/server-ops-api/pkg/ops"
	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/authcontext"
	"quadsmith/shared/pkg/blob"
	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/opsauth"
	"quadsmith/shared/pkg/telemetry"

	configpb "quadsmith/shared/gen/quadsmith/config"
	opspb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

func connectPool(ctx context.Context, dbCfg *configpb.DatabaseConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dbCfg.GetUrl())
	if err != nil {
		return nil, err
	}
	if dbCfg.GetMaxConnections() > 0 {
		poolCfg.MaxConns = dbCfg.GetMaxConnections()
	}
	if dbCfg.GetMinConnections() > 0 {
		poolCfg.MinConns = dbCfg.GetMinConnections()
	}
	dbName := dbCfg.GetName()
	if dbName == "" {
		dbName = "postgres"
	}
	poolCfg.ConnConfig.Tracer = telemetry.NewDBQueryTracer(dbName)
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to ops API server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-ops-api: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadOpsApiServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-ops-api: configuration validation failed: %v", err)
	}

	port := 8082
	maxReqBytes := 1 << 20
	var readTimeout, writeTimeout, idleTimeout time.Duration
	if cfg.Server != nil {
		if cfg.Server.GetPort() > 0 {
			port = int(cfg.Server.GetPort())
		}
		if cfg.Server.GetMaxRequestBytes() > 0 {
			maxReqBytes = int(cfg.Server.GetMaxRequestBytes())
		}
		if cfg.Server.ReadTimeout != nil {
			readTimeout = cfg.Server.ReadTimeout.AsDuration()
		}
		if cfg.Server.WriteTimeout != nil {
			writeTimeout = cfg.Server.WriteTimeout.AsDuration()
		}
		if cfg.Server.IdleTimeout != nil {
			idleTimeout = cfg.Server.IdleTimeout.AsDuration()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publicPool, err := connectPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Unable to connect to public database (%s): %v\n", cfg.Database.GetName(), err)
	}
	defer publicPool.Close()

	internalPool, err := connectPool(ctx, cfg.InternalDatabase)
	if err != nil {
		log.Fatalf("Unable to connect to internal database (%s): %v\n", cfg.InternalDatabase.GetName(), err)
	}
	defer internalPool.Close()

	reg := telemetry.NewRegistry("server-ops-api", cfg.GetEnvironment(), true)
	reg.AddCheck("postgres:quadsmith_api", telemetry.PostgresPingCheck(publicPool))
	reg.AddCheck("postgres:quadsmith_ops", telemetry.PostgresPingCheck(internalPool))
	reg.SetDBPools(publicPool, internalPool)

	service := ops.NewOpsService(publicPool, internalPool, cfg, reg)
	interceptor := opsauth.NewOpsInterceptor(internalPool, cfg.BootstrapTokens, cfg.Groups, cfg.Jit)

	handlerOpts := []connect.HandlerOption{
		connect.WithInterceptors(interceptor),
		connect.WithReadMaxBytes(maxReqBytes),
	}

	mux := http.NewServeMux()
	path, handler := opsconnect.NewOpsServiceHandler(service, handlerOpts...)
	mux.Handle(path, handler)

	mountCatalogOpsHandlers(mux, publicPool, handlerOpts...)

	// Ops component image upload & blob streaming endpoints
	rawBlobHandler := blob.NewHandler(publicPool, func(reqCtx context.Context, _ *http.Request) *authcontext.UserContext {
		_, _ = auth.EnsureSystemUser(reqCtx, publicPool)
		return &authcontext.UserContext{
			UserUUID: authcontext.SystemUserUUID,
			UserID:   authcontext.SystemUserHandle,
			Role:     authcontext.RoleSystem,
		}
	})
	opsBlobUploadHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cookieTok string
		if c, err := r.Cookie("qs_ops_token"); err == nil {
			cookieTok = c.Value
		}
		t0 := time.Now()
		opCtx, authErr := interceptor.AuthenticateSensitiveHTTP(r.Context(), r.Header.Get("Authorization"), cookieTok, "UploadCatalogBlob")
		if authErr != nil {
			code := http.StatusForbidden
			if connect.CodeOf(authErr) == connect.CodeUnauthenticated {
				code = http.StatusUnauthorized
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":` + strconv.Quote(authErr.Error()) + `}`))
			return
		}
		rawBlobHandler.ServeHTTP(w, r)
		interceptor.RecordAudit(r.Context(), "UploadCatalogBlob", opCtx, "OK", nil, time.Since(t0).Milliseconds())
	})
	mux.Handle("/api/ops/blobs", opsBlobUploadHandler)
	mux.Handle("/api/ops/blobs/", opsBlobUploadHandler)
	mux.Handle("/blobs/", rawBlobHandler)

	mux.HandleFunc("/healthz", reg.HealthzHandler())
	mux.HandleFunc("/metrics", reg.MetricsHandler())

	corsMiddleware := cors.New(cors.Options{
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
			"X-Job-Id",
		},
		AllowCredentials: true,
	})

	instrumentedHandler := reg.Middleware(dynamicOpsMaxBytesHandler(mux, int64(maxReqBytes)))

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           h2c.NewHandler(corsMiddleware.Handler(instrumentedHandler), &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting Quadsmith Ops API Server on port %d...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down ops API server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown warning: %v", err)
	}

	log.Println("Ops API server stopped gracefully")
}

func dynamicOpsMaxBytesHandler(next http.Handler, defaultLimit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := defaultLimit
		if strings.HasPrefix(r.URL.Path, "/api/ops/blobs") {
			limit = 100 << 20 // 100MB for catalog image/media uploads
		}
		http.MaxBytesHandler(next, limit).ServeHTTP(w, r)
	})
}

func mountCatalogOpsHandlers(mux *http.ServeMux, pool *pgxpool.Pool, opts ...connect.HandlerOption) {
	register := func(path string, h http.Handler) {
		mux.Handle(path, h)
	}
	register(opsconnect.NewAntennaOpsServiceHandler(opspb.NewAntennaOpsServiceHandler(pool), opts...))
	register(opsconnect.NewBatteryOpsServiceHandler(opspb.NewBatteryOpsServiceHandler(pool), opts...))
	register(opsconnect.NewCameraOpsServiceHandler(opspb.NewCameraOpsServiceHandler(pool), opts...))
	register(opsconnect.NewChargerOpsServiceHandler(opspb.NewChargerOpsServiceHandler(pool), opts...))
	register(opsconnect.NewElectronicSpeedControllerOpsServiceHandler(opspb.NewElectronicSpeedControllerOpsServiceHandler(pool), opts...))
	register(opsconnect.NewEscFirmwareOpsServiceHandler(opspb.NewEscFirmwareOpsServiceHandler(pool), opts...))
	register(opsconnect.NewFcFirmwareOpsServiceHandler(opspb.NewFcFirmwareOpsServiceHandler(pool), opts...))
	register(opsconnect.NewFieldToolOpsServiceHandler(opspb.NewFieldToolOpsServiceHandler(pool), opts...))
	register(opsconnect.NewFlightControllerOpsServiceHandler(opspb.NewFlightControllerOpsServiceHandler(pool), opts...))
	register(opsconnect.NewFrameOpsServiceHandler(opspb.NewFrameOpsServiceHandler(pool), opts...))
	register(opsconnect.NewGogglesOpsServiceHandler(opspb.NewGogglesOpsServiceHandler(pool), opts...))
	register(opsconnect.NewGpsReceiverOpsServiceHandler(opspb.NewGpsReceiverOpsServiceHandler(pool), opts...))
	register(opsconnect.NewMotorOpsServiceHandler(opspb.NewMotorOpsServiceHandler(pool), opts...))
	register(opsconnect.NewPropellerOpsServiceHandler(opspb.NewPropellerOpsServiceHandler(pool), opts...))
	register(opsconnect.NewRadioFirmwareOpsServiceHandler(opspb.NewRadioFirmwareOpsServiceHandler(pool), opts...))
	register(opsconnect.NewReceiverOpsServiceHandler(opspb.NewReceiverOpsServiceHandler(pool), opts...))
	register(opsconnect.NewSoftwareToolOpsServiceHandler(opspb.NewSoftwareToolOpsServiceHandler(pool), opts...))
	register(opsconnect.NewTransmitterOpsServiceHandler(opspb.NewTransmitterOpsServiceHandler(pool), opts...))
	register(opsconnect.NewVideoTransmitterOpsServiceHandler(opspb.NewVideoTransmitterOpsServiceHandler(pool), opts...))
	register(opsconnect.NewVtxFirmwareOpsServiceHandler(opspb.NewVtxFirmwareOpsServiceHandler(pool), opts...))
}
