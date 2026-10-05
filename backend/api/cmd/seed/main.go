package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"

	pb "quadsmith/api/gen/quadsmith"
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

	// Read the textproto
	path := os.Getenv("SEED_FILE_PATH")
	if path == "" {
		path = "../db/seeds/components.textproto"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read textproto: %v", err)
	}

	var seedList pb.ComponentSeedList
	if err := prototext.Unmarshal(data, &seedList); err != nil {
		log.Fatalf("Failed to parse textproto: %v", err)
	}

	fmt.Printf("Parsed %d components. Seeding database...\n", len(seedList.GetComponents()))

	for _, comp := range seedList.GetComponents() {
		// Convert the profile oneof to JSONB
		// We have to extract just the oneof value to store it as JSONB
		// To do this reliably, we can marshal the whole component to JSON, then extract the inner profile field.
		// For MVP, we use protojson to marshal the Component, which contains the oneof field.

		jsonBytes, err := protojson.MarshalOptions{
			EmitUnpopulated: false,
			UseProtoNames:   true,
		}.Marshal(comp)
		if err != nil {
			log.Printf("Failed to marshal %s: %v", comp.GetResource().GetId(), err)
			continue
		}

		// The JSON looks like {"id": "...", "name": "...", "motor": {"kv_rating": 2400}}
		// Our table wants just the profile data in `data`. For MVP, storing the whole JSON is fine since our mapping works on `data->'motor'`.

		// 1. Check if resource exists, or generate new UUID
		var newUuid string
		err = pool.QueryRow(context.Background(), `SELECT uuid FROM resources WHERE id = $1 LIMIT 1`, comp.GetResource().GetId()).Scan(&newUuid)
		if err != nil {
			u, _ := uuid.NewV7()
			newUuid = u.String()
		}

		tx, err := pool.Begin(context.Background())
		if err != nil {
			log.Printf("Failed to begin transaction for %s: %v", comp.GetResource().GetId(), err)
			continue
		}

		// 1. Upsert base resource
		resourceQuery := `
			INSERT INTO resources (uuid, id, resource_type) 
			VALUES ($1, $2, 'COMPONENT') 
			ON CONFLICT (uuid) DO UPDATE SET resource_type = EXCLUDED.resource_type
		`
		if _, err = tx.Exec(context.Background(), resourceQuery, newUuid, comp.GetResource().GetId()); err != nil {
			log.Printf("Failed to upsert resource %s: %v", comp.GetResource().GetId(), err)
			tx.Rollback(context.Background())
			continue
		}

		// 2. Upsert component data
		query := `
			INSERT INTO components (uuid, type, name, data)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (uuid) DO UPDATE SET data = EXCLUDED.data, name = EXCLUDED.name, type = EXCLUDED.type
		`
		cType := getComponentTypeString(comp)
		if _, err = tx.Exec(context.Background(), query, newUuid, cType, comp.GetResource().GetName(), string(jsonBytes)); err != nil {
			log.Printf("Failed to upsert component %s: %v", comp.GetResource().GetId(), err)
			tx.Rollback(context.Background())
			continue
		}

		if err := tx.Commit(context.Background()); err != nil {
			log.Printf("Failed to commit transaction for %s: %v", comp.GetResource().GetId(), err)
		}
	}
	fmt.Println("Components seeding complete!")

	// -----------------------------------------
	// Seed Builds
	// -----------------------------------------
	buildsPath := os.Getenv("BUILDS_SEED_FILE_PATH")
	if buildsPath == "" {
		buildsPath = "../db/seeds/builds.textproto"
	}
	buildsData, err := os.ReadFile(buildsPath)
	if err == nil {
		var buildSeedList pb.BuildSeedList
		if err := prototext.Unmarshal(buildsData, &buildSeedList); err != nil {
			log.Fatalf("Failed to parse builds textproto: %v", err)
		}

		fmt.Printf("Parsed %d builds. Seeding database...\n", len(buildSeedList.GetBuilds()))

		for _, b := range buildSeedList.GetBuilds() {
			var newUuid string
			err = pool.QueryRow(context.Background(), `SELECT uuid FROM resources WHERE id = $1 LIMIT 1`, b.GetResource().GetId()).Scan(&newUuid)
			if err != nil {
				u, _ := uuid.NewV7()
				newUuid = u.String()
			}

			tx, err := pool.Begin(context.Background())
			if err != nil {
				log.Printf("Failed to begin transaction for build %s: %v", b.GetResource().GetId(), err)
				continue
			}

			// 1. Upsert base resource
			resourceQuery := `
				INSERT INTO resources (uuid, id, resource_type) 
				VALUES ($1, $2, 'BUILD') 
				ON CONFLICT (uuid) DO UPDATE SET resource_type = EXCLUDED.resource_type
			`
			if _, err = tx.Exec(context.Background(), resourceQuery, newUuid, b.GetResource().GetId()); err != nil {
				log.Printf("Failed to upsert resource %s: %v", b.GetResource().GetId(), err)
				tx.Rollback(context.Background())
				continue
			}

			// 2. Upsert build data
			jsonBytes, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(b)
			if err != nil {
				log.Printf("Failed to marshal build %s: %v", b.GetResource().GetId(), err)
				tx.Rollback(context.Background())
				continue
			}

			query := `
				INSERT INTO builds (
					uuid, data,
					frame_uuid, motor_uuid, propeller_uuid, camera_uuid, gps_uuid
				) VALUES ($1, $2, 
					(SELECT uuid FROM resources WHERE (id = $3 OR uuid::text = $3) AND resource_type = 'COMPONENT' LIMIT 1),
					(SELECT uuid FROM resources WHERE (id = $4 OR uuid::text = $4) AND resource_type = 'COMPONENT' LIMIT 1),
					(SELECT uuid FROM resources WHERE (id = $5 OR uuid::text = $5) AND resource_type = 'COMPONENT' LIMIT 1),
					(SELECT uuid FROM resources WHERE (id = $6 OR uuid::text = $6) AND resource_type = 'COMPONENT' LIMIT 1),
					(SELECT uuid FROM resources WHERE (id = $7 OR uuid::text = $7) AND resource_type = 'COMPONENT' LIMIT 1)
				)
				ON CONFLICT (uuid) DO UPDATE SET 
					data = EXCLUDED.data,
					frame_uuid = EXCLUDED.frame_uuid, motor_uuid = EXCLUDED.motor_uuid, 
					propeller_uuid = EXCLUDED.propeller_uuid, camera_uuid = EXCLUDED.camera_uuid, 
					gps_uuid = EXCLUDED.gps_uuid
			`

			// Helper inline func for empty string to null pointer
			nullIfEmpty := func(s string) *string {
				if s == "" {
					return nil
				}
				return &s
			}

			if _, err = tx.Exec(context.Background(), query,
				newUuid, jsonBytes,
				nullIfEmpty(b.GetFrameId()), nullIfEmpty(b.GetMotorId()), nullIfEmpty(b.GetPropellerId()), nullIfEmpty(b.GetCameraId()), nullIfEmpty(b.GetGpsId()),
			); err != nil {
				log.Printf("Failed to upsert build %s: %v", b.GetResource().GetId(), err)
				tx.Rollback(context.Background())
				continue
			}

			if err := tx.Commit(context.Background()); err != nil {
				log.Printf("Failed to commit transaction for build %s: %v", b.GetResource().GetId(), err)
			}
		}
		fmt.Println("Builds seeding complete!")
	} else {
		fmt.Printf("Skipping builds seeding (could not read %s: %v)\n", buildsPath, err)
	}
}

func getComponentTypeString(c *pb.Component) string {
	if c == nil {
		return "UNSPECIFIED"
	}
	switch c.WhichType() {
	case pb.Component_Frame_case:
		return "FRAME"
	case pb.Component_Motor_case:
		return "MOTOR"
	case pb.Component_Propeller_case:
		return "PROPELLER"
	case pb.Component_FlightController_case:
		return "FLIGHT_CONTROLLER"
	case pb.Component_Esc_case:
		return "ESC"
	case pb.Component_Battery_case:
		return "BATTERY"
	case pb.Component_Vtx_case:
		return "VTX"
	case pb.Component_Camera_case:
		return "CAMERA"
	case pb.Component_Receiver_case:
		return "RECEIVER"
	case pb.Component_Gps_case:
		return "GPS"
	case pb.Component_Antenna_case:
		return "ANTENNA"
	case pb.Component_Radio_case:
		return "RADIO"
	case pb.Component_Transmitter_case:
		return "TRANSMITTER"
	case pb.Component_Goggles_case:
		return "GOGGLES"
	case pb.Component_FcFirmware_case:
		return "FC_FIRMWARE"
	case pb.Component_EscFirmware_case:
		return "ESC_FIRMWARE"
	case pb.Component_VtxFirmware_case:
		return "VTX_FIRMWARE"
	case pb.Component_TransmitterFirmware_case:
		return "TRANSMITTER_FIRMWARE"
	case pb.Component_RadioModule_case:
		return "RADIO_MODULE"
	case pb.Component_RadioOs_case:
		return "RADIO_OS"
		return "TRANSMITTER_OS"
	default:
		return "UNSPECIFIED"
	}
}
