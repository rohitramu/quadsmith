package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
)

func genUUID() string {
	u, _ := uuid.NewV7()
	return u.String()
}

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

	// Clean tables? The tables are fresh, but let's just insert.
	// Actually we should truncate to make it idempotent
	tables := []string{
		"builds", "motors", "frames", "batteries", "escs", "flight_controllers",
		"receivers", "video_transmitters", "antennas", "cameras", "propellers",
	}
	for _, table := range tables {
		tx.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE;")
	}

	fmt.Println("Seeding Motors...")
	m1 := &pb.Motor{
		Uuid:             genUUID(),
		Id:               "tmotor-f80-pro-1900kv",
		Manufacturer:     "T-Motor",
		Model:            "F80 PRO 1900KV",
		WeightG:          39.7,
		StatorDiameterMm: 24,
		StatorHeightMm:   8,
		Kv:               1900,
	}
	if err := pb.CreateMotor(ctx, tx, m1); err != nil { panic(err) }

	fmt.Println("Seeding Frames...")
	f1 := &pb.Frame{
		Uuid:         genUUID(),
		Id:           "geprc-mark5",
		Manufacturer: "GEPRC",
		Model:        "Mark5",
		WeightG:      130.0,
		WheelbaseMm:  225.0,
		Geometry:     "True-X",
	}
	if err := pb.CreateFrame(ctx, tx, f1); err != nil { panic(err) }

	fmt.Println("Seeding Batteries...")
	b1 := &pb.Battery{
		Uuid:         genUUID(),
		Id:           "cnhl-black-1300mah-6s",
		Manufacturer: "CNHL",
		Model:        "Black Series 1300mAh 6S",
		WeightG:      230.0,
		CapacityMah:  1300,
		CellCountS: 6,
		Chemistry: "LiPo",
		
		Connector:    "XT60",
	}
	if err := pb.CreateBattery(ctx, tx, b1); err != nil { panic(err) }

	fmt.Println("Seeding ESCs...")
	e1 := &pb.Esc{
		Uuid:             genUUID(),
		Id:               "hobbywing-xrotor-60a-4in1",
		Manufacturer:     "Hobbywing",
		Model:            "XRotor 60A 4-in-1",
		WeightG:          15.0,
		ContinuousAmps: 60,
		BurstAmps: 80,
		Firmware: "BLHeli_32",
	}
	if err := pb.CreateEsc(ctx, tx, e1); err != nil { panic(err) }

	fmt.Println("Seeding Receivers...")
	r1 := &pb.Receiver{
		Uuid:         genUUID(),
		Id:           "happymodel-ep1-elrs",
		Manufacturer: "Happymodel",
		Model:        "EP1 ELRS",
		WeightG:      0.42,
		Protocol:     "ExpressLRS",
		FrequencyBandGhz: 2.4,
	}
	if err := pb.CreateReceiver(ctx, tx, r1); err != nil { panic(err) }

	fmt.Println("Seeding VTXs...")
	v1 := &pb.VideoTransmitter{
		Uuid:         genUUID(),
		Id:           "dji-o3-air-unit",
		Manufacturer: "DJI",
		Model:        "O3 Air Unit",
		WeightG:      36.4,
		Protocol: "DJI O3",
		MaxPowerMw:   1995,
	}
	if err := pb.CreateVideoTransmitter(ctx, tx, v1); err != nil { panic(err) }

	fmt.Println("Seeding Flight Controllers...")
	fc1 := &pb.FlightController{
		Uuid:         genUUID(),
		Id:           "matek-f405-te",
		Manufacturer: "Matek",
		Model:        "F405-TE",
		WeightG:      7.0,
		Processor: "STM32F405",
		Gyro: "ICM42688P",
		InternalEscUuid: e1.Uuid,
		InternalReceiverUuid: r1.Uuid,
		InternalVtxUuid: v1.Uuid,
	}
	if err := pb.CreateFlightController(ctx, tx, fc1); err != nil { panic(err) }



	fmt.Println("Seeding Antennas...")
	a1 := &pb.Antenna{
		Uuid:             genUUID(),
		Id:               "lumenier-axii-2",
		Manufacturer:     "Lumenier",
		Model:            "AXII 2",
		WeightG:          2.2,
		Connector:        "SMA",
		Polarization:     "RHCP",
		FrequencyBandGhz: 5.8,
		LengthMm:         20.0,
		GainDbi:          2.2,
	}
	if err := pb.CreateAntenna(ctx, tx, a1); err != nil { panic(err) }

	fmt.Println("Seeding Cameras...")
	c1 := &pb.Camera{
		Uuid:         genUUID(),
		Id:           "runcam-phoenix-2",
		Manufacturer: "RunCam",
		Model:        "Phoenix 2",
		WeightG:      9.0,
		Protocol: "MIPI",
		WidthMm: 19,
		LensSizeMm: 2.1,
		SensorSize:   "1/2\"",
	}
	if err := pb.CreateCamera(ctx, tx, c1); err != nil { panic(err) }

	fmt.Println("Seeding Propellers...")
	p1 := &pb.Propeller{
		Uuid:         genUUID(),
		Id:           "hqprop-5x4.3x3-v1s",
		Manufacturer: "HQProp",
		Model:        "5x4.3x3 V1S",
		WeightG:      3.8,
		DiameterInches: 5.0,
		PitchInches:  4.3,
		Blades:       3,
		Material:     "Polycarbonate",
	}
	if err := pb.CreatePropeller(ctx, tx, p1); err != nil { panic(err) }

	fmt.Println("Seeding Builds...")
	build1 := &pb.Build{
		Uuid:               genUUID(),
		Id:                 "mark5-o3-freestyle",
		Name:               "Mark5 O3 Freestyle",
		FrameUuid:          f1.Uuid,
		MotorUuid:          m1.Uuid,
		BatteryUuid:        b1.Uuid,
		EscUuids: []string{e1.Uuid},
		FlightControllerUuid: fc1.Uuid,
		ReceiverUuids: []string{r1.Uuid},
		
		AntennaUuids: []string{a1.Uuid},
		CameraUuids: []string{c1.Uuid},
		PropellerUuid:      p1.Uuid,
	}
	if err := pb.CreateBuild(ctx, tx, build1); err != nil { panic(err) }

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}
