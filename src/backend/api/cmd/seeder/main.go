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
		"builds", "motors", "frames", "batteries", "electronic_speed_controllers", "flight_controllers",
		"receivers", "video_transmitters", "antennas", "cameras", "propellers", "gps_receivers",
	}
	for _, table := range tables {
		tx.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE;")
	}

	fmt.Println("Seeding Database from textproto...")

	dir := "src/backend/db/seeds/"

	// Motors
	if b, err := os.ReadFile(dir + "motors.textproto"); err == nil {
		m := &pb.ListMotorsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Motors {
			if err := pb.CreateMotor(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Frames
	if b, err := os.ReadFile(dir + "frames.textproto"); err == nil {
		m := &pb.ListFramesResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Frames {
			if v.MotorCount == 0 {
				v.MotorCount = 4
			}
			if err := pb.CreateFrame(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Batteries
	if b, err := os.ReadFile(dir + "batteries.textproto"); err == nil {
		m := &pb.ListBatteriesResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Batteries {
			if err := pb.CreateBattery(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Electronic Speed Controllers
	if b, err := os.ReadFile(dir + "electronic_speed_controllers.textproto"); err == nil {
		m := &pb.ListElectronicSpeedControllersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.ElectronicSpeedControllers {
			if err := pb.CreateElectronicSpeedController(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Antennas
	if b, err := os.ReadFile(dir + "antennas.textproto"); err == nil {
		m := &pb.ListAntennasResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Antennas {
			if err := pb.CreateAntenna(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Cameras
	if b, err := os.ReadFile(dir + "cameras.textproto"); err == nil {
		m := &pb.ListCamerasResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Cameras {
			if err := pb.CreateCamera(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Propellers
	if b, err := os.ReadFile(dir + "propellers.textproto"); err == nil {
		m := &pb.ListPropellersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Propellers {
			if err := pb.CreatePropeller(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Receivers
	if b, err := os.ReadFile(dir + "receivers.textproto"); err == nil {
		m := &pb.ListReceiversResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Receivers {
			if err := pb.CreateReceiver(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Video Transmitters
	if b, err := os.ReadFile(dir + "video_transmitters.textproto"); err == nil {
		m := &pb.ListVideoTransmittersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.VideoTransmitters {
			if err := pb.CreateVideoTransmitter(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Flight Controllers
	if b, err := os.ReadFile(dir + "flight_controllers.textproto"); err == nil {
		m := &pb.ListFlightControllersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.FlightControllers {
			if err := pb.CreateFlightController(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// GPS Receivers
	if b, err := os.ReadFile(dir + "gps_receivers.textproto"); err == nil {
		m := &pb.ListGpsReceiversResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.GpsReceivers {
			if err := pb.CreateGpsReceiver(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Builds
	if b, err := os.ReadFile(dir + "builds.textproto"); err == nil {
		m := &pb.ListBuildsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Builds {
			if err := pb.CreateBuild(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Clean up IEEE float precision artifacts on numeric columns
	tx.Exec(ctx, "UPDATE batteries SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), max_current_a = ROUND(max_current_a, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE flight_controllers SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE electronic_speed_controllers SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE motors SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), max_current_a = ROUND(max_current_a, 2), weight_g = ROUND(weight_g, 2);")

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}
