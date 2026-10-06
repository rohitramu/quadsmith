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

	mux := http.NewServeMux()

	path_NewMotorServiceHandler, h_NewMotorServiceHandler := quadsmithconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool))
	mux.Handle(path_NewMotorServiceHandler, h_NewMotorServiceHandler)
	path_NewFrameServiceHandler, h_NewFrameServiceHandler := quadsmithconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool))
	mux.Handle(path_NewFrameServiceHandler, h_NewFrameServiceHandler)
	path_NewBatteryServiceHandler, h_NewBatteryServiceHandler := quadsmithconnect.NewBatteryServiceHandler(pb.NewBatteryServiceHandler(pool))
	mux.Handle(path_NewBatteryServiceHandler, h_NewBatteryServiceHandler)
	path_NewEscServiceHandler, h_NewEscServiceHandler := quadsmithconnect.NewEscServiceHandler(pb.NewEscServiceHandler(pool))
	mux.Handle(path_NewEscServiceHandler, h_NewEscServiceHandler)
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
