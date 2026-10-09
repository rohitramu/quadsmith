package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/engines/compatibility"
	"quadsmith/api/internal/engines/evaluator"
	"quadsmith/api/internal/engines/linkpreview"
	"quadsmith/api/internal/engines/search"
	"quadsmith/api/internal/static"
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

	path_NewMotorServiceHandler, h_NewMotorServiceHandler := quadsmithconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool))
	mux.Handle(path_NewMotorServiceHandler, h_NewMotorServiceHandler)
	path_NewFrameServiceHandler, h_NewFrameServiceHandler := quadsmithconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool))
	mux.Handle(path_NewFrameServiceHandler, h_NewFrameServiceHandler)
	path_NewBatteryServiceHandler, h_NewBatteryServiceHandler := quadsmithconnect.NewBatteryServiceHandler(pb.NewBatteryServiceHandler(pool))
	mux.Handle(path_NewBatteryServiceHandler, h_NewBatteryServiceHandler)
	path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler := quadsmithconnect.NewElectronicSpeedControllerServiceHandler(pb.NewElectronicSpeedControllerServiceHandler(pool))
	mux.Handle(path_NewElectronicSpeedControllerServiceHandler, h_NewElectronicSpeedControllerServiceHandler)
	path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler := quadsmithconnect.NewFlightControllerServiceHandler(pb.NewFlightControllerServiceHandler(pool))
	mux.Handle(path_NewFlightControllerServiceHandler, h_NewFlightControllerServiceHandler)
	path_NewReceiverServiceHandler, h_NewReceiverServiceHandler := quadsmithconnect.NewReceiverServiceHandler(pb.NewReceiverServiceHandler(pool))
	mux.Handle(path_NewReceiverServiceHandler, h_NewReceiverServiceHandler)
	path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler := quadsmithconnect.NewVideoTransmitterServiceHandler(pb.NewVideoTransmitterServiceHandler(pool))
	mux.Handle(path_NewVideoTransmitterServiceHandler, h_NewVideoTransmitterServiceHandler)
	path_NewAntennaServiceHandler, h_NewAntennaServiceHandler := quadsmithconnect.NewAntennaServiceHandler(pb.NewAntennaServiceHandler(pool))
	mux.Handle(path_NewAntennaServiceHandler, h_NewAntennaServiceHandler)
	path_NewCameraServiceHandler, h_NewCameraServiceHandler := quadsmithconnect.NewCameraServiceHandler(pb.NewCameraServiceHandler(pool))
	mux.Handle(path_NewCameraServiceHandler, h_NewCameraServiceHandler)
	path_NewPropellerServiceHandler, h_NewPropellerServiceHandler := quadsmithconnect.NewPropellerServiceHandler(pb.NewPropellerServiceHandler(pool))
	mux.Handle(path_NewPropellerServiceHandler, h_NewPropellerServiceHandler)
	path_NewBuildServiceHandler, h_NewBuildServiceHandler := quadsmithconnect.NewBuildServiceHandler(pb.NewBuildServiceHandler(pool))
	mux.Handle(path_NewBuildServiceHandler, h_NewBuildServiceHandler)
	path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler := quadsmithconnect.NewGpsReceiverServiceHandler(pb.NewGpsReceiverServiceHandler(pool))
	mux.Handle(path_NewGpsReceiverServiceHandler, h_NewGpsReceiverServiceHandler)

	path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler := quadsmithconnect.NewEvaluatorServiceHandler(evaluator.NewEvaluatorServiceHandler(pool))
	mux.Handle(path_NewEvaluatorServiceHandler, h_NewEvaluatorServiceHandler)

	path_NewSearchServiceHandler, h_NewSearchServiceHandler := quadsmithconnect.NewSearchServiceHandler(search.NewSearchServiceHandler(pool))
	mux.Handle(path_NewSearchServiceHandler, h_NewSearchServiceHandler)

	path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler := quadsmithconnect.NewLinkPreviewServiceHandler(linkpreview.NewLinkPreviewServiceHandler())
	mux.Handle(path_NewLinkPreviewServiceHandler, h_NewLinkPreviewServiceHandler)

	path_NewCompatibilityServiceHandler, h_NewCompatibilityServiceHandler := quadsmithconnect.NewCompatibilityServiceHandler(compatibility.NewCompatibilityServiceHandler(pool))
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
