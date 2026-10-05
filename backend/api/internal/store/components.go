package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"

	"quadsmith/api/pkg/cel2sql"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"

	pb "quadsmith/api/gen/quadsmith"
)

type ComponentStore struct {
	db      *pgxpool.Pool
	celEnv  *cel.Env
	celComp *cel2sql.Compiler
}

func NewComponentStore(db *pgxpool.Pool) (*ComponentStore, error) {
	// Define the schema types for CEL to ensure the user's AST is strictly typed before compiling to SQL
	env, err := cel.NewEnv(
		cel.Variable("id", cel.StringType),
		cel.Variable("uuid", cel.StringType),
		cel.Variable("motor.kv_rating", cel.IntType),
		cel.Variable("frame.is_ducted", cel.BoolType),
		cel.Variable("vtx.input_voltage_max_v", cel.DoubleType),
		cel.Variable("type", cel.StringType),
		ext.Strings(),
	)
	if err != nil {
		return nil, err
	}

	compiler := cel2sql.NewCompiler(cel2sql.ConvertOptions{
		FieldMapper: func(path string) (string, error) {
			// Fast path for top-level relational columns
			if path == "type" {
				return "c.type", nil
			}
			if path == "id" {
				return "r.id", nil
			}
			if path == "uuid" {
				return "r.uuid::text", nil
			}

			// Map JSONB attributes safely
			parts := strings.Split(path, ".")
			if len(parts) == 2 {
				// Prevent SQL injection on JSON keys by ensuring they only contain alphanumeric/underscores
				for _, p := range parts {
					for _, char := range p {
						if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
							return "", fmt.Errorf("invalid character in field path: %s", path)
						}
					}
				}

				profileType := parts[0]
				field := parts[1]

				// Basic type inference for PostgreSQL casts based on field name suffix
				cast := "text"
				if strings.HasSuffix(field, "_v") || strings.HasSuffix(field, "_a") || strings.HasSuffix(field, "_mm") {
					cast = "numeric"
				} else if strings.HasSuffix(field, "_rating") || strings.HasSuffix(field, "_count") {
					cast = "numeric"
				} else if strings.HasPrefix(field, "is_") || strings.HasPrefix(field, "has_") {
					cast = "boolean"
				}

				return fmt.Sprintf("(c.data->'%s'->>'%s')::%s", profileType, field, cast), nil
			}
			return "", fmt.Errorf("unsupported field path: %s", path)
		},
	})

	return &ComponentStore{
		db:      db,
		celEnv:  env,
		celComp: compiler,
	}, nil
}

func buildJSONBProjection(fields []string) string {
	if len(fields) == 0 {
		return "c.data"
	}

	tree := make(map[string]any)
	for _, f := range fields {
		if f == "id" || f == "type" {
			continue // Handled at top level
		}
		parts := strings.Split(f, ".")
		current := tree
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = true
			} else {
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]any)
				}
				current = current[part].(map[string]any)
			}
		}
	}

	if len(tree) == 0 {
		return "c.data"
	}

	var buildSQL func(node map[string]any, path []string) string
	buildSQL = func(node map[string]any, path []string) string {
		var args []string
		for k, v := range node {
			args = append(args, fmt.Sprintf("'%s'", k))
			if sub, ok := v.(map[string]any); ok {
				newPath := append(path, k)
				args = append(args, fmt.Sprintf("jsonb_strip_nulls(%s)", buildSQL(sub, newPath)))
			} else {
				valPath := append(path, k)
				var pgPath string
				for j, p := range valPath {
					if j == 0 {
						pgPath = fmt.Sprintf("c.data->'%s'", p)
					} else {
						pgPath += fmt.Sprintf("->'%s'", p)
					}
				}
				args = append(args, pgPath)
			}
		}
		return fmt.Sprintf("jsonb_build_object(%s)", strings.Join(args, ", "))
	}

	return fmt.Sprintf("jsonb_strip_nulls(%s)", buildSQL(tree, nil))
}

func (s *ComponentStore) SearchComponents(ctx context.Context, filterCEL string, fieldMask []string) ([]*pb.Component, error) {
	dataProjection := buildJSONBProjection(fieldMask)
	query := fmt.Sprintf(`SELECT c.uuid, r.id, c.type, %s FROM components c JOIN resources r ON c.uuid = r.uuid`, dataProjection)
	var args []any

	if filterCEL != "" {
		// 1. Lex and Parse CEL to AST
		ast, issues := s.celEnv.Compile(filterCEL)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("invalid filter syntax: %w", issues.Err())
		}

		// 2. Compile AST to parameterized SQL
		whereSQL, sqlArgs, err := s.celComp.Compile(ast)
		if err != nil {
			return nil, fmt.Errorf("failed to compile filter: %w", err)
		}

		query += " WHERE " + whereSQL
		args = sqlArgs
	}

	query += " LIMIT 50" // Hardcoded pagination for MVP

	// 3. Execute dynamically built, perfectly parameterized SQL
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var results []*pb.Component
	for rows.Next() {
		var uuidStr string
		var id string
		var cType string
		var profileData []byte

		if err := rows.Scan(&uuidStr, &id, &cType, &profileData); err != nil {
			return nil, err
		}

		comp := &pb.Component{}

		// 4. Unmarshal JSONB directly into Protobuf definition
		if len(profileData) > 0 {
			err = protojson.Unmarshal(profileData, comp)
			if err != nil {
				// Log error, but continue (skip corrupted row in MVP)
				continue
			}
		}

		// Ensure the returned ID is always the human readable one
		if comp.GetResource() == nil { comp.SetResource(&pb.ResourceMetadata{}) }; comp.GetResource().SetId(id)

		results = append(results, comp)
	}

	return results, nil
}

func (s *ComponentStore) CreateComponent(ctx context.Context, comp *pb.Component) error {
	jsonBytes, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(comp)

	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	newUuid, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO resources (uuid, id, resource_type) VALUES ($1, $2, 'COMPONENT')`, newUuid, comp.GetResource().GetId())
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `INSERT INTO components (uuid, type, name, weight_g, data) VALUES ($1, $2, $3, $4, $5)`,
		newUuid, getComponentTypeString(comp), comp.GetResource().GetName(), comp.GetWeightG(), jsonBytes)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *ComponentStore) UpdateComponent(ctx context.Context, comp *pb.Component) error {
	jsonBytes, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(comp)

	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		UPDATE components SET type = $2, name = $3, weight_g = $4, data = $5 
		WHERE uuid = (SELECT uuid FROM resources WHERE (id = $1 OR uuid::text = $1 OR id LIKE '%/' || $1) AND resource_type = 'COMPONENT' LIMIT 1)
	`, comp.GetResource().GetId(), getComponentTypeString(comp), comp.GetResource().GetName(), comp.GetWeightG(), jsonBytes)
	return err
}

func (s *ComponentStore) DeleteComponent(ctx context.Context, idOrUuid string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM resources WHERE (id = $1 OR uuid::text = $1 OR id LIKE '%/' || $1) AND resource_type = 'COMPONENT'
	`, idOrUuid)
	return err
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
