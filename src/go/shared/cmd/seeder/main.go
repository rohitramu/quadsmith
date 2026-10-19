package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/prototext"

	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/authcontext"
	"quadsmith/shared/pkg/config"

	"google.golang.org/protobuf/types/known/timestamppb"
	pb "quadsmith/shared/gen/quadsmith/api"
	storepb "quadsmith/shared/gen/quadsmith/store"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to API server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			fmt.Fprintln(os.Stderr, "seeder: --config flag is required; no auto-resolution allowed")
			os.Exit(1)
		}
	}

	cfg, err := config.LoadAPIServerConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := config.ValidateSeeder(cfg.Database, cfg.GetEnvironment()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Database.GetUrl())
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

	dir := "test/sandbox/db/seed_data/"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		for _, fallback := range []string{
			"../../test/sandbox/db/seed_data/",
			"../../../test/sandbox/db/seed_data/",
			"test/seed_data/",
			"../../test/seed_data/",
			"../../../test/seed_data/",
		} {
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
			if err := storepb.CreateUser(ctx, tx, &storepb.User{
				Uuid:      userUUID,
				Id:        u.Id,
				CreatedAt: timestamppb.New(twoDaysAgo),
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}

			if err := storepb.CreateUserProfile(ctx, tx, &storepb.UserProfile{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				AvatarUrl: "",
				Bio:       u.Bio,
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}

			if err := storepb.CreateUserEmail(ctx, tx, &storepb.UserEmail{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				Email:     u.Email,
				IsPrimary: true,
				CreatedAt: timestamppb.New(twoDaysAgo),
			}); err != nil {
				panic(err)
			}

			if err := storepb.CreateUserReputation(ctx, tx, &storepb.UserReputation{
				Uuid:      authcontext.NewUUIDv7(),
				UserUuid:  userUUID,
				Score:     u.ReputationScore,
				UpdatedAt: timestamppb.New(now),
			}); err != nil {
				panic(err)
			}
		}
	}
	if _, err := auth.EnsureSystemUser(ctx, tx); err != nil {
		panic(fmt.Errorf("failed to seed @quadsmith system user: %w", err))
	}

	// Motors
	if b, err := os.ReadFile(dir + "motors.textproto"); err == nil {
		m := &pb.ListMotorsResponse{}
		if err := prototext.Unmarshal(b, m); err != nil {
			panic(err)
		}
		for _, v := range m.Motors {
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("motor %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "motors", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("frame %s: %w", v.Id, err))
			}
			if v.MotorCount == 0 {
				v.MotorCount = 4
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "frames", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("battery %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "batteries", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("electronic speed controller %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "electronic-speed-controllers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("antenna %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "antennas", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("camera %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "cameras", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("propeller %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "propellers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("receiver %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "receivers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("video transmitter %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "video-transmitters", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("flight controller %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "flight-controllers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("gps receiver %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "gps-receivers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("build %s: %w", v.Id, err))
			}
			authorStr := ""
			if v.AuthorUsername != nil {
				authorStr = *v.AuthorUsername
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "builds", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, authorStr, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("fc_firmware %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "fc-firmware", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("esc_firmware %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "esc-firmware", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("vtx_firmware %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "vtx-firmware", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("radio_firmware %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "radio-firmware", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("software_tool %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "software-tools", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("transmitter %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "transmitters", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("goggles %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "goggles", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("charger %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "chargers", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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
			if err := pb.ValidateReferenceLinks(v.ReferenceLinks); err != nil {
				panic(fmt.Errorf("field_tool %s: %w", v.Id, err))
			}
			if err := seedEntity(ctx, tx, &v.Uuid, "field-tools", v.Id, v.Name, v.Description, v.PrimaryDisplayImage, v.ReferenceLinks, v.Media, v.Manufacturer, v.UserUuid, v.AuthorUsername); err != nil {
				panic(err)
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

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}

const (
	defaultSeederUserUUID = "018f0000-0000-7000-8000-000000000010"
	defaultSeederUsername = "alex"
)

func seedEntity(ctx context.Context, tx pgx.Tx, uuidPtr *string, collection, id, name, description string, primaryDisplayImage *string, refLinks []*pb.ReferenceLink, media []*pb.Media, authorOrMfr string, userUUID *string, authorUsername *string) error {
	if *uuidPtr == "" {
		*uuidPtr = authcontext.NewUUIDv7()
	}
	if userUUID == nil || *userUUID == "" {
		u := defaultSeederUserUUID
		userUUID = &u
	}
	if authorUsername == nil || *authorUsername == "" {
		a := defaultSeederUsername
		authorUsername = &a
	}
	if authorOrMfr == "" && authorUsername != nil {
		authorOrMfr = *authorUsername
	}
	var refLinksJSON []byte
	var err error
	if len(refLinks) > 0 {
		refLinksJSON, err = json.Marshal(refLinks)
		if err != nil {
			return err
		}
	}
	var mediaJSON []byte
	if len(media) > 0 {
		mediaJSON, err = json.Marshal(media)
		if err != nil {
			return err
		}
	}
	query := `
		INSERT INTO entities (uuid, collection, id, name, description, primary_display_image, reference_links, media, author_or_manufacturer, user_uuid, is_archived, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, NOW(), NOW())
		ON CONFLICT (collection, id) DO UPDATE
		SET name = EXCLUDED.name, description = EXCLUDED.description, primary_display_image = EXCLUDED.primary_display_image, author_or_manufacturer = EXCLUDED.author_or_manufacturer, user_uuid = EXCLUDED.user_uuid, updated_at = NOW()
	`
	_, err = tx.Exec(ctx, query, *uuidPtr, collection, id, name, description, primaryDisplayImage, refLinksJSON, mediaJSON, authorOrMfr, userUUID)
	return err
}
