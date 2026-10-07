//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"strings"
)

var domains = []struct {
	Name   string
	Plural string
}{
	{"Motor", "Motors"},
	{"Frame", "Frames"},
	{"Battery", "Batteries"},
	{"Esc", "Escs"},
	{"FlightController", "FlightControllers"},
	{"Receiver", "Receivers"},
	{"VideoTransmitter", "VideoTransmitters"},
	{"Antenna", "Antennas"},
	{"Camera", "Cameras"},
	{"Propeller", "Propellers"},
	{"Build", "Builds"},
}

func main() {
	f, _ := os.Create("main.go")
	defer f.Close()

	fmt.Fprintln(f, `package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
		"reflect"
	"regexp"
		"sort"
		"strings"
		"text/tabwriter"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

var (
	apiURL = "http://localhost:8080"
	jsonOut bool
	yamlOut bool
)

func newRootCmd() *cobra.Command {
	if url := os.Getenv("QS_API_URL"); url != "" {
		apiURL = url
	}

	rootCmd := &cobra.Command{
		Use:   "qs",
		Short: "Quadsmith CLI",
	}
	rootCmd.CompletionOptions.DisableDescriptions = true

	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output format as JSON")
	rootCmd.PersistentFlags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	}
`)

	for _, d := range domains {
		lowerName := strings.ToLower(d.Name)
		lowerPlural := strings.ToLower(d.Plural)

		fmt.Fprintf(f, "\n\t// --- %s ---\n", d.Plural)
		fmt.Fprintf(f, "\t%sClient := quadsmithconnect.New%sServiceClient(http.DefaultClient, apiURL)\n", lowerName, d.Name)
		fmt.Fprintf(f, "\t%sCmd := &cobra.Command{Use: \"%s\"}\n", lowerPlural, lowerPlural)

		fmt.Fprintf(f, "\t%sListCmd := &cobra.Command{\n", lowerPlural)
		fmt.Fprintf(f, "\t\tUse: \"list\",\n")
		fmt.Fprintf(f, "\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")
		fmt.Fprintf(f, "\t\t\tfilter, _ := cmd.Flags().GetString(\"filter\")\n")
		fmt.Fprintf(f, "\t\t\tcolumns, _ := cmd.Flags().GetStringSlice(\"column\")\n")
		fmt.Fprintf(f, "\t\t\tsortOpts, _ := cmd.Flags().GetStringSlice(\"sort\")\n")
		fmt.Fprintf(f, "\t\t\tpageSize, _ := cmd.Flags().GetInt32(\"page-size\")\n")
		fmt.Fprintf(f, "\t\t\tpageToken, _ := cmd.Flags().GetString(\"page-token\")\n")
		fmt.Fprintf(f, "\t\t\tif pageToken != \"\" {\n")
		fmt.Fprintf(f, "\t\t\t\tif cmd.Flags().Changed(\"filter\") && cmd.Flags().Changed(\"sort\") {\n")
		fmt.Fprintf(f, "\t\t\t\t\tcmd.SilenceUsage = false\n")
		fmt.Fprintf(f, "\t\t\t\t\treturn fmt.Errorf(\"cannot specify --filter or --sort when --page-token is provided\")\n")
		fmt.Fprintf(f, "\t\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\t\tif cmd.Flags().Changed(\"filter\") {\n")
		fmt.Fprintf(f, "\t\t\t\t\tcmd.SilenceUsage = false\n")
		fmt.Fprintf(f, "\t\t\t\t\treturn fmt.Errorf(\"cannot specify --filter when --page-token is provided\")\n")
		fmt.Fprintf(f, "\t\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\t\tif cmd.Flags().Changed(\"sort\") {\n")
		fmt.Fprintf(f, "\t\t\t\t\tcmd.SilenceUsage = false\n")
		fmt.Fprintf(f, "\t\t\t\t\treturn fmt.Errorf(\"cannot specify --sort when --page-token is provided\")\n")
		fmt.Fprintf(f, "\t\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\tif (jsonOut || yamlOut) && pageSize == 0 {\n")
		fmt.Fprintf(f, "\t\t\t\tpageSize = 100\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\tvar all []*pb.%s\n", d.Name)
		fmt.Fprintf(f, "\t\t\tcurrentToken := pageToken\n")
		fmt.Fprintf(f, "\t\t\tvar lastNextPageToken string\n")
		fmt.Fprintf(f, "\t\t\tfor {\n")
		fmt.Fprintf(f, "\t\t\t\treq := &pb.List%sRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}\n", d.Plural)
		fmt.Fprintf(f, "\t\t\t\tres, err := %sClient.List%s(context.Background(), connect.NewRequest(req))\n", lowerName, d.Plural)
		fmt.Fprintf(f, "\t\t\t\tif err != nil { return err }\n")
		fmt.Fprintf(f, "\t\t\t\tall = append(all, res.Msg.%s...)\n", d.Plural)
		fmt.Fprintf(f, "\t\t\t\tlastNextPageToken = res.Msg.NextPageToken\n")
		fmt.Fprintf(f, "\t\t\t\tif !(jsonOut || yamlOut) || res.Msg.NextPageToken == \"\" {\n")
		fmt.Fprintf(f, "\t\t\t\t\tbreak\n")
		fmt.Fprintf(f, "\t\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\t\tcurrentToken = res.Msg.NextPageToken\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\tif len(columns) == 0 {\n")
		fmt.Fprintf(f, "\t\t\t\tcolumns = GetDefaultColumns(&pb.%s{})\n", d.Name)
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\terr := printOutput(all, columns, parsePageOffset(pageToken))\n")
		fmt.Fprintf(f, "\t\t\tif err != nil { return err }\n")
		fmt.Fprintf(f, "\t\t\tif lastNextPageToken != \"\" && !jsonOut && !yamlOut {\n")
		fmt.Fprintf(f, "\t\t\t\tfmt.Printf(\"\\nNext page token: %%s\\n\", lastNextPageToken)\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\treturn nil\n")
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t}\n")
		fmt.Fprintf(f, "\t%sListCmd.Flags().StringP(\"filter\", \"f\", \"\", \"CEL filter string\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sListCmd.Flags().Int32P(\"page-size\", \"p\", 0, \"Maximum number of items to return\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sListCmd.Flags().String(\"page-token\", \"\", \"Page token for next page of results\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sListCmd.RegisterFlagCompletionFunc(\"filter\", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n", lowerPlural)
		fmt.Fprintf(f, "\t\tcols := GetColumns(&pb.%s{})\n", d.Name)
		fmt.Fprintf(f, "\t\tre := regexp.MustCompile(`([a-zA-Z_]+)$`)\n")
		fmt.Fprintf(f, "\t\tmatch := re.FindStringSubmatch(toComplete)\n")
		fmt.Fprintf(f, "\t\tprefix := \"\"\n")
		fmt.Fprintf(f, "\t\tbase := toComplete\n")
		fmt.Fprintf(f, "\t\tif len(match) > 0 {\n")
		fmt.Fprintf(f, "\t\t\tprefix = match[1]\n")
		fmt.Fprintf(f, "\t\t\tbase = toComplete[:len(toComplete)-len(prefix)]\n")
		fmt.Fprintf(f, "\t\t}\n")
		fmt.Fprintf(f, "\t\tvar filtered []string\n")
		fmt.Fprintf(f, "\t\tfor _, c := range cols {\n")
		fmt.Fprintf(f, "\t\t\tif len(prefix) == 0 || strings.HasPrefix(c, prefix) {\n")
		fmt.Fprintf(f, "\t\t\t\tfiltered = append(filtered, base + c)\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t}\n")
		fmt.Fprintf(f, "\t\treturn filtered, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t})\n")
		fmt.Fprintf(f, "\t%sListCmd.Flags().StringSliceP(\"column\", \"c\", nil, \"Columns to select\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sListCmd.Flags().StringSliceP(\"sort\", \"s\", nil, \"Columns to sort by (e.g. ^kv)\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sListCmd.RegisterFlagCompletionFunc(\"column\", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n", lowerPlural)
		fmt.Fprintf(f, "\t\tcols := GetColumns(&pb.%s{})\n", d.Name)
		fmt.Fprintf(f, "\t\tselected, _ := cmd.Flags().GetStringSlice(\"column\")\n")
		fmt.Fprintf(f, "\t\tselectedMap := make(map[string]bool)\n")
		fmt.Fprintf(f, "\t\tfor _, s := range selected { selectedMap[s] = true }\n")
		fmt.Fprintf(f, "\t\tvar filtered []string\n")
		fmt.Fprintf(f, "\t\tfor _, c := range cols {\n")
		fmt.Fprintf(f, "\t\t\tif !selectedMap[c] && strings.HasPrefix(c, toComplete) {\n")
		fmt.Fprintf(f, "\t\t\t\tfiltered = append(filtered, c)\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t}\n")
		fmt.Fprintf(f, "\t\treturn filtered, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t})\n")

		fmt.Fprintf(f, "\t%sListCmd.RegisterFlagCompletionFunc(\"sort\", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n", lowerPlural)
		fmt.Fprintf(f, "\t\tcols := GetColumns(&pb.%s{})\n", d.Name)
		fmt.Fprintf(f, "\t\tselected, _ := cmd.Flags().GetStringSlice(\"sort\")\n")
		fmt.Fprintf(f, "\t\tselectedMap := make(map[string]bool)\n")
		fmt.Fprintf(f, "\t\tfor _, s := range selected { selectedMap[strings.TrimPrefix(s, \"^\")] = true }\n")
		fmt.Fprintf(f, "\t\tisDesc := strings.HasPrefix(toComplete, \"^\")\n")
		fmt.Fprintf(f, "\t\tcleanPrefix := strings.TrimPrefix(toComplete, \"^\")\n")
		fmt.Fprintf(f, "\t\tvar filtered []string\n")
		fmt.Fprintf(f, "\t\tfor _, c := range cols {\n")
		fmt.Fprintf(f, "\t\t\tif !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {\n")
		fmt.Fprintf(f, "\t\t\t\tif isDesc {\n")
		fmt.Fprintf(f, "\t\t\t\t\tfiltered = append(filtered, \"^\" + c)\n")
		fmt.Fprintf(f, "\t\t\t\t} else {\n")
		fmt.Fprintf(f, "\t\t\t\t\tfiltered = append(filtered, c)\n")
		fmt.Fprintf(f, "\t\t\t\t\tfiltered = append(filtered, \"^\" + c)\n")
		fmt.Fprintf(f, "\t\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t}\n")
		fmt.Fprintf(f, "\t\treturn filtered, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t})\n")
		fmt.Fprintf(f, "\t%sCmd.AddCommand(%sListCmd)\n\n", lowerPlural, lowerPlural)
		// removed
		fmt.Fprintf(f, "\t%sGetCmd := &cobra.Command{\n", lowerPlural)
		fmt.Fprintf(f, "\t\tUse: \"get [id]\",\n")
		fmt.Fprintf(f, "\t\tArgs: cobra.ExactArgs(1),\n")
		fmt.Fprintf(f, "\t\tValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n")
		fmt.Fprintf(f, "\t\t\tif len(args) != 0 { return nil, cobra.ShellCompDirectiveNoFileComp }\n")
		fmt.Fprintf(f, "\t\t\tres, err := %sClient.List%s(context.Background(), connect.NewRequest(&pb.List%sRequest{PageSize: 100}))\n", lowerName, d.Plural, d.Plural)
		fmt.Fprintf(f, "\t\t\tif err != nil { return nil, cobra.ShellCompDirectiveError }\n")
		fmt.Fprintf(f, "\t\t\tvar comps []string\n")
		fmt.Fprintf(f, "\t\t\tfor _, item := range res.Msg.%s {\n", d.Plural)
		fmt.Fprintf(f, "\t\t\t\tcomps = append(comps, item.Id)\n") // Assuming every domain has an 'Id' field
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\treturn comps, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")
		fmt.Fprintf(f, "\t\t\tcolumns, _ := cmd.Flags().GetStringSlice(\"column\")\n")
		fmt.Fprintf(f, "\t\t\treq := &pb.Get%sRequest{Id: args[0], Columns: columns}\n", d.Name)
		fmt.Fprintf(f, "\t\t\tres, err := %sClient.Get%s(context.Background(), connect.NewRequest(req))\n", lowerName, d.Name)
		fmt.Fprintf(f, "\t\t\tif err != nil { return err }\n")
		fmt.Fprintf(f, "\t\t\treturn printOutput(res.Msg, columns)\n")
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t}\n")
		fmt.Fprintf(f, "\t%sGetCmd.Flags().StringSliceP(\"column\", \"c\", nil, \"Columns to select\")\n", lowerPlural)
		fmt.Fprintf(f, "\t%sGetCmd.RegisterFlagCompletionFunc(\"column\", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n", lowerPlural)
		fmt.Fprintf(f, "\t\tcols := GetColumns(&pb.%s{})\n", d.Name)
		fmt.Fprintf(f, "\t\tselected, _ := cmd.Flags().GetStringSlice(\"column\")\n")
		fmt.Fprintf(f, "\t\tselectedMap := make(map[string]bool)\n")
		fmt.Fprintf(f, "\t\tfor _, s := range selected { selectedMap[s] = true }\n")
		fmt.Fprintf(f, "\t\tvar filtered []string\n")
		fmt.Fprintf(f, "\t\tfor _, c := range cols {\n")
		fmt.Fprintf(f, "\t\t\tif !selectedMap[c] && strings.HasPrefix(c, toComplete) {\n")
		fmt.Fprintf(f, "\t\t\t\tfiltered = append(filtered, c)\n")
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t}\n")
		fmt.Fprintf(f, "\t\treturn filtered, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t})\n")
		fmt.Fprintf(f, "\t%sCmd.AddCommand(%sGetCmd)\n\n", lowerPlural, lowerPlural)

		fmt.Fprintf(f, "\trootCmd.AddCommand(%sCmd)\n", lowerPlural)
	}

	fmt.Fprintln(f, `
	// --- EVALUATOR ---
	evalClient := quadsmithconnect.NewEvaluatorServiceClient(http.DefaultClient, apiURL)
	evalCmd := &cobra.Command{
		Use: "evaluate [build-id]",
		Short: "Run physics estimation and compatibility checks on a build",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 { return nil, cobra.ShellCompDirectiveNoFileComp }
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(&pb.ListBuildsRequest{PageSize: 100}))
			if err != nil { return nil, cobra.ShellCompDirectiveError }
			var comps []string
			for _, item := range res.Msg.Builds {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bReq := &pb.GetBuildRequest{Id: args[0]}
			bRes, err := buildClient.GetBuild(context.Background(), connect.NewRequest(bReq))
			if err != nil { return fmt.Errorf("failed to fetch build: %w", err) }
			
			payload, _ := cmd.Flags().GetFloat32("payload")
			eReq := &pb.EvaluateBuildRequest{
				Build: bRes.Msg,
				PayloadWeightG: payload,
			}
			eRes, err := evalClient.EvaluateBuild(context.Background(), connect.NewRequest(eReq))
			if err != nil { return fmt.Errorf("evaluation failed: %w", err) }
			
			return printOutput(eRes.Msg, nil)
		},
	}
	evalCmd.Flags().Float32("payload", 0, "Payload weight in grams")
	rootCmd.AddCommand(evalCmd)

	// --- CUSTOM COMPLETION ---
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate the autocompletion script for the specified shell",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				err := rootCmd.GenBashCompletionV2(os.Stdout, true)
				if err == nil {
					// Dynamically bind to the exact path used to invoke the binary
					if os.Args[0] != "qs" {
						fmt.Printf("\ncomplete -o default -o nospace -F __start_qs %q\n", os.Args[0])
					}
					// Also support dynamic loading via bash-completion
					fmt.Println("if [[ -n \"$1\" && \"$1\" != \"qs\" && \"$1\" != \"\" ]]; then complete -o default -o nospace -F __start_qs \"$1\"; fi")
				}
				return err
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return nil
			}
		},
	}
	rootCmd.AddCommand(completionCmd)

	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}


func GetDefaultColumns(m proto.Message) []string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, pb.E_DefaultColumns) {
		if cols, ok := proto.GetExtension(opts, pb.E_DefaultColumns).([]string); ok && len(cols) > 0 {
			return cols
		}
	}
	return nil
}

func GetColumns(m interface{}) []string {
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
func parsePageOffset(tokenStr string) int {
	if tokenStr == "" {
		return 0
	}
	data, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(tokenStr)
		if err != nil {
			data, err = base64.StdEncoding.DecodeString(tokenStr)
			if err != nil {
				return 0
			}
		}
	}
	var t map[string]interface{}
	if err := json.Unmarshal(data, &t); err == nil {
		if off, ok := t["offset"].(float64); ok && off >= 0 {
			return int(off)
		}
	}
	return 0
}

func printOutput(data interface{}, cols []string, startOffset ...int) error {
	var isNilSlice bool
	if data != nil {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			isNilSlice = true
		}
	}

	if jsonOut {
		if isNilSlice {
			fmt.Println("[]")
			return nil
		}
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	} else if yamlOut {
		if isNilSlice {
			fmt.Println("[]")
			return nil
		}
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if data == nil || isNilSlice {
		fmt.Println("No records found.")
		return nil
	}

	printTable(data, cols, startOffset...)
	return nil
}

func printTable(data interface{}, cols []string, startOffset ...int) {
	printTableTo(os.Stdout, data, cols, startOffset...)
}

func printTableTo(out io.Writer, data interface{}, cols []string, startOffset ...int) {
	offset := 0
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}

	b, _ := json.Marshal(data)
	var v interface{}
	json.Unmarshal(b, &v)

	if v == nil {
		fmt.Fprintln(out, "No records found.")
		return
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	
	switch val := v.(type) {
	case []interface{}:
		if len(val) == 0 {
			fmt.Fprintln(out, "No records found.")
			return
		}
		
		var orderedKeys []string
		if len(cols) > 0 {
			orderedKeys = cols
		} else {
			keyMap := make(map[string]bool)
			for _, item := range val {
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
			
			for _, priority := range []string{"id", "name"} {
				if keyMap[priority] {
					orderedKeys = append(orderedKeys, priority)
				}
			}
			for _, k := range keys {
				if k != "id" && k != "name" {
					orderedKeys = append(orderedKeys, k)
				}
			}
		}

		startIdx := offset + 1
		fmt.Fprintf(w, "#")
		if len(orderedKeys) > 0 {
			fmt.Fprintf(w, "\t")
		}
		for i, k := range orderedKeys {
			fmt.Fprintf(w, "%s", strings.ToUpper(k))
			if i < len(orderedKeys)-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintln(w)
		for idx, item := range val {
			fmt.Fprintf(w, "%d", startIdx+idx)
			if len(orderedKeys) > 0 {
				fmt.Fprintf(w, "\t")
			}
			if m, ok := item.(map[string]interface{}); ok {
				for i, k := range orderedKeys {
					if m[k] == nil {
						fmt.Fprintf(w, "-")
					} else {
						fmt.Fprintf(w, "%v", m[k])
					}
					if i < len(orderedKeys)-1 {
						fmt.Fprintf(w, "\t")
					}
				}
			} else {
				fmt.Fprintf(w, "%v", item)
			}
			fmt.Fprintln(w)
		}
	case map[string]interface{}:
		var keys []string
		if len(cols) > 0 {
			keys = cols
		} else {
			for k := range val {
				keys = append(keys, k)
			}
			sort.Strings(keys)
		}
		for _, k := range keys {
			if val[k] != nil {
				fmt.Fprintf(w, "%s\t%v\n", k, val[k])
			} else {
				fmt.Fprintf(w, "%s\t-\n", k)
			}
		}
	default:
		fmt.Fprintf(out, "%v\n", v)
	}
	w.Flush()
}
`)
}
