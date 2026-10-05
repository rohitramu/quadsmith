package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"

	pb "quadsmith/api/gen/quadsmith"
)

type BuildStore struct {
	db *pgxpool.Pool
}

func NewBuildStore(db *pgxpool.Pool) *BuildStore {
	return &BuildStore{
		db: db,
	}
}

func (s *BuildStore) ListBuilds(ctx context.Context, filter string, fieldMask []string) ([]*pb.Build, error) {
	dataProjection := buildJSONBProjection(fieldMask)
	// MVP: Simple list without CEL filtering for now
	query := fmt.Sprintf(`
		SELECT rb.id, %s,
		       cf.id, cm.id, cp.id, cc.id, cg.id
		FROM builds b
		JOIN resources rb ON b.uuid = rb.uuid
		LEFT JOIN resources cf ON b.frame_uuid = cf.uuid
		LEFT JOIN resources cm ON b.motor_uuid = cm.uuid
		LEFT JOIN resources cp ON b.propeller_uuid = cp.uuid
		LEFT JOIN resources cc ON b.camera_uuid = cc.uuid
		LEFT JOIN resources cg ON b.gps_uuid = cg.uuid
		LIMIT 50
	`, strings.ReplaceAll(dataProjection, "c.data", "b.data"))
	
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var results []*pb.Build
	for rows.Next() {
		var id string
		var dataBytes []byte
		var frame, motor, prop, cam, gps *string

		if err := rows.Scan(
			&id, &dataBytes,
			&frame, &motor, &prop, &cam, &gps,
		); err != nil {
			return nil, err
		}

		b := &pb.Build{}
		if len(dataBytes) > 0 {
			if err := protojson.Unmarshal(dataBytes, b); err != nil {
				continue
			}
		}
		
		b.SetId(id)
		if frame != nil { b.SetFrameId(*frame) }
		if motor != nil { b.SetMotorId(*motor) }
		if prop != nil { b.SetPropellerId(*prop) }
		if cam != nil { b.SetCameraId(*cam) }
		if gps != nil { b.SetGpsId(*gps) }

		results = append(results, b)
	}
	return results, nil
}

func (s *BuildStore) GetBuild(ctx context.Context, id string, fieldMask []string) (*pb.Build, error) {
	dataProjection := buildJSONBProjection(fieldMask)
	query := fmt.Sprintf(`
		SELECT rb.id, %s,
		       cf.id, cm.id, cp.id, cc.id, cg.id
		FROM builds b
		JOIN resources rb ON b.uuid = rb.uuid
		LEFT JOIN resources cf ON b.frame_uuid = cf.uuid
		LEFT JOIN resources cm ON b.motor_uuid = cm.uuid
		LEFT JOIN resources cp ON b.propeller_uuid = cp.uuid
		LEFT JOIN resources cc ON b.camera_uuid = cc.uuid
		LEFT JOIN resources cg ON b.gps_uuid = cg.uuid
		WHERE rb.id = $1 OR b.uuid::text = $1 OR rb.id LIKE '%%/' || $1
	`, strings.ReplaceAll(dataProjection, "c.data", "b.data"))

	b := &pb.Build{}
	var dataBytes []byte
	var frame, motor, prop, cam, gps *string

		err := s.db.QueryRow(ctx, query, id).Scan(
		&id, &dataBytes,
		&frame, &motor, &prop, &cam, &gps,
	)
	if err != nil {
		return nil, fmt.Errorf("build not found: %w", err)
	}

	b.SetId(id)

	if len(dataBytes) > 0 {
		if err := protojson.Unmarshal(dataBytes, b); err != nil {
			return nil, err
		}
	}

	if frame != nil { b.SetFrameId(*frame) }
	if motor != nil { b.SetMotorId(*motor) }
	if prop != nil { b.SetPropellerId(*prop) }
	if cam != nil { b.SetCameraId(*cam) }
	if gps != nil { b.SetGpsId(*gps) }

	return b, nil
}

func (s *BuildStore) CreateBuild(ctx context.Context, b *pb.Build) error {
	jsonBytes, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal build: %w", err)
	}

	newUuid, err := uuid.NewV7()
	if err != nil {
		return err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO resources (uuid, id, resource_type) VALUES ($1, $2, 'BUILD')`, newUuid, b.GetId())
	if err != nil {
		return err
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
	`
	_, err = tx.Exec(ctx, query,
		newUuid, jsonBytes,
		nullIfEmpty(b.GetFrameId()), nullIfEmpty(b.GetMotorId()), nullIfEmpty(b.GetPropellerId()), nullIfEmpty(b.GetCameraId()), nullIfEmpty(b.GetGpsId()),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *BuildStore) UpdateBuild(ctx context.Context, b *pb.Build) error {
	jsonBytes, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal build: %w", err)
	}

	query := `
		UPDATE builds SET
			data = $2,
			frame_uuid = (SELECT uuid FROM resources WHERE (id = $3 OR uuid::text = $3) AND resource_type = 'COMPONENT' LIMIT 1),
			motor_uuid = (SELECT uuid FROM resources WHERE (id = $4 OR uuid::text = $4) AND resource_type = 'COMPONENT' LIMIT 1),
			propeller_uuid = (SELECT uuid FROM resources WHERE (id = $5 OR uuid::text = $5) AND resource_type = 'COMPONENT' LIMIT 1),
			camera_uuid = (SELECT uuid FROM resources WHERE (id = $6 OR uuid::text = $6) AND resource_type = 'COMPONENT' LIMIT 1),
			gps_uuid = (SELECT uuid FROM resources WHERE (id = $7 OR uuid::text = $7) AND resource_type = 'COMPONENT' LIMIT 1)
		WHERE uuid = (SELECT uuid FROM resources WHERE id = $1 OR uuid::text = $1 OR id LIKE '%%/' || $1 LIMIT 1)
	`
	_, err = s.db.Exec(ctx, query,
		b.GetId(), jsonBytes,
		nullIfEmpty(b.GetFrameId()), nullIfEmpty(b.GetMotorId()), nullIfEmpty(b.GetPropellerId()), nullIfEmpty(b.GetCameraId()), nullIfEmpty(b.GetGpsId()),
	)
	return err
}

func (s *BuildStore) DeleteBuild(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM resources WHERE (id = $1 OR uuid::text = $1 OR id LIKE '%/' || $1) AND resource_type = 'BUILD'`, id)
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
