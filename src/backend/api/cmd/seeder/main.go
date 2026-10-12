package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/internal/authcontext"
	"quadsmith/api/internal/validation"
	bconfig "quadsmith/backend/config"
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
			fmt.Fprintf(os.Stderr, "Failed to locate config for environment %q: %v\n", envName, err)
			os.Exit(1)
		}
		configPath = resolved
	}

	cfg, err := bconfig.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := cfg.ValidateSeeder(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Database.URL)
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
		"users", "user_profiles", "user_emails", "user_reputations",
		"builds", "motors", "frames", "batteries", "electronic_speed_controllers", "flight_controllers",
		"receivers", "video_transmitters", "antennas", "cameras", "propellers", "gps_receivers",
		"flight_controller_firmwares", "electronic_speed_controller_firmwares", "video_transmitter_firmwares",
		"radio_firmwares", "software_tools", "transmitters", "goggles", "chargers", "field_tools",
		"reviews", "review_votes", "review_comments", "questions", "answers", "question_votes", "answer_votes",
		"question_comments", "answer_comments", "collections", "collection_items", "entities",
	}

	for _, table := range tables {
		tx.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE;")
	}

	fmt.Println("Seeding Database from textproto...")

	dir := "test/seed_data/"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		for _, fallback := range []string{"../../test/seed_data/", "../../../test/seed_data/"} {
			if _, err := os.Stat(fallback); err == nil {
				dir = fallback
				break
			}
		}
	}

	// Users
	if b, err := os.ReadFile(dir + "users.textproto"); err == nil {
		m := &pb.ListUsersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, u := range m.Users {
			userUUID := u.Uuid
			if userUUID == "" {
				userUUID = authcontext.NewUUIDv7()
			}
			now := time.Now()
			twoDaysAgo := now.Add(-48 * time.Hour)
			if err := pb.CreateUser(ctx, tx, &pb.User{
				Uuid:      userUUID,
				Id:        u.Id,
				Email:     u.Email,
				Role:      u.Role,
				CreatedAt: timestamppb.New(twoDaysAgo),
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}

			if err := pb.CreateUserProfile(ctx, tx, &pb.UserProfile{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				AvatarUrl: "",
				Bio:       u.Bio,
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}

			if err := pb.CreateUserEmail(ctx, tx, &pb.UserEmail{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				Email:     u.Email,
				IsPrimary: true,
				CreatedAt: timestamppb.New(twoDaysAgo),
			}); err != nil {
				panic(err)
			}

			if err := pb.CreateUserReputation(ctx, tx, &pb.UserReputation{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				Score:     u.ReputationScore,
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}
		}
	}

	// Motors
	if b, err := os.ReadFile(dir + "motors.textproto"); err == nil {
		m := &pb.ListMotorsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Motors {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("motor %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("frame %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("battery %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("electronic speed controller %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("antenna %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("camera %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("propeller %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("receiver %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("video transmitter %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("flight controller %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("gps receiver %s: %w", v.Id, err))
			}
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
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("build %s: %w", v.Id, err))
			}
			if err := pb.CreateBuild(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Flight Controller Firmwares
	if b, err := os.ReadFile(dir + "fc_firmwares.textproto"); err == nil {
		m := &pb.ListFcFirmwaresResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.FcFirmwares {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("fc_firmware %s: %w", v.Id, err))
			}
			if err := pb.CreateFcFirmware(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// ESC Firmwares
	if b, err := os.ReadFile(dir + "esc_firmwares.textproto"); err == nil {
		m := &pb.ListEscFirmwaresResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.EscFirmwares {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("esc_firmware %s: %w", v.Id, err))
			}
			if err := pb.CreateEscFirmware(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Video Transmitter Firmwares
	if b, err := os.ReadFile(dir + "vtx_firmwares.textproto"); err == nil {
		m := &pb.ListVtxFirmwaresResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.VtxFirmwares {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("vtx_firmware %s: %w", v.Id, err))
			}
			if err := pb.CreateVtxFirmware(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Radio Firmwares
	if b, err := os.ReadFile(dir + "radio_firmwares.textproto"); err == nil {
		m := &pb.ListRadioFirmwaresResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.RadioFirmwares {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("radio_firmware %s: %w", v.Id, err))
			}
			if err := pb.CreateRadioFirmware(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Software Tools
	if b, err := os.ReadFile(dir + "software_tools.textproto"); err == nil {
		m := &pb.ListSoftwareToolsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.SoftwareTools {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("software_tool %s: %w", v.Id, err))
			}
			if err := pb.CreateSoftwareTool(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Transmitters
	if b, err := os.ReadFile(dir + "transmitters.textproto"); err == nil {
		m := &pb.ListTransmittersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Transmitters {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("transmitter %s: %w", v.Id, err))
			}
			if err := pb.CreateTransmitter(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Goggles
	if b, err := os.ReadFile(dir + "goggles.textproto"); err == nil {
		m := &pb.ListGogglesResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Goggles {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("goggles %s: %w", v.Id, err))
			}
			if err := pb.CreateGoggles(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Chargers
	if b, err := os.ReadFile(dir + "chargers.textproto"); err == nil {
		m := &pb.ListChargersResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Chargers {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("charger %s: %w", v.Id, err))
			}
			if err := pb.CreateCharger(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Field Tools
	if b, err := os.ReadFile(dir + "field_tools.textproto"); err == nil {
		m := &pb.ListFieldToolsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.FieldTools {
			if err := validation.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("field_tool %s: %w", v.Id, err))
			}
			if err := pb.CreateFieldTool(ctx, tx, v); err != nil {
				panic(err)
			}
		}
	}

	// Clean up IEEE float precision artifacts on numeric columns
	tx.Exec(ctx, "UPDATE batteries SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), max_current_a = ROUND(max_current_a, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE flight_controllers SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE electronic_speed_controllers SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), weight_g = ROUND(weight_g, 2);")
	tx.Exec(ctx, "UPDATE motors SET min_voltage = ROUND(min_voltage, 2), max_voltage = ROUND(max_voltage, 2), max_current_a = ROUND(max_current_a, 2), weight_g = ROUND(weight_g, 2);")

	// Populate unified components registry table
	componentTables := []struct {
		table      string
		collection string
	}{
		{"motors", "motors"},
		{"frames", "frames"},
		{"batteries", "batteries"},
		{"electronic_speed_controllers", "electronic-speed-controllers"},
		{"flight_controllers", "flight-controllers"},
		{"receivers", "receivers"},
		{"video_transmitters", "video-transmitters"},
		{"antennas", "antennas"},
		{"cameras", "cameras"},
		{"propellers", "propellers"},
		{"gps_receivers", "gps-receivers"},
		{"transmitters", "transmitters"},
		{"goggles", "goggles"},
		{"chargers", "chargers"},
		{"field_tools", "field-tools"},
		{"flight_controller_firmwares", "fc-firmware"},
		{"electronic_speed_controller_firmwares", "esc-firmware"},
		{"video_transmitter_firmwares", "vtx-firmware"},
		{"radio_firmwares", "radio-firmware"},
		{"software_tools", "software-tools"},
	}

	// Populate unified entities table from builds
	query := `
		INSERT INTO entities (uuid, collection, id, name, description, primary_display_image, reference_links, media, author_or_manufacturer, is_archived, created_at, updated_at)
		SELECT uuid, 'builds', id, name, description, primary_display_image, reference_links, media, COALESCE(author_username, ''), false, NOW(), NOW() FROM builds
		ON CONFLICT (collection, id) DO NOTHING;
	`
	if _, err := tx.Exec(ctx, query); err != nil {
		panic(fmt.Errorf("failed populating entities from builds: %w", err))
	}

	for _, ct := range componentTables {
		q := fmt.Sprintf(`
			INSERT INTO entities (uuid, collection, id, name, description, primary_display_image, reference_links, media, author_or_manufacturer, is_archived, created_at, updated_at)
			SELECT uuid, '%s', id, name, description, primary_display_image, reference_links, media, manufacturer, false, NOW(), NOW() FROM %s
			ON CONFLICT (collection, id) DO NOTHING;
		`, ct.collection, ct.table)
		if _, err := tx.Exec(ctx, q); err != nil {
			panic(fmt.Errorf("failed populating entities from %s: %w", ct.table, err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}
