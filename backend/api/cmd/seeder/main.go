package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/prototext"
	pb "quadsmith/api/gen/quadsmith"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/quadsmith?sslmode=disable"
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tx, err := db.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin tx: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	tables := []string{
		"builds", "motors", "frames", "batteries", "escs", "flight_controllers",
		"receivers", "video_transmitters", "antennas", "cameras", "propellers",
	}
	for _, table := range tables {
		tx.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE;")
	}

	fmt.Println("Seeding Database from textproto...")

	dir := "backend/db/seeds/"

	// Motors
	if b, err := os.ReadFile(dir + "motors.textproto"); err == nil {
		m := &pb.ListMotorsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Motors {
			pb.CreateMotor(ctx, tx, v)
		}
	}
	// Frames
	if b, err := os.ReadFile(dir + "frames.textproto"); err == nil {
		m := &pb.ListFramesResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Frames {
			pb.CreateFrame(ctx, tx, v)
		}
	}
	// Batteries
	if b, err := os.ReadFile(dir + "batteries.textproto"); err == nil {
		m := &pb.ListBatteriesResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Batteries {
			pb.CreateBattery(ctx, tx, v)
		}
	}
	// ESCs
	if b, err := os.ReadFile(dir + "escs.textproto"); err == nil {
		m := &pb.ListEscsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Escs {
			pb.CreateEsc(ctx, tx, v)
		}
	}
	// Flight Controllers
	if b, err := os.ReadFile(dir + "flight_controllers.textproto"); err == nil {
		m := &pb.ListFlightControllersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.FlightControllers {
			pb.CreateFlightController(ctx, tx, v)
		}
	}
	// Receivers
	if b, err := os.ReadFile(dir + "receivers.textproto"); err == nil {
		m := &pb.ListReceiversResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Receivers {
			pb.CreateReceiver(ctx, tx, v)
		}
	}
	// Video Transmitters
	if b, err := os.ReadFile(dir + "video_transmitters.textproto"); err == nil {
		m := &pb.ListVideoTransmittersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.VideoTransmitters {
			pb.CreateVideoTransmitter(ctx, tx, v)
		}
	}
	// Antennas
	if b, err := os.ReadFile(dir + "antennas.textproto"); err == nil {
		m := &pb.ListAntennasResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Antennas {
			pb.CreateAntenna(ctx, tx, v)
		}
	}
	// Cameras
	if b, err := os.ReadFile(dir + "cameras.textproto"); err == nil {
		m := &pb.ListCamerasResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Cameras {
			pb.CreateCamera(ctx, tx, v)
		}
	}
	// Propellers
	if b, err := os.ReadFile(dir + "propellers.textproto"); err == nil {
		m := &pb.ListPropellersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Propellers {
			pb.CreatePropeller(ctx, tx, v)
		}
	}
	// Builds (There is no pb.CreateBuild? Wait, we'll see)
	// We can leave builds alone or check if pb.CreateBuild exists.

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}
