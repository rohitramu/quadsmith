package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

func protoToMap(msg proto.Message) map[string]any {
	b, _ := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(msg)
	var res map[string]any
	json.Unmarshal(b, &res)
	return res
}

func formatInlineMap(m map[string]any) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func getField(data map[string]any, path string) string {
	parts := strings.Split(path, ".")
	var current any = data
	for _, p := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[p]
	}
	if current == nil {
		return ""
	}

	switch v := current.(type) {
	case string:
		if strings.Contains(path, "id") && strings.Contains(v, "/") {
			parts := strings.Split(v, "/")
			return parts[len(parts)-1]
		}
		return v
	case map[string]any:

		return formatInlineMap(v)
	case []any:
		var formatted []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.Contains(path, "id") && strings.Contains(s, "/") {
				parts := strings.Split(s, "/")
				formatted = append(formatted, parts[len(parts)-1])
			} else {
				formatted = append(formatted, fmt.Sprintf("%v", item))
			}
		}
		return strings.Join(formatted, ", ")
	default:
		return fmt.Sprintf("%v", current)
	}
}

func resolveCols(rawCols []string, cType string) []string {
	cols := make([]string, 0, len(rawCols))
	prefix := ""
	if cType != "" {
		prefix = cType + "."
	}
	topLevelFields := map[string]bool{
		"uuid": true, "id": true, "type": true, "name": true,
		"manufacturer_uuid": true, "release_date": true, "weight_g": true,
		"company_ids": true, "reference_links": true,
	}
	for _, col := range rawCols {
		col = strings.TrimSpace(col)
		if !strings.Contains(col, ".") && prefix != "" && !topLevelFields[col] {
			col = prefix + col
		}
		cols = append(cols, col)
	}
	return cols
}

func printTable[T proto.Message](items []T, cols []string) {
	if len(items) == 0 {
		fmt.Println("No items found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	var headers []string
	for _, col := range cols {
		parts := strings.Split(col, ".")
		headers = append(headers, strings.ToUpper(parts[len(parts)-1]))
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, item := range items {
		m := protoToMap(item)
		var row []string
		for _, col := range cols {
			row = append(row, getField(m, col))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func formatOutput(msg proto.Message) string {
	b, err := protojson.MarshalOptions{Multiline: true, EmitUnpopulated: false}.Marshal(msg)
	if err != nil {
		return fmt.Sprintf("error marshaling output: %v", err)
	}

	if jsonOut {
		return string(b)
	}

	y, err := yaml.JSONToYAML(b)
	if err != nil {
		return fmt.Sprintf("error converting to yaml: %v", err)
	}
	return string(y)
}
