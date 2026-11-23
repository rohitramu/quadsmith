package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sigs.k8s.io/yaml"

	"quadsmith/shared/gen/quadsmith/common"
)

type opsBlobUploadResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	MIMEType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	IsDuplicate bool   `json:"is_duplicate,omitempty"`
}

func newOpsBlobCmd() *cobra.Command {
	blobCmd := &cobra.Command{
		Use:   "blob",
		Short: "Upload binary blobs for the component catalog (requires active HUMAN JIT elevation)",
	}

	uploadCmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a catalog image, 3D model, or video via /api/ops/blobs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", filepath.Base(filePath))
			if err != nil {
				return err
			}
			if _, err := io.Copy(part, file); err != nil {
				return err
			}
			if err := writer.Close(); err != nil {
				return err
			}

			endpoint := strings.TrimRight(getOpsServerURL(), "/") + "/api/ops/blobs"
			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, endpoint, &body)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := getOpsHTTPClient().Do(req)
			if err != nil {
				return fmt.Errorf("upload request failed: %w", err)
			}
			defer resp.Body.Close()

			respBytes, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				var errMap map[string]string
				if json.Unmarshal(respBytes, &errMap) == nil && errMap["error"] != "" {
					return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errMap["error"])
				}
				return fmt.Errorf("upload failed with HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
			}

			var res opsBlobUploadResponse
			if err := json.Unmarshal(respBytes, &res); err != nil {
				return fmt.Errorf("failed parsing server response: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✓ Catalog blob uploaded successfully!")
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID:\t%s\n", res.ID)
			fmt.Fprintf(w, "URL:\t%s\n", res.URL)
			fmt.Fprintf(w, "Filename:\t%s\n", res.Filename)
			fmt.Fprintf(w, "MIME Type:\t%s\n", res.MIMEType)
			fmt.Fprintf(w, "Size:\t%d bytes\n", res.SizeBytes)
			return w.Flush()
		},
	}

	blobCmd.AddCommand(uploadCmd)
	return blobCmd
}

func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func readSpecPayload(cmd *cobra.Command) ([]byte, bool, error) {
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "-" || (filePath == "" && hasStdinData()) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, false, err
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return nil, false, nil
		}
		if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
			converted, yErr := yaml.YAMLToJSON(data)
			if yErr == nil {
				data = converted
			}
		}
		return data, true, nil
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, false, err
		}
		if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
			data, err = yaml.YAMLToJSON(data)
			if err != nil {
				return nil, false, fmt.Errorf("invalid YAML: %w", err)
			}
		}
		return data, true, nil
	}
	return nil, false, nil
}

func protoFieldFlagName(fd protoreflect.FieldDescriptor) string {
	flagName := strings.ReplaceAll(string(fd.Name()), "_", "-")
	if flagName == "file" || flagName == "token" || flagName == "json" {
		return "spec-" + flagName
	}
	return flagName
}

func shouldExposeProtoFlag(fd protoreflect.FieldDescriptor) bool {
	name := string(fd.Name())
	if name == "uuid" || name == "is_archived" {
		return false
	}
	if fd.IsMap() || fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind || fd.Kind() == protoreflect.BytesKind {
		return false
	}
	return true
}

func registerProtoFlags(cmd *cobra.Command, m proto.Message, skipID bool) {
	fields := m.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if !shouldExposeProtoFlag(fd) {
			continue
		}
		if skipID && string(fd.Name()) == "id" {
			continue
		}
		flagName := protoFieldFlagName(fd)
		desc := fmt.Sprintf("Set %s", fd.Name())

		if fd.IsList() {
			switch fd.Kind() {
			case protoreflect.StringKind, protoreflect.EnumKind:
				cmd.Flags().StringSlice(flagName, nil, desc)
			case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
				cmd.Flags().Int32Slice(flagName, nil, desc)
			case protoreflect.FloatKind, protoreflect.DoubleKind:
				cmd.Flags().Float64Slice(flagName, nil, desc)
			}
			continue
		}

		switch fd.Kind() {
		case protoreflect.StringKind:
			cmd.Flags().String(flagName, "", desc)
		case protoreflect.BoolKind:
			cmd.Flags().Bool(flagName, false, desc)
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			cmd.Flags().Int32(flagName, 0, desc)
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			cmd.Flags().Int64(flagName, 0, desc)
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			cmd.Flags().Uint32(flagName, 0, desc)
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			cmd.Flags().Uint64(flagName, 0, desc)
		case protoreflect.FloatKind:
			cmd.Flags().Float32(flagName, 0, desc)
		case protoreflect.DoubleKind:
			cmd.Flags().Float64(flagName, 0, desc)
		case protoreflect.EnumKind:
			var names []string
			vals := fd.Enum().Values()
			for j := 0; j < vals.Len(); j++ {
				ev := vals.Get(j)
				if ev.Number() != 0 {
					names = append(names, string(ev.Name()))
				}
			}
			cmd.Flags().String(flagName, "", fmt.Sprintf("%s (%s)", desc, strings.Join(names, "|")))
			enumCopy := names
			_ = cmd.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				var matches []string
				upper := strings.ToUpper(toComplete)
				for _, n := range enumCopy {
					if strings.HasPrefix(n, upper) {
						matches = append(matches, n)
					}
				}
				return matches, cobra.ShellCompDirectiveNoFileComp
			})
		}
	}
}

func resolveEnumNumber(ed protoreflect.EnumDescriptor, input string) (protoreflect.EnumNumber, error) {
	cleaned := strings.TrimSpace(strings.ToUpper(strings.ReplaceAll(input, "-", "_")))
	vals := ed.Values()
	for i := 0; i < vals.Len(); i++ {
		ev := vals.Get(i)
		evName := string(ev.Name())
		if evName == cleaned || strings.HasSuffix(evName, "_"+cleaned) {
			return ev.Number(), nil
		}
	}
	return 0, fmt.Errorf("invalid %s value %q", ed.Name(), input)
}

func applyProtoFlags(cmd *cobra.Command, m proto.Message, skipID bool) (bool, error) {
	pr := m.ProtoReflect()
	fields := pr.Descriptor().Fields()
	changed := false

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if !shouldExposeProtoFlag(fd) {
			continue
		}
		if skipID && string(fd.Name()) == "id" {
			continue
		}
		flagName := protoFieldFlagName(fd)
		if !cmd.Flags().Changed(flagName) {
			continue
		}
		changed = true

		if fd.IsList() {
			list := pr.Mutable(fd).List()
			list.Truncate(0)
			switch fd.Kind() {
			case protoreflect.StringKind:
				vals, err := cmd.Flags().GetStringSlice(flagName)
				if err != nil {
					return false, err
				}
				for _, v := range vals {
					list.Append(protoreflect.ValueOfString(v))
				}
			case protoreflect.EnumKind:
				vals, err := cmd.Flags().GetStringSlice(flagName)
				if err != nil {
					return false, err
				}
				for _, v := range vals {
					num, err := resolveEnumNumber(fd.Enum(), v)
					if err != nil {
						return false, err
					}
					list.Append(protoreflect.ValueOfEnum(num))
				}
			case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
				vals, err := cmd.Flags().GetInt32Slice(flagName)
				if err != nil {
					return false, err
				}
				for _, v := range vals {
					list.Append(protoreflect.ValueOfInt32(v))
				}
			case protoreflect.FloatKind:
				vals, err := cmd.Flags().GetFloat64Slice(flagName)
				if err != nil {
					return false, err
				}
				for _, v := range vals {
					list.Append(protoreflect.ValueOfFloat32(float32(v)))
				}
			case protoreflect.DoubleKind:
				vals, err := cmd.Flags().GetFloat64Slice(flagName)
				if err != nil {
					return false, err
				}
				for _, v := range vals {
					list.Append(protoreflect.ValueOfFloat64(v))
				}
			}
			continue
		}

		switch fd.Kind() {
		case protoreflect.StringKind:
			v, err := cmd.Flags().GetString(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfString(v))
		case protoreflect.BoolKind:
			v, err := cmd.Flags().GetBool(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfBool(v))
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			v, err := cmd.Flags().GetInt32(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfInt32(v))
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			v, err := cmd.Flags().GetInt64(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfInt64(v))
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			v, err := cmd.Flags().GetUint32(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfUint32(v))
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			v, err := cmd.Flags().GetUint64(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfUint64(v))
		case protoreflect.FloatKind:
			v, err := cmd.Flags().GetFloat32(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfFloat32(v))
		case protoreflect.DoubleKind:
			v, err := cmd.Flags().GetFloat64(flagName)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfFloat64(v))
		case protoreflect.EnumKind:
			v, err := cmd.Flags().GetString(flagName)
			if err != nil {
				return false, err
			}
			num, err := resolveEnumNumber(fd.Enum(), v)
			if err != nil {
				return false, err
			}
			pr.Set(fd, protoreflect.ValueOfEnum(num))
		}
	}

	return changed, nil
}

func getDefaultColumns(m proto.Message) []string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, common.E_Frontend) {
		if front, ok := proto.GetExtension(opts, common.E_Frontend).(*common.FrontendOptions); ok && front != nil && len(front.DefaultColumns) > 0 {
			cols := append([]string{}, front.DefaultColumns...)
			hasID := false
			for _, c := range cols {
				if c == "id" {
					hasID = true
					break
				}
			}
			if !hasID {
				cols = append([]string{"id"}, cols...)
			}
			return cols
		}
	}
	return nil
}

func getColumns(m interface{}) []string {
	var cols []string
	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() || strings.HasPrefix(f.Name, "XXX_") {
			continue
		}
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" || jsonTag == "" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		cols = append(cols, name)
	}
	return cols
}

func dataToJSON(data interface{}) ([]byte, error) {
	if m, ok := data.(proto.Message); ok {
		return protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}.Marshal(m)
	}
	v := reflect.ValueOf(data)
	if v.IsValid() && v.Kind() == reflect.Slice {
		var parts [][]byte
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			if pm, ok := elem.(proto.Message); ok {
				b, err := protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}.Marshal(pm)
				if err != nil {
					return nil, err
				}
				parts = append(parts, b)
			} else {
				b, err := json.Marshal(elem)
				if err != nil {
					return nil, err
				}
				parts = append(parts, b)
			}
		}
		var sb strings.Builder
		sb.WriteString("[")
		for i, p := range parts {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.Write(p)
		}
		sb.WriteString("]")
		return []byte(sb.String()), nil
	}
	return json.Marshal(data)
}

func formatCatalogValue(val interface{}) string {
	if val == nil {
		return "-"
	}
	if s, ok := val.(string); ok && s == "" {
		return "-"
	}
	if arr, ok := val.([]interface{}); ok && len(arr) == 0 {
		return "-"
	}
	return fmt.Sprintf("%v", val)
}

func printCatalogGetOutput(out io.Writer, data interface{}, cols []string) error {
	if data == nil {
		fmt.Fprintln(out, "No record found.")
		return nil
	}

	b, err := dataToJSON(data)
	if err != nil {
		return err
	}
	var raw interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(cols) > 0 {
		if m, ok := raw.(map[string]interface{}); ok {
			filtered := make(map[string]interface{})
			for _, c := range cols {
				if val, exists := m[c]; exists {
					filtered[c] = val
				}
			}
			raw = filtered
		}
	}

	if jsonOut {
		indented, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(indented))
		return nil
	}

	yb, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	fmt.Fprint(out, string(yb))
	return nil
}

func printCatalogListOutput(out io.Writer, data interface{}, cols []string) error {
	var isNilSlice bool
	if data != nil {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			isNilSlice = true
		}
	}

	if jsonOut {
		if isNilSlice {
			fmt.Fprintln(out, "[]")
			return nil
		}
		b, err := dataToJSON(data)
		if err != nil {
			return err
		}
		var raw interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		indented, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(indented))
		return nil
	} else if yamlOut {
		if isNilSlice {
			fmt.Fprintln(out, "[]")
			return nil
		}
		b, err := dataToJSON(data)
		if err != nil {
			return err
		}
		var raw interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		yb, err := yaml.Marshal(raw)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(yb))
		return nil
	}

	if data == nil || isNilSlice {
		fmt.Fprintln(out, "No records found.")
		return nil
	}

	b, err := dataToJSON(data)
	if err != nil {
		return err
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) == 0 {
		fmt.Fprintln(out, "No records found.")
		return nil
	}

	var orderedKeys []string
	if len(cols) > 0 {
		orderedKeys = cols
	} else {
		keyMap := make(map[string]bool)
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				for k := range m {
					keyMap[k] = true
				}
			}
		}
		var keys []string
		for k := range keyMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, priority := range []string{"id", "name", "is_archived"} {
			if keyMap[priority] {
				orderedKeys = append(orderedKeys, priority)
			}
		}
		for _, k := range keys {
			if k != "id" && k != "name" && k != "is_archived" {
				orderedKeys = append(orderedKeys, k)
			}
		}
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	numWidth := len(fmt.Sprintf("%d", len(arr)))
	if len(orderedKeys) > 0 {
		fmt.Fprintf(w, "\t")
		for i, k := range orderedKeys {
			fmt.Fprintf(w, "%s", strings.ToUpper(k))
			if i < len(orderedKeys)-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintln(w)
	}
	for idx, item := range arr {
		fmt.Fprintf(w, "%*d", numWidth, idx+1)
		if len(orderedKeys) > 0 {
			fmt.Fprintf(w, "\t")
		}
		if m, ok := item.(map[string]interface{}); ok {
			for i, k := range orderedKeys {
				fmt.Fprintf(w, "%s", formatCatalogValue(m[k]))
				if i < len(orderedKeys)-1 {
					fmt.Fprintf(w, "\t")
				}
			}
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}
