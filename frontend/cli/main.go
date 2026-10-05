package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"

	"connectrpc.com/connect"
)

var (
	client  quadsmithconnect.QuadsmithAPIClient
	verbose bool
	jsonOut bool
	yamlOut bool
	columns string
)

// resolveCols resolves shorthand column names to their full protobuf paths.

// generateCompletions creates shell completion strings by extracting id and name
func generateCompletions[T proto.Message](items []T, toComplete string) []string {
	var completions []string
	for _, item := range items {
		m := protoToMap(item)
		id := getField(m, "id")
		name := getField(m, "name")
		parts := strings.Split(id, "/")
		slug := parts[len(parts)-1]
		if strings.HasPrefix(slug, toComplete) || strings.HasPrefix(id, toComplete) {
			completions = append(completions, fmt.Sprintf("%s\t%s", slug, name))
		}
	}
	return completions
}

// getClient ensures the API client is initialized. This is specifically needed
// because Cobra's ValidArgsFunction (autocompletion) bypasses PersistentPreRun.

// protoToMap converts a protobuf message to a map for dynamic field extraction

// getField dynamically extracts a field value using dot notation (e.g. "motor.kv_rating")

// printTable dynamically renders a list of protobuf messages as a table

// formatOutput formats a protobuf message as JSON or YAML based on the --json flag

// getMessageFields recursively extracts field paths (e.g., "id", "motor.kv_rating")

// createResourceCmd creates list and describe subcommands for any generic resource.
func main() {
	rootCmd := &cobra.Command{
		Use:   "cli",
		Short: "Quadsmith CLI",
		Long: `Quadsmith CLI for managing quadcopter components and evaluating builds.

To enable terminal auto-completion, run:
  Bash:  source <(bin/cli completion bash)
  Zsh:   source <(bin/cli completion zsh)

If you use the Makefile, you can permanently add the auto-generated files to your profile:
  Bash:  echo 'source '\"\$PWD\"'/bin/completion.bash' >> ~/.bashrc
  Zsh:   echo 'source '\"\$PWD\"'/bin/completion.zsh' >> ~/.zshrc`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			getClient()
		},
	}

	rootCmd.CompletionOptions.DisableDescriptions = true

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (raw requests/responses)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&yamlOut, "yaml", false, "Output in YAML format")
	rootCmd.PersistentFlags().StringVar(&columns, "columns", "", "Comma-separated list of columns to display")

	rootCmd.RegisterFlagCompletionFunc("columns", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var msg protoreflect.MessageDescriptor
		path := cmd.CommandPath()
		if strings.Contains(path, "components") || strings.Contains(path, "hardware") {
			msg = (&pb.Component{}).ProtoReflect().Descriptor()
		} else if strings.Contains(path, "builds") {
			msg = (&pb.Build{}).ProtoReflect().Descriptor()
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		allFields := getMessageFields(msg, "")

		if msg == (&pb.Component{}).ProtoReflect().Descriptor() {
			var filtered []string
			var allowedPrefixes []string

			// Dynamically determine allowed prefixes based on command path
			oneof := msg.Oneofs().ByName("type")
			var specifics []string
			for i := 0; i < oneof.Fields().Len(); i++ {
				fname := string(oneof.Fields().Get(i).Name())
				specifics = append(specifics, fname)

				// Reconstruct cmdName from fname
				nameClean := strings.ReplaceAll(fname, "_", "-")
				cmdName := nameClean + "s"
				switch nameClean {
				case "battery":
					cmdName = "batteries"
				case "gps":
					cmdName = "gps"
				}
				if strings.HasSuffix(nameClean, "s") && nameClean != "gps" {
					cmdName = nameClean
				}

				// If the command path contains the cmdName, it's allowed
				if strings.Contains(path, " "+cmdName) {
					allowedPrefixes = append(allowedPrefixes, fname)
				}
			}

			// Filter fields
			for _, f := range allFields {
				isSpecific := false
				belongsTo := ""
				for _, s := range specifics {
					if f == s || strings.HasPrefix(f, s+".") {
						isSpecific = true
						belongsTo = s
						break
					}
				}

				if !isSpecific {
					if f != "type" {
						filtered = append(filtered, f)
					}
				} else {
					if len(allowedPrefixes) == 0 {
						filtered = append(filtered, f)
					} else {
						for _, ap := range allowedPrefixes {
							if belongsTo == ap {
								trimmed := strings.TrimPrefix(f, ap+".")
								if trimmed != ap && trimmed != "" {
									filtered = append(filtered, trimmed)
								}
								break
							}
						}
					}
				}
			}
			allFields = filtered
		}

		lastComma := strings.LastIndex(toComplete, ",")
		prefix := ""
		currentComp := toComplete
		typedFields := make(map[string]bool)

		if lastComma != -1 {
			prefix = toComplete[:lastComma+1]
			currentComp = toComplete[lastComma+1:]
			for _, p := range strings.Split(toComplete[:lastComma], ",") {
				typedFields[strings.TrimSpace(p)] = true
			}
		}

		var completions []string
		for _, f := range allFields {
			if typedFields[f] {
				continue // Skip fields that have already been typed
			}
			if strings.HasPrefix(f, currentComp) {
				completions = append(completions, prefix+f)
			}
		}
		return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.MarkFlagsMutuallyExclusive("json", "yaml")

	// Group: components
	componentsCmd := &cobra.Command{
		Use:   "components",
		Short: "Manage quadcopter components",
	}
	rootCmd.AddCommand(componentsCmd)

	// Group: hardware
	hardwareCmd := &cobra.Command{
		Use:   "hardware",
		Short: "Manage hardware components",
	}
	componentsCmd.AddCommand(hardwareCmd)

	// Group: gear
	gearCmd := &cobra.Command{
		Use:   "gear",
		Short: "Manage gear",
	}
	componentsCmd.AddCommand(gearCmd)

	// Group: software
	softwareCmd := &cobra.Command{
		Use:   "software",
		Short: "Manage software components",
	}
	componentsCmd.AddCommand(softwareCmd)

	// Dynamically generate subcommands for each component type
	var c pb.Component
	msgDesc := c.ProtoReflect().Descriptor()
	oneof := msgDesc.Oneofs().ByName("type")
	fields := oneof.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		typeName := string(field.Name())
		nameClean := strings.ReplaceAll(typeName, "_", "-")
		cType := typeName // Fix closure capture

		cmdName := nameClean + "s"

		// Handle irregular plurals
		switch nameClean {
		case "battery":
			cmdName = "batteries"
		case "gps":
			cmdName = "gps"
		}
		if strings.HasSuffix(nameClean, "s") && nameClean != "gps" {
			cmdName = nameClean
		}

		// Determine which group this belongs to
		var groupCmd *cobra.Command
		if strings.HasSuffix(typeName, "_firmware") || typeName == "radio_os" {
			groupCmd = softwareCmd
		} else if typeName == "radio" || typeName == "goggles" || typeName == "radio_module" {
			groupCmd = gearCmd
		} else {
			groupCmd = hardwareCmd
		}

		var cols []string
		compOpts := (&pb.Component{}).ProtoReflect().Descriptor().Options().(*descriptorpb.MessageOptions)
		if proto.HasExtension(compOpts, pb.E_DefaultColumns) {
			baseCols := proto.GetExtension(compOpts, pb.E_DefaultColumns).([]string)
			cols = make([]string, len(baseCols))
			copy(cols, baseCols)
		}

		subMsg := field.Message()
		if subMsg != nil {
			opts := subMsg.Options().(*descriptorpb.MessageOptions)
			if proto.HasExtension(opts, pb.E_DefaultColumns) {
				cols = append(cols, proto.GetExtension(opts, pb.E_DefaultColumns).([]string)...)
			}
		}

		listFn := func(ctx context.Context, filter string) (proto.Message, []*pb.Component, error) {
			mask := cols
				if columns != "" {
					mask = strings.Split(columns, ",")
				}
				mask = resolveCols(mask, cType)
			req := &pb.ListComponentsRequest{}
			req.SetComponentType(cType)
			req.SetFilter(filter)
			req.SetFieldMask(mask)
			res, err := getClient().ListComponents(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, nil, err
			}
			return res.Msg, res.Msg.GetComponents(), nil
		}

		getFn := func(ctx context.Context, id string) (proto.Message, error) {
			mask := cols
				if columns != "" {
					mask = strings.Split(columns, ",")
				}
				mask = resolveCols(mask, cType)
			req := &pb.GetComponentRequest{}
			req.SetId(id)
			req.SetFieldMask(mask)
			res, err := getClient().GetComponent(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return res.Msg, nil
		}

		typeCmd := createResourceCmd(cmdName, cType, cols, listFn, getFn)
		groupCmd.AddCommand(typeCmd)
	}

	listFn := func(ctx context.Context, filter string) (proto.Message, []*pb.Build, error) {
		req := &pb.ListBuildsRequest{}
		req.SetFilter(filter)
		req.SetFieldMask([]string{"name", "id", "crash_resistance_rating", "is_verified"})
		if columns != "" {
			req.SetFieldMask(strings.Split(columns, ","))
		}
		res, err := getClient().ListBuilds(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, nil, err
		}
		return res.Msg, res.Msg.GetBuilds(), nil
	}

	getFn := func(ctx context.Context, id string) (proto.Message, error) {
		req := &pb.GetBuildRequest{}
		req.SetId(id)
		req.SetFieldMask([]string{"name", "id", "crash_resistance_rating", "is_verified"})
		if columns != "" {
			req.SetFieldMask(strings.Split(columns, ","))
		}
		res, err := getClient().GetBuild(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return res.Msg, nil
	}

	var buildCols []string
	buildOpts := (&pb.Build{}).ProtoReflect().Descriptor().Options().(*descriptorpb.MessageOptions)
	if proto.HasExtension(buildOpts, pb.E_DefaultColumns) {
		buildCols = proto.GetExtension(buildOpts, pb.E_DefaultColumns).([]string)
	}
	if len(buildCols) == 0 {
		buildCols = []string{"name", "id", "crash_resistance_rating", "is_verified"}
	}
	buildsCmd := createResourceCmd("builds", "", buildCols, listFn, getFn)
	rootCmd.AddCommand(buildsCmd)

	evalCmd := &cobra.Command{
		Use:   "evaluate [build-id]",
		Short: "Evaluate a build's flight dynamics",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			_, items, err := listFn(context.Background(), "")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return generateCompletions(items, toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, args []string) {
			buildID := args[0]
			payload, _ := cmd.Flags().GetFloat64("payload")

			// MVP: The backend currently expects component IDs, so if this is a comma-separated list,
			// it will split it. In the future, this should pass the actual build ID.
			req := &pb.EvaluateBuildRequest{}
			req.SetComponentIds(strings.Split(buildID, ","))
			req.SetPayloadWeightG(payload)

			res, err := client.EvaluateBuild(context.Background(), connect.NewRequest(req))
			if err != nil {
				log.Fatalf("Evaluation failed: %v", err)
			}

			if jsonOut {
				fmt.Println(formatOutput(res.Msg))
				return
			}

			r := res.Msg.GetResult()
			fmt.Printf("\n--- FLIGHT DYNAMICS RESULT ---\n")
			fmt.Printf("Total Mass:  %.1fg\n", r.GetTotalMassG())
			fmt.Printf("Thrust-to-Weight: %.2f\n", r.GetThrustToWeightRatio())
			fmt.Printf("Hover Time:  %.1f mins\n", r.GetHoverFlightTimeS()/60)
			fmt.Printf("Mixed Time:  %.1f mins\n", r.GetFreestyleFlightTimeS()/60)
			fmt.Printf("Top Speed:   %.1f KPH\n", r.GetMaxSpeedKph())
		},
	}
	evalCmd.Flags().Float64("payload", 0, "Payload weight in grams (e.g. GoPro)")
	buildsCmd.AddCommand(evalCmd)

	compatCmd := &cobra.Command{
		Use:   "compatibility [ids]",
		Short: "Check compatibility between components",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// Handle comma-separated list prefix
			lastComma := strings.LastIndex(toComplete, ",")
			prefix := ""
			searchStr := toComplete
			if lastComma != -1 {
				prefix = toComplete[:lastComma+1]
				searchStr = toComplete[lastComma+1:]
			}

			compClient := quadsmithconnect.NewQuadsmithAPIClient(http.DefaultClient, "http://localhost:8080")
			res, err := compClient.ListComponents(context.Background(), connect.NewRequest(&pb.ListComponentsRequest{}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, c := range res.Msg.GetComponents() {
				if strings.HasPrefix(c.GetId(), searchStr) {
					// We return prefix + id so bash replaces the whole token, but only visually completes the last ID
					completions = append(completions, fmt.Sprintf("%s%s\t%s", prefix, c.GetId(), c.GetName()))
				}
			}
			// Use NoSpace so the user can easily type the next comma without deleting spaces
			return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, args []string) {
			idsStr := args[0]

			req := &pb.CheckCompatibilityRequest{}
			req.SetComponentIds(strings.Split(idsStr, ","))

			res, err := client.CheckCompatibility(context.Background(), connect.NewRequest(req))
			if err != nil {
				log.Fatalf("Compatibility check failed: %v", err)
			}

			if jsonOut {
				fmt.Println(formatOutput(res.Msg))
				return
			}

			fmt.Printf("\n--- COMPATIBILITY RESULTS ---\n")
			if len(res.Msg.GetResults()) == 0 {
				fmt.Println("No incompatibilities found! (100% Compatible)")
				return
			}

			for _, rule := range res.Msg.GetResults() {
				fmt.Printf("\n[Rule: %s]\n", rule.GetCheckerName())
				for _, msg := range rule.GetMessages() {
					fmt.Printf(" - ERROR: %s\n", msg.GetMessage())
					fmt.Printf("   FIX:   %s\n", msg.GetResolution())
				}
			}
		},
	}
	componentsCmd.AddCommand(compatCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
