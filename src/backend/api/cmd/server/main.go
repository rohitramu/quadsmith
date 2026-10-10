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

	// Start background worker for 7-day handle release warnings
	auth.StartHandleReleaseWarningWorker(context.Background(), pool, 1*time.Hour)

	mux := http.NewServeMux()

	authInterceptor := connect.WithInterceptors(auth.NewAuthInterceptor(pool))

	path_NewAuthServiceHandler, h_NewAuthServiceHandler := quadsmithconnect.NewAuthServiceHandler(auth.NewAuthService(pool), authInterceptor)
	mux.Handle(path_NewAuthServiceHandler, h_NewAuthServiceHandler)

	path_NewUserServiceHandler, h_NewUserServiceHandler := quadsmithconnect.NewUserServiceHandler(auth.NewUserService(pool), authInterceptor)
	mux.Handle(path_NewUserServiceHandler, h_NewUserServiceHandler)

	path_NewMotorServiceHandler, h_NewMotorServiceHandler := quadsmithconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewMotorServiceHandler, h_NewMotorServiceHandler)
	path_NewFrameServiceHandler, h_NewFrameServiceHandler := quadsmithconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewFrameServiceHandler, h_NewFrameServiceHandler)
	path_NewBatteryServiceHandler, h_NewBatteryServiceHandler := quadsmithconnect.NewBatteryServiceHandler(pb.NewBatteryServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewBatteryServiceHandler, h_NewBatteryServiceHandler)
	path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler := quadsmithconnect.NewElectronicSpeedControllerServiceHandler(pb.NewElectronicSpeedControllerServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler)
	path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler := quadsmithconnect.NewFlightControllerServiceHandler(pb.NewFlightControllerServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler)
	path_NewReceiverServiceHandler, h_NewReceiverServiceHandler := quadsmithconnect.NewReceiverServiceHandler(pb.NewReceiverServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewReceiverServiceHandler, h_NewReceiverServiceHandler)
	path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler := quadsmithconnect.NewVideoTransmitterServiceHandler(pb.NewVideoTransmitterServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler)
	path_NewAntennaServiceHandler, h_NewAntennaServiceHandler := quadsmithconnect.NewAntennaServiceHandler(pb.NewAntennaServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewAntennaServiceHandler, h_NewAntennaServiceHandler)
	path_NewCameraServiceHandler, h_NewCameraServiceHandler := quadsmithconnect.NewCameraServiceHandler(pb.NewCameraServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewCameraServiceHandler, h_NewCameraServiceHandler)
	path_NewPropellerServiceHandler, h_NewPropellerServiceHandler := quadsmithconnect.NewPropellerServiceHandler(pb.NewPropellerServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewPropellerServiceHandler, h_NewPropellerServiceHandler)
	path_NewBuildServiceHandler, h_NewBuildServiceHandler := quadsmithconnect.NewBuildServiceHandler(pb.NewBuildServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewBuildServiceHandler, h_NewBuildServiceHandler)
	path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler := quadsmithconnect.NewGpsReceiverServiceHandler(pb.NewGpsReceiverServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler)

	path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler := quadsmithconnect.NewEvaluatorServiceHandler(evaluator.NewEvaluatorServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler)

	path_NewSearchServiceHandler, h_NewSearchServiceHandler := quadsmithconnect.NewSearchServiceHandler(search.NewSearchServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewSearchServiceHandler, h_NewSearchServiceHandler)

	path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler := quadsmithconnect.NewLinkPreviewServiceHandler(linkpreview.NewLinkPreviewServiceHandler(), authInterceptor)
	mux.Handle(path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler)

	path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler := quadsmithconnect.NewCompatibilityServiceHandler(compatibility.NewCompatibilityServiceHandler(pool), authInterceptor)
	mux.Handle(path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler)

	// Serve the React SPA for any unmatched paths
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "/app/web/dist"
	}
	mux.Handle("/", static.ServeSPA(staticDir, pool))

	// TODO: Add HTTP middleware for CORS to allow frontend applications to call this API.
	// TODO: Add Authentication/Authorization interceptors to secure write operations (Create/Update/Delete).

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Quadsmith API Server on port %s...\n", port)

	err = http.ListenAndServe(
		":"+port,
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
