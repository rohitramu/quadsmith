package search

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/internal/cel2sql"
)

// CollectionTarget represents a resolved collection with its merged CEL filter.
type CollectionTarget struct {
	Collection *pb.SearchCollectionDef
	Filter     string
}

// ResolveTargets resolves and validates an incoming array of selectors into unique targets,
// merging duplicate paths with (filter1) && (filter2).
// If selectors is empty, returns all registered collections with empty filters.
func ResolveTargets(selectors []*pb.SearchSelector) ([]CollectionTarget, error) {
	if len(selectors) == 0 {
		targets := make([]CollectionTarget, 0, len(pb.AllSearchCollections))
		for _, col := range pb.AllSearchCollections {
			targets = append(targets, CollectionTarget{
				Collection: col,
				Filter:     "",
			})
		}
		return targets, nil
	}

	filtersByPath := make(map[string][]string)
	colByPath := make(map[string]*pb.SearchCollectionDef)
	var canonicalOrder []string

	for _, sel := range selectors {
		path := strings.TrimSpace(sel.GetPath())
		if path == "" {
			return nil, fmt.Errorf("selector path cannot be empty")
		}

		col, ok := pb.LookupSearchCollection(path)
		if !ok {
			return nil, fmt.Errorf("unknown collection path: %q", path)
		}

		cPath := col.CanonicalPath
		if _, exists := colByPath[cPath]; !exists {
			colByPath[cPath] = col
			canonicalOrder = append(canonicalOrder, cPath)
		}

		filter := strings.TrimSpace(sel.GetFilter())
		if filter != "" {
			filtersByPath[cPath] = append(filtersByPath[cPath], filter)
		}
	}

	targets := make([]CollectionTarget, 0, len(canonicalOrder))
	for _, cPath := range canonicalOrder {
		col := colByPath[cPath]
		filters := filtersByPath[cPath]

		var mergedFilter string
		if len(filters) == 1 {
			mergedFilter = filters[0]
		} else if len(filters) > 1 {
			var wrapped []string
			for _, f := range filters {
				wrapped = append(wrapped, "("+f+")")
			}
			mergedFilter = strings.Join(wrapped, " && ")
		}

		targets = append(targets, CollectionTarget{
			Collection: col,
			Filter:     mergedFilter,
		})
	}

	return targets, nil
}

// Search executes fuzzy text search across the specified collection targets.
func Search(ctx context.Context, db *pgxpool.Pool, query string, selectors []*pb.SearchSelector, limit int32) ([]*pb.SearchResultItem, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)

	var g errgroup.Group
	resChan := make(chan []*pb.SearchResultItem, len(targets))

	for _, target := range targets {
		t := target
		g.Go(func() error {
			items, err := searchCollection(ctx, db, t, query, limit)
			if err != nil {
				return err
			}
			resChan <- items
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resChan)

	var allResults []*pb.SearchResultItem
	seenUUIDs := make(map[string]bool)

	for items := range resChan {
		for _, item := range items {
			if seenUUIDs[item.Uuid] {
				continue
			}
			seenUUIDs[item.Uuid] = true
			allResults = append(allResults, item)
		}
	}

	// Sort globally: match_score DESC, then name ASC, then id ASC
	sort.Slice(allResults, func(i, j int) bool {
		if allResults[i].MatchScore != allResults[j].MatchScore {
			return allResults[i].MatchScore > allResults[j].MatchScore
		}
		if allResults[i].Name != allResults[j].Name {
			return allResults[i].Name < allResults[j].Name
		}
		return allResults[i].Id < allResults[j].Id
	})

	if int32(len(allResults)) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

func searchCollection(ctx context.Context, db *pgxpool.Pool, target CollectionTarget, query string, limit int32) ([]*pb.SearchResultItem, error) {
	col := target.Collection
	tableName := col.TableName

	var whereClauses []string
	var args []any

	if target.Filter != "" {
		celWhere, celArgs, err := cel2sql.Compile(target.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid CEL filter for %s: %w", col.CanonicalPath, err)
		}
		if celWhere != "" {
			whereClauses = append(whereClauses, celWhere)
			args = append(args, celArgs...)
		}
	}

	var scoreExpr string

	if query == "" {
		scoreExpr = "1.0::float4"
	} else {
		qIdx := len(args) + 1
		pfxIdx := len(args) + 2
		wbIdx := len(args) + 3
		subIdx := len(args) + 4

		args = append(args, query)
		args = append(args, query+"%")
		args = append(args, `\y`+regexp.QuoteMeta(query))
		args = append(args, "%"+query+"%")

		scoreExpr = fmt.Sprintf(`(CASE
			WHEN LOWER(name) = LOWER($%d) OR id = $%d THEN 1.0::float4
			WHEN name ILIKE $%d OR id ILIKE $%d THEN 0.85::float4
			WHEN name ~* $%d THEN 0.75::float4
			WHEN name ILIKE $%d THEN 0.65::float4
			WHEN to_jsonb(t.*)::text ILIKE $%d THEN 0.45::float4
			ELSE (GREATEST(similarity(name, $%d), word_similarity($%d, name)) * 0.5)::float4
		END)`, qIdx, qIdx, pfxIdx, pfxIdx, wbIdx, subIdx, subIdx, qIdx, qIdx)

		textMatchCond := fmt.Sprintf(`(
			name ILIKE $%d
			OR id ILIKE $%d
			OR to_jsonb(t.*)::text ILIKE $%d
			OR similarity(name, $%d) > 0.15
			OR word_similarity($%d, name) > 0.25
		)`, subIdx, subIdx, subIdx, qIdx, qIdx)

		whereClauses = append(whereClauses, textMatchCond)
	}

	limitIdx := len(args) + 1
	args = append(args, limit)

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	querySQL := fmt.Sprintf(`SELECT
		uuid::text,
		id,
		name,
		COALESCE(description, ''),
		COALESCE(primary_display_image, ''),
		%s AS match_score,
		to_jsonb(t.*)
	FROM %s t
	%s
	ORDER BY match_score DESC, name ASC, id ASC
	LIMIT $%d`, scoreExpr, tableName, whereSQL, limitIdx)

	rows, err := db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query collection %s failed: %w", col.CanonicalPath, err)
	}
	defer rows.Close()

	var results []*pb.SearchResultItem
	for rows.Next() {
		var uuidStr, id, name, description, primaryImg string
		var matchScore float32
		var rawData []byte

		if err := rows.Scan(&uuidStr, &id, &name, &description, &primaryImg, &matchScore, &rawData); err != nil {
			return nil, fmt.Errorf("scan collection %s row failed: %w", col.CanonicalPath, err)
		}

		metadata := make(map[string]string)
		if len(rawData) > 0 {
			var rawMap map[string]any
			if err := json.Unmarshal(rawData, &rawMap); err == nil {
				for _, colName := range col.DefaultColumns {
					if val, ok := rawMap[colName]; ok && val != nil {
						metadata[colName] = fmt.Sprint(val)
					}
				}
				// Also include weight_g if available and not already included
				if val, ok := rawMap["weight_g"]; ok && val != nil && metadata["weight_g"] == "" {
					metadata["weight_g"] = fmt.Sprint(val)
				}
				// Also include manufacturer if available
				if val, ok := rawMap["manufacturer"]; ok && val != nil && metadata["manufacturer"] == "" {
					metadata["manufacturer"] = fmt.Sprint(val)
				}
			}
		}

		var pImg *string
		if primaryImg != "" {
			pImg = &primaryImg
		}

		results = append(results, &pb.SearchResultItem{
			Id:                  id,
			Uuid:                uuidStr,
			Name:                name,
			Path:                col.CanonicalPath,
			CollectionName:      col.DisplayName,
			PrimaryDisplayImage: pImg,
			Description:         description,
			MatchScore:          matchScore,
			Metadata:            metadata,
		})
	}

	return results, rows.Err()
}
