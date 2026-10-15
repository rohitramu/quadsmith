package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	"quadsmith/shared/pkg/config"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
	"quadsmith/shared/gen/quadsmith/common"
	configpb "quadsmith/shared/gen/quadsmith/config"
)

var (
	jsonOut           bool
	yamlOut           bool
	overrideTargetURL string
)

func getEmbeddedConfig() *configpb.CliConfig {
	var cfg configpb.CliConfig
	opts := prototext.UnmarshalOptions{DiscardUnknown: false}
	if err := opts.Unmarshal(embeddedConfigRaw, &cfg); err != nil {
		panic(fmt.Sprintf("failed to parse embedded CLI configuration: %v", err))
	}
	if err := config.ValidateCliConfig(&cfg); err != nil {
		panic(fmt.Sprintf("invalid embedded CLI configuration: %v", err))
	}
	return &cfg
}

func newRootCmd() *cobra.Command {
	jsonOut = false
	yamlOut = false
	if _, ok := os.LookupEnv("QS_COMPLETION_DESCRIPTIONS"); !ok {
		_ = os.Setenv("QS_COMPLETION_DESCRIPTIONS", "false")
	}
	cliCfg := getEmbeddedConfig()
	targetURL := cliCfg.GetApiUrl()
	if envURL := os.Getenv("QS_CLI_API_URL"); envURL != "" {
		targetURL = envURL
	} else if envURL := os.Getenv("QS_API_URL"); envURL != "" {
		targetURL = envURL
	}
	if overrideTargetURL != "" {
		targetURL = overrideTargetURL
	}
	SetActiveTargetURL(targetURL)

	rootCmd := &cobra.Command{
		Use:   "qs",
		Short: "Quadsmith CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			target := cliCfg.GetApiUrl()
			if envURL := os.Getenv("QS_CLI_API_URL"); envURL != "" {
				target = envURL
			} else if envURL := os.Getenv("QS_API_URL"); envURL != "" {
				target = envURL
			}
			if overrideTargetURL != "" {
				target = overrideTargetURL
			}
			SetActiveTargetURL(target)
			return nil
		},
	}
	rootCmd.CompletionOptions.DisableDescriptions = true

	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output format as JSON")
	httpClient := newCLIHTTPClient()

	componentsCmd := &cobra.Command{
		Use:     "components",
		Aliases: []string{"component"},
		Short:   "Hardware and software component collections",
	}
	rootCmd.AddCommand(componentsCmd)

	componentsHardwareCmd := &cobra.Command{
		Use:     "hardware",
		Aliases: []string{"hw"},
		Short:   "Hardware component collections",
	}
	componentsCmd.AddCommand(componentsHardwareCmd)

	componentsSoftwareCmd := &cobra.Command{
		Use:   "software",
		Short: "Software collections",
	}
	componentsCmd.AddCommand(componentsSoftwareCmd)

	componentsGearCmd := &cobra.Command{
		Use:   "gear",
		Short: "Gear collections",
	}
	componentsCmd.AddCommand(componentsGearCmd)

	// --- Antennas ---
	antennaClient := apiconnect.NewAntennaServiceClient(httpClient, targetURL)
	antennaCmd := &cobra.Command{Use: "antennas", Aliases: []string{"antenna"}}
	antennaListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Antenna
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListAntennasRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := antennaClient.ListAntennas(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Antennas...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Antenna{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	antennaListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	antennaListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	antennaListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	antennaListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Antenna{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	antennaListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	antennaListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	antennaListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Antenna{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	antennaListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Antenna{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	antennaCmd.AddCommand(antennaListCmd)

	antennaGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := antennaClient.ListAntennas(context.Background(), connect.NewRequest(&pb.ListAntennasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Antennas {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetAntennaRequest{Id: args[0], Columns: columns}
			res, err := antennaClient.GetAntenna(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	antennaGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	antennaGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Antenna{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	antennaCmd.AddCommand(antennaGetCmd)
	antennaCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new antenna",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Antenna
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse antenna JSON: %w", err)
			}
			res, err := antennaClient.CreateAntenna(context.Background(), connect.NewRequest(&pb.CreateAntennaRequest{Antenna: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created antenna '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	antennaCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	antennaUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing antenna",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := antennaClient.ListAntennas(context.Background(), connect.NewRequest(&pb.ListAntennasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Antennas {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Antenna
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse antenna JSON: %w", err)
			}
			item.Id = args[0]
			res, err := antennaClient.UpdateAntenna(context.Background(), connect.NewRequest(&pb.UpdateAntennaRequest{Antenna: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated antenna '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	antennaUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	antennaDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing antenna",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := antennaClient.ListAntennas(context.Background(), connect.NewRequest(&pb.ListAntennasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Antennas {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := antennaClient.DeleteAntenna(context.Background(), connect.NewRequest(&pb.DeleteAntennaRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	antennaCmd.AddCommand(antennaCreateCmd)
	antennaCmd.AddCommand(antennaUpdateCmd)
	antennaCmd.AddCommand(antennaDeleteCmd)
	componentsHardwareCmd.AddCommand(antennaCmd)

	// --- Batteries ---
	batteryClient := apiconnect.NewBatteryServiceClient(httpClient, targetURL)
	batteryCmd := &cobra.Command{Use: "batteries", Aliases: []string{"battery"}}
	batteryListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Battery
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListBatteriesRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Batteries...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Battery{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	batteryListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	batteryListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	batteryListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	batteryListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Battery{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	batteryListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	batteryListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	batteryListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Battery{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	batteryListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Battery{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	batteryCmd.AddCommand(batteryListCmd)

	batteryGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(&pb.ListBatteriesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Batteries {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetBatteryRequest{Id: args[0], Columns: columns}
			res, err := batteryClient.GetBattery(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	batteryGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	batteryGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Battery{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	batteryCmd.AddCommand(batteryGetCmd)
	batteryCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new battery",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Battery
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse battery JSON: %w", err)
			}
			res, err := batteryClient.CreateBattery(context.Background(), connect.NewRequest(&pb.CreateBatteryRequest{Battery: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created battery '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	batteryCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	batteryUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing battery",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(&pb.ListBatteriesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Batteries {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Battery
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse battery JSON: %w", err)
			}
			item.Id = args[0]
			res, err := batteryClient.UpdateBattery(context.Background(), connect.NewRequest(&pb.UpdateBatteryRequest{Battery: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated battery '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	batteryUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	batteryDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing battery",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(&pb.ListBatteriesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Batteries {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := batteryClient.DeleteBattery(context.Background(), connect.NewRequest(&pb.DeleteBatteryRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	batteryCmd.AddCommand(batteryCreateCmd)
	batteryCmd.AddCommand(batteryUpdateCmd)
	batteryCmd.AddCommand(batteryDeleteCmd)
	componentsHardwareCmd.AddCommand(batteryCmd)

	// --- Builds ---
	buildClient := apiconnect.NewBuildServiceClient(httpClient, targetURL)
	buildCmd := &cobra.Command{Use: "builds", Aliases: []string{"build"}}
	buildListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Build
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListBuildsRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Builds...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Build{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	buildListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	buildListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	buildListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	buildListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Build{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	buildListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	buildListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	buildListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Build{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	buildListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Build{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	buildCmd.AddCommand(buildListCmd)

	buildGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(&pb.ListBuildsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Builds {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetBuildRequest{Id: args[0], Columns: columns}
			res, err := buildClient.GetBuild(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	buildGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	buildGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Build{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	buildCmd.AddCommand(buildGetCmd)
	buildCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new build",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Build
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse build JSON: %w", err)
			}
			res, err := buildClient.CreateBuild(context.Background(), connect.NewRequest(&pb.CreateBuildRequest{Build: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created build '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	buildCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	buildDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing build",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(&pb.ListBuildsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Builds {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := buildClient.DeleteBuild(context.Background(), connect.NewRequest(&pb.DeleteBuildRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	buildCmd.AddCommand(buildCreateCmd)
	buildCmd.AddCommand(buildDeleteCmd)
	rootCmd.AddCommand(buildCmd)

	// --- Cameras ---
	cameraClient := apiconnect.NewCameraServiceClient(httpClient, targetURL)
	cameraCmd := &cobra.Command{Use: "cameras", Aliases: []string{"camera"}}
	cameraListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Camera
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListCamerasRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := cameraClient.ListCameras(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Cameras...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Camera{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cameraListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	cameraListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	cameraListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	cameraListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Camera{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	cameraListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	cameraListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	cameraListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Camera{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	cameraListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Camera{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	cameraCmd.AddCommand(cameraListCmd)

	cameraGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := cameraClient.ListCameras(context.Background(), connect.NewRequest(&pb.ListCamerasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Cameras {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetCameraRequest{Id: args[0], Columns: columns}
			res, err := cameraClient.GetCamera(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	cameraGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	cameraGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Camera{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	cameraCmd.AddCommand(cameraGetCmd)
	cameraCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new camera",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Camera
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse camera JSON: %w", err)
			}
			res, err := cameraClient.CreateCamera(context.Background(), connect.NewRequest(&pb.CreateCameraRequest{Camera: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created camera '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	cameraCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	cameraUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing camera",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := cameraClient.ListCameras(context.Background(), connect.NewRequest(&pb.ListCamerasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Cameras {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Camera
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse camera JSON: %w", err)
			}
			item.Id = args[0]
			res, err := cameraClient.UpdateCamera(context.Background(), connect.NewRequest(&pb.UpdateCameraRequest{Camera: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated camera '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	cameraUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	cameraDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing camera",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := cameraClient.ListCameras(context.Background(), connect.NewRequest(&pb.ListCamerasRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Cameras {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := cameraClient.DeleteCamera(context.Background(), connect.NewRequest(&pb.DeleteCameraRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	cameraCmd.AddCommand(cameraCreateCmd)
	cameraCmd.AddCommand(cameraUpdateCmd)
	cameraCmd.AddCommand(cameraDeleteCmd)
	componentsHardwareCmd.AddCommand(cameraCmd)

	// --- ElectronicSpeedControllers ---
	electronicSpeedControllerClient := apiconnect.NewElectronicSpeedControllerServiceClient(httpClient, targetURL)
	electronicSpeedControllerCmd := &cobra.Command{Use: "electronic-speed-controllers", Aliases: []string{"electronicspeedcontrollers", "escs", "esc", "electronic-speed-controller"}}
	electronicSpeedControllerListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.ElectronicSpeedController
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListElectronicSpeedControllersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := electronicSpeedControllerClient.ListElectronicSpeedControllers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.ElectronicSpeedControllers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.ElectronicSpeedController{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	electronicSpeedControllerListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	electronicSpeedControllerListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	electronicSpeedControllerListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	electronicSpeedControllerListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.ElectronicSpeedController{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	electronicSpeedControllerListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	electronicSpeedControllerListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	electronicSpeedControllerListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.ElectronicSpeedController{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	electronicSpeedControllerListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.ElectronicSpeedController{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	electronicSpeedControllerCmd.AddCommand(electronicSpeedControllerListCmd)

	electronicSpeedControllerGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := electronicSpeedControllerClient.ListElectronicSpeedControllers(context.Background(), connect.NewRequest(&pb.ListElectronicSpeedControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.ElectronicSpeedControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetElectronicSpeedControllerRequest{Id: args[0], Columns: columns}
			res, err := electronicSpeedControllerClient.GetElectronicSpeedController(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	electronicSpeedControllerGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	electronicSpeedControllerGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.ElectronicSpeedController{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	electronicSpeedControllerCmd.AddCommand(electronicSpeedControllerGetCmd)
	electronicSpeedControllerCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new electronicspeedcontroller",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.ElectronicSpeedController
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse electronicspeedcontroller JSON: %w", err)
			}
			res, err := electronicSpeedControllerClient.CreateElectronicSpeedController(context.Background(), connect.NewRequest(&pb.CreateElectronicSpeedControllerRequest{ElectronicSpeedController: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created electronicspeedcontroller '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	electronicSpeedControllerCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	electronicSpeedControllerUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing electronicspeedcontroller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := electronicSpeedControllerClient.ListElectronicSpeedControllers(context.Background(), connect.NewRequest(&pb.ListElectronicSpeedControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.ElectronicSpeedControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.ElectronicSpeedController
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse electronicspeedcontroller JSON: %w", err)
			}
			item.Id = args[0]
			res, err := electronicSpeedControllerClient.UpdateElectronicSpeedController(context.Background(), connect.NewRequest(&pb.UpdateElectronicSpeedControllerRequest{ElectronicSpeedController: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated electronicspeedcontroller '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	electronicSpeedControllerUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	electronicSpeedControllerDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing electronicspeedcontroller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := electronicSpeedControllerClient.ListElectronicSpeedControllers(context.Background(), connect.NewRequest(&pb.ListElectronicSpeedControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.ElectronicSpeedControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := electronicSpeedControllerClient.DeleteElectronicSpeedController(context.Background(), connect.NewRequest(&pb.DeleteElectronicSpeedControllerRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	electronicSpeedControllerCmd.AddCommand(electronicSpeedControllerCreateCmd)
	electronicSpeedControllerCmd.AddCommand(electronicSpeedControllerUpdateCmd)
	electronicSpeedControllerCmd.AddCommand(electronicSpeedControllerDeleteCmd)
	componentsHardwareCmd.AddCommand(electronicSpeedControllerCmd)

	// --- FlightControllers ---
	flightControllerClient := apiconnect.NewFlightControllerServiceClient(httpClient, targetURL)
	flightControllerCmd := &cobra.Command{Use: "flight-controllers", Aliases: []string{"flightcontrollers", "fc", "fcs", "flight-controller"}}
	flightControllerListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.FlightController
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListFlightControllersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := flightControllerClient.ListFlightControllers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.FlightControllers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.FlightController{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flightControllerListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	flightControllerListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	flightControllerListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	flightControllerListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FlightController{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	flightControllerListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	flightControllerListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	flightControllerListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FlightController{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	flightControllerListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FlightController{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	flightControllerCmd.AddCommand(flightControllerListCmd)

	flightControllerGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := flightControllerClient.ListFlightControllers(context.Background(), connect.NewRequest(&pb.ListFlightControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FlightControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetFlightControllerRequest{Id: args[0], Columns: columns}
			res, err := flightControllerClient.GetFlightController(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	flightControllerGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	flightControllerGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FlightController{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	flightControllerCmd.AddCommand(flightControllerGetCmd)
	flightControllerCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new flightcontroller",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FlightController
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse flightcontroller JSON: %w", err)
			}
			res, err := flightControllerClient.CreateFlightController(context.Background(), connect.NewRequest(&pb.CreateFlightControllerRequest{FlightController: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created flightcontroller '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	flightControllerCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	flightControllerUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing flightcontroller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := flightControllerClient.ListFlightControllers(context.Background(), connect.NewRequest(&pb.ListFlightControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FlightControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FlightController
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse flightcontroller JSON: %w", err)
			}
			item.Id = args[0]
			res, err := flightControllerClient.UpdateFlightController(context.Background(), connect.NewRequest(&pb.UpdateFlightControllerRequest{FlightController: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated flightcontroller '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	flightControllerUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	flightControllerDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing flightcontroller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := flightControllerClient.ListFlightControllers(context.Background(), connect.NewRequest(&pb.ListFlightControllersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FlightControllers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := flightControllerClient.DeleteFlightController(context.Background(), connect.NewRequest(&pb.DeleteFlightControllerRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	flightControllerCmd.AddCommand(flightControllerCreateCmd)
	flightControllerCmd.AddCommand(flightControllerUpdateCmd)
	flightControllerCmd.AddCommand(flightControllerDeleteCmd)
	componentsHardwareCmd.AddCommand(flightControllerCmd)

	// --- Frames ---
	frameClient := apiconnect.NewFrameServiceClient(httpClient, targetURL)
	frameCmd := &cobra.Command{Use: "frames", Aliases: []string{"frame"}}
	frameListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Frame
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListFramesRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := frameClient.ListFrames(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Frames...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Frame{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	frameListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	frameListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	frameListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	frameListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Frame{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	frameListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	frameListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	frameListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Frame{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	frameListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Frame{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	frameCmd.AddCommand(frameListCmd)

	frameGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := frameClient.ListFrames(context.Background(), connect.NewRequest(&pb.ListFramesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Frames {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetFrameRequest{Id: args[0], Columns: columns}
			res, err := frameClient.GetFrame(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	frameGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	frameGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Frame{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	frameCmd.AddCommand(frameGetCmd)
	frameCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new frame",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Frame
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse frame JSON: %w", err)
			}
			res, err := frameClient.CreateFrame(context.Background(), connect.NewRequest(&pb.CreateFrameRequest{Frame: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created frame '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	frameCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	frameUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing frame",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := frameClient.ListFrames(context.Background(), connect.NewRequest(&pb.ListFramesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Frames {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Frame
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse frame JSON: %w", err)
			}
			item.Id = args[0]
			res, err := frameClient.UpdateFrame(context.Background(), connect.NewRequest(&pb.UpdateFrameRequest{Frame: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated frame '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	frameUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	frameDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing frame",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := frameClient.ListFrames(context.Background(), connect.NewRequest(&pb.ListFramesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Frames {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := frameClient.DeleteFrame(context.Background(), connect.NewRequest(&pb.DeleteFrameRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	frameCmd.AddCommand(frameCreateCmd)
	frameCmd.AddCommand(frameUpdateCmd)
	frameCmd.AddCommand(frameDeleteCmd)
	componentsHardwareCmd.AddCommand(frameCmd)

	// --- GpsReceivers ---
	gpsReceiverClient := apiconnect.NewGpsReceiverServiceClient(httpClient, targetURL)
	gpsReceiverCmd := &cobra.Command{Use: "gps-receivers", Aliases: []string{"gpsreceivers", "gps", "gps-receiver"}}
	gpsReceiverListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.GpsReceiver
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListGpsReceiversRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := gpsReceiverClient.ListGpsReceivers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.GpsReceivers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.GpsReceiver{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	gpsReceiverListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	gpsReceiverListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	gpsReceiverListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	gpsReceiverListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.GpsReceiver{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gpsReceiverListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	gpsReceiverListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	gpsReceiverListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.GpsReceiver{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gpsReceiverListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.GpsReceiver{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gpsReceiverCmd.AddCommand(gpsReceiverListCmd)

	gpsReceiverGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gpsReceiverClient.ListGpsReceivers(context.Background(), connect.NewRequest(&pb.ListGpsReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.GpsReceivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetGpsReceiverRequest{Id: args[0], Columns: columns}
			res, err := gpsReceiverClient.GetGpsReceiver(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	gpsReceiverGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	gpsReceiverGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.GpsReceiver{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gpsReceiverCmd.AddCommand(gpsReceiverGetCmd)
	gpsReceiverCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new gpsreceiver",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.GpsReceiver
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse gpsreceiver JSON: %w", err)
			}
			res, err := gpsReceiverClient.CreateGpsReceiver(context.Background(), connect.NewRequest(&pb.CreateGpsReceiverRequest{GpsReceiver: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created gpsreceiver '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	gpsReceiverCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	gpsReceiverUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing gpsreceiver",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gpsReceiverClient.ListGpsReceivers(context.Background(), connect.NewRequest(&pb.ListGpsReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.GpsReceivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.GpsReceiver
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse gpsreceiver JSON: %w", err)
			}
			item.Id = args[0]
			res, err := gpsReceiverClient.UpdateGpsReceiver(context.Background(), connect.NewRequest(&pb.UpdateGpsReceiverRequest{GpsReceiver: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated gpsreceiver '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	gpsReceiverUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	gpsReceiverDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing gpsreceiver",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gpsReceiverClient.ListGpsReceivers(context.Background(), connect.NewRequest(&pb.ListGpsReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.GpsReceivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := gpsReceiverClient.DeleteGpsReceiver(context.Background(), connect.NewRequest(&pb.DeleteGpsReceiverRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	gpsReceiverCmd.AddCommand(gpsReceiverCreateCmd)
	gpsReceiverCmd.AddCommand(gpsReceiverUpdateCmd)
	gpsReceiverCmd.AddCommand(gpsReceiverDeleteCmd)
	componentsHardwareCmd.AddCommand(gpsReceiverCmd)

	// --- Motors ---
	motorClient := apiconnect.NewMotorServiceClient(httpClient, targetURL)
	motorCmd := &cobra.Command{Use: "motors", Aliases: []string{"motor"}}
	motorListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Motor
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListMotorsRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := motorClient.ListMotors(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Motors...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Motor{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	motorListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	motorListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	motorListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	motorListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Motor{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	motorListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	motorListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	motorListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Motor{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	motorListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Motor{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	motorCmd.AddCommand(motorListCmd)

	motorGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := motorClient.ListMotors(context.Background(), connect.NewRequest(&pb.ListMotorsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Motors {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetMotorRequest{Id: args[0], Columns: columns}
			res, err := motorClient.GetMotor(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	motorGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	motorGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Motor{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	motorCmd.AddCommand(motorGetCmd)
	motorCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new motor",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Motor
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse motor JSON: %w", err)
			}
			res, err := motorClient.CreateMotor(context.Background(), connect.NewRequest(&pb.CreateMotorRequest{Motor: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created motor '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	motorCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	motorUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing motor",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := motorClient.ListMotors(context.Background(), connect.NewRequest(&pb.ListMotorsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Motors {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Motor
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse motor JSON: %w", err)
			}
			item.Id = args[0]
			res, err := motorClient.UpdateMotor(context.Background(), connect.NewRequest(&pb.UpdateMotorRequest{Motor: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated motor '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	motorUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	motorDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing motor",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := motorClient.ListMotors(context.Background(), connect.NewRequest(&pb.ListMotorsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Motors {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := motorClient.DeleteMotor(context.Background(), connect.NewRequest(&pb.DeleteMotorRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	motorCmd.AddCommand(motorCreateCmd)
	motorCmd.AddCommand(motorUpdateCmd)
	motorCmd.AddCommand(motorDeleteCmd)
	componentsHardwareCmd.AddCommand(motorCmd)

	// --- Propellers ---
	propellerClient := apiconnect.NewPropellerServiceClient(httpClient, targetURL)
	propellerCmd := &cobra.Command{Use: "propellers", Aliases: []string{"propeller", "props", "prop"}}
	propellerListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Propeller
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListPropellersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := propellerClient.ListPropellers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Propellers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Propeller{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	propellerListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	propellerListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	propellerListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	propellerListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Propeller{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	propellerListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	propellerListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	propellerListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Propeller{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	propellerListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Propeller{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	propellerCmd.AddCommand(propellerListCmd)

	propellerGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := propellerClient.ListPropellers(context.Background(), connect.NewRequest(&pb.ListPropellersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Propellers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetPropellerRequest{Id: args[0], Columns: columns}
			res, err := propellerClient.GetPropeller(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	propellerGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	propellerGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Propeller{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	propellerCmd.AddCommand(propellerGetCmd)
	propellerCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new propeller",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Propeller
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse propeller JSON: %w", err)
			}
			res, err := propellerClient.CreatePropeller(context.Background(), connect.NewRequest(&pb.CreatePropellerRequest{Propeller: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created propeller '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	propellerCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	propellerUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing propeller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := propellerClient.ListPropellers(context.Background(), connect.NewRequest(&pb.ListPropellersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Propellers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Propeller
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse propeller JSON: %w", err)
			}
			item.Id = args[0]
			res, err := propellerClient.UpdatePropeller(context.Background(), connect.NewRequest(&pb.UpdatePropellerRequest{Propeller: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated propeller '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	propellerUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	propellerDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing propeller",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := propellerClient.ListPropellers(context.Background(), connect.NewRequest(&pb.ListPropellersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Propellers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := propellerClient.DeletePropeller(context.Background(), connect.NewRequest(&pb.DeletePropellerRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	propellerCmd.AddCommand(propellerCreateCmd)
	propellerCmd.AddCommand(propellerUpdateCmd)
	propellerCmd.AddCommand(propellerDeleteCmd)
	componentsHardwareCmd.AddCommand(propellerCmd)

	// --- Receivers ---
	receiverClient := apiconnect.NewReceiverServiceClient(httpClient, targetURL)
	receiverCmd := &cobra.Command{Use: "receivers", Aliases: []string{"rx", "rxs", "receiver"}}
	receiverListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Receiver
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListReceiversRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := receiverClient.ListReceivers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Receivers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Receiver{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	receiverListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	receiverListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	receiverListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	receiverListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Receiver{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	receiverListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	receiverListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	receiverListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Receiver{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	receiverListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Receiver{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	receiverCmd.AddCommand(receiverListCmd)

	receiverGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := receiverClient.ListReceivers(context.Background(), connect.NewRequest(&pb.ListReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Receivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetReceiverRequest{Id: args[0], Columns: columns}
			res, err := receiverClient.GetReceiver(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	receiverGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	receiverGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Receiver{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	receiverCmd.AddCommand(receiverGetCmd)
	receiverCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new receiver",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Receiver
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse receiver JSON: %w", err)
			}
			res, err := receiverClient.CreateReceiver(context.Background(), connect.NewRequest(&pb.CreateReceiverRequest{Receiver: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created receiver '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	receiverCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	receiverUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing receiver",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := receiverClient.ListReceivers(context.Background(), connect.NewRequest(&pb.ListReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Receivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Receiver
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse receiver JSON: %w", err)
			}
			item.Id = args[0]
			res, err := receiverClient.UpdateReceiver(context.Background(), connect.NewRequest(&pb.UpdateReceiverRequest{Receiver: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated receiver '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	receiverUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	receiverDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing receiver",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := receiverClient.ListReceivers(context.Background(), connect.NewRequest(&pb.ListReceiversRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Receivers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := receiverClient.DeleteReceiver(context.Background(), connect.NewRequest(&pb.DeleteReceiverRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	receiverCmd.AddCommand(receiverCreateCmd)
	receiverCmd.AddCommand(receiverUpdateCmd)
	receiverCmd.AddCommand(receiverDeleteCmd)
	componentsHardwareCmd.AddCommand(receiverCmd)

	// --- VideoTransmitters ---
	videoTransmitterClient := apiconnect.NewVideoTransmitterServiceClient(httpClient, targetURL)
	videoTransmitterCmd := &cobra.Command{Use: "video-transmitters", Aliases: []string{"videotransmitters", "vtx", "vtxs", "video-transmitter"}}
	videoTransmitterListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.VideoTransmitter
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListVideoTransmittersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := videoTransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.VideoTransmitters...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.VideoTransmitter{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	videoTransmitterListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	videoTransmitterListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	videoTransmitterListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	videoTransmitterListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VideoTransmitter{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	videoTransmitterListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	videoTransmitterListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	videoTransmitterListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VideoTransmitter{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	videoTransmitterListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VideoTransmitter{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	videoTransmitterCmd.AddCommand(videoTransmitterListCmd)

	videoTransmitterGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := videoTransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(&pb.ListVideoTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VideoTransmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetVideoTransmitterRequest{Id: args[0], Columns: columns}
			res, err := videoTransmitterClient.GetVideoTransmitter(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	videoTransmitterGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	videoTransmitterGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VideoTransmitter{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	videoTransmitterCmd.AddCommand(videoTransmitterGetCmd)
	videoTransmitterCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new videotransmitter",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.VideoTransmitter
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse videotransmitter JSON: %w", err)
			}
			res, err := videoTransmitterClient.CreateVideoTransmitter(context.Background(), connect.NewRequest(&pb.CreateVideoTransmitterRequest{VideoTransmitter: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created videotransmitter '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	videoTransmitterCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	videoTransmitterUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing videotransmitter",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := videoTransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(&pb.ListVideoTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VideoTransmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.VideoTransmitter
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse videotransmitter JSON: %w", err)
			}
			item.Id = args[0]
			res, err := videoTransmitterClient.UpdateVideoTransmitter(context.Background(), connect.NewRequest(&pb.UpdateVideoTransmitterRequest{VideoTransmitter: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated videotransmitter '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	videoTransmitterUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	videoTransmitterDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing videotransmitter",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := videoTransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(&pb.ListVideoTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VideoTransmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := videoTransmitterClient.DeleteVideoTransmitter(context.Background(), connect.NewRequest(&pb.DeleteVideoTransmitterRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	videoTransmitterCmd.AddCommand(videoTransmitterCreateCmd)
	videoTransmitterCmd.AddCommand(videoTransmitterUpdateCmd)
	videoTransmitterCmd.AddCommand(videoTransmitterDeleteCmd)
	componentsHardwareCmd.AddCommand(videoTransmitterCmd)

	// --- FcFirmwares ---
	fcFirmwareClient := apiconnect.NewFcFirmwareServiceClient(httpClient, targetURL)
	fcFirmwareCmd := &cobra.Command{Use: "fc-firmware", Aliases: []string{"fcfirmwares", "fc-firmwares"}}
	fcFirmwareListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.FcFirmware
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListFcFirmwaresRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := fcFirmwareClient.ListFcFirmwares(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.FcFirmwares...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.FcFirmware{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	fcFirmwareListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	fcFirmwareListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	fcFirmwareListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	fcFirmwareListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FcFirmware{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fcFirmwareListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	fcFirmwareListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	fcFirmwareListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FcFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fcFirmwareListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FcFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fcFirmwareCmd.AddCommand(fcFirmwareListCmd)

	fcFirmwareGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fcFirmwareClient.ListFcFirmwares(context.Background(), connect.NewRequest(&pb.ListFcFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FcFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetFcFirmwareRequest{Id: args[0], Columns: columns}
			res, err := fcFirmwareClient.GetFcFirmware(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	fcFirmwareGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	fcFirmwareGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FcFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fcFirmwareCmd.AddCommand(fcFirmwareGetCmd)
	fcFirmwareCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new fcfirmware",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FcFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse fcfirmware JSON: %w", err)
			}
			res, err := fcFirmwareClient.CreateFcFirmware(context.Background(), connect.NewRequest(&pb.CreateFcFirmwareRequest{FcFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created fcfirmware '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	fcFirmwareCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	fcFirmwareUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing fcfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fcFirmwareClient.ListFcFirmwares(context.Background(), connect.NewRequest(&pb.ListFcFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FcFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FcFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse fcfirmware JSON: %w", err)
			}
			item.Id = args[0]
			res, err := fcFirmwareClient.UpdateFcFirmware(context.Background(), connect.NewRequest(&pb.UpdateFcFirmwareRequest{FcFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated fcfirmware '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	fcFirmwareUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	fcFirmwareDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing fcfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fcFirmwareClient.ListFcFirmwares(context.Background(), connect.NewRequest(&pb.ListFcFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FcFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := fcFirmwareClient.DeleteFcFirmware(context.Background(), connect.NewRequest(&pb.DeleteFcFirmwareRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	fcFirmwareCmd.AddCommand(fcFirmwareCreateCmd)
	fcFirmwareCmd.AddCommand(fcFirmwareUpdateCmd)
	fcFirmwareCmd.AddCommand(fcFirmwareDeleteCmd)
	componentsSoftwareCmd.AddCommand(fcFirmwareCmd)

	// --- EscFirmwares ---
	escFirmwareClient := apiconnect.NewEscFirmwareServiceClient(httpClient, targetURL)
	escFirmwareCmd := &cobra.Command{Use: "esc-firmware", Aliases: []string{"escfirmwares", "esc-firmwares"}}
	escFirmwareListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.EscFirmware
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListEscFirmwaresRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := escFirmwareClient.ListEscFirmwares(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.EscFirmwares...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.EscFirmware{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	escFirmwareListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	escFirmwareListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	escFirmwareListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	escFirmwareListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.EscFirmware{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	escFirmwareListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	escFirmwareListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	escFirmwareListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.EscFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	escFirmwareListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.EscFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	escFirmwareCmd.AddCommand(escFirmwareListCmd)

	escFirmwareGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := escFirmwareClient.ListEscFirmwares(context.Background(), connect.NewRequest(&pb.ListEscFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.EscFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetEscFirmwareRequest{Id: args[0], Columns: columns}
			res, err := escFirmwareClient.GetEscFirmware(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	escFirmwareGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	escFirmwareGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.EscFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	escFirmwareCmd.AddCommand(escFirmwareGetCmd)
	escFirmwareCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new escfirmware",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.EscFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse escfirmware JSON: %w", err)
			}
			res, err := escFirmwareClient.CreateEscFirmware(context.Background(), connect.NewRequest(&pb.CreateEscFirmwareRequest{EscFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created escfirmware '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	escFirmwareCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	escFirmwareUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing escfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := escFirmwareClient.ListEscFirmwares(context.Background(), connect.NewRequest(&pb.ListEscFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.EscFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.EscFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse escfirmware JSON: %w", err)
			}
			item.Id = args[0]
			res, err := escFirmwareClient.UpdateEscFirmware(context.Background(), connect.NewRequest(&pb.UpdateEscFirmwareRequest{EscFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated escfirmware '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	escFirmwareUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	escFirmwareDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing escfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := escFirmwareClient.ListEscFirmwares(context.Background(), connect.NewRequest(&pb.ListEscFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.EscFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := escFirmwareClient.DeleteEscFirmware(context.Background(), connect.NewRequest(&pb.DeleteEscFirmwareRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	escFirmwareCmd.AddCommand(escFirmwareCreateCmd)
	escFirmwareCmd.AddCommand(escFirmwareUpdateCmd)
	escFirmwareCmd.AddCommand(escFirmwareDeleteCmd)
	componentsSoftwareCmd.AddCommand(escFirmwareCmd)

	// --- VtxFirmwares ---
	vtxFirmwareClient := apiconnect.NewVtxFirmwareServiceClient(httpClient, targetURL)
	vtxFirmwareCmd := &cobra.Command{Use: "vtx-firmware", Aliases: []string{"vtxfirmwares", "vtx-firmwares"}}
	vtxFirmwareListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.VtxFirmware
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListVtxFirmwaresRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := vtxFirmwareClient.ListVtxFirmwares(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.VtxFirmwares...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.VtxFirmware{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	vtxFirmwareListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	vtxFirmwareListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	vtxFirmwareListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	vtxFirmwareListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VtxFirmware{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	vtxFirmwareListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	vtxFirmwareListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	vtxFirmwareListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VtxFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	vtxFirmwareListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VtxFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	vtxFirmwareCmd.AddCommand(vtxFirmwareListCmd)

	vtxFirmwareGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := vtxFirmwareClient.ListVtxFirmwares(context.Background(), connect.NewRequest(&pb.ListVtxFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VtxFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetVtxFirmwareRequest{Id: args[0], Columns: columns}
			res, err := vtxFirmwareClient.GetVtxFirmware(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	vtxFirmwareGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	vtxFirmwareGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.VtxFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	vtxFirmwareCmd.AddCommand(vtxFirmwareGetCmd)
	vtxFirmwareCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new vtxfirmware",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.VtxFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse vtxfirmware JSON: %w", err)
			}
			res, err := vtxFirmwareClient.CreateVtxFirmware(context.Background(), connect.NewRequest(&pb.CreateVtxFirmwareRequest{VtxFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created vtxfirmware '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	vtxFirmwareCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	vtxFirmwareUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing vtxfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := vtxFirmwareClient.ListVtxFirmwares(context.Background(), connect.NewRequest(&pb.ListVtxFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VtxFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.VtxFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse vtxfirmware JSON: %w", err)
			}
			item.Id = args[0]
			res, err := vtxFirmwareClient.UpdateVtxFirmware(context.Background(), connect.NewRequest(&pb.UpdateVtxFirmwareRequest{VtxFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated vtxfirmware '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	vtxFirmwareUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	vtxFirmwareDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing vtxfirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := vtxFirmwareClient.ListVtxFirmwares(context.Background(), connect.NewRequest(&pb.ListVtxFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.VtxFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := vtxFirmwareClient.DeleteVtxFirmware(context.Background(), connect.NewRequest(&pb.DeleteVtxFirmwareRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	vtxFirmwareCmd.AddCommand(vtxFirmwareCreateCmd)
	vtxFirmwareCmd.AddCommand(vtxFirmwareUpdateCmd)
	vtxFirmwareCmd.AddCommand(vtxFirmwareDeleteCmd)
	componentsSoftwareCmd.AddCommand(vtxFirmwareCmd)

	// --- RadioFirmwares ---
	radioFirmwareClient := apiconnect.NewRadioFirmwareServiceClient(httpClient, targetURL)
	radioFirmwareCmd := &cobra.Command{Use: "radio-firmware", Aliases: []string{"radiofirmwares", "radio-firmwares"}}
	radioFirmwareListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.RadioFirmware
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListRadioFirmwaresRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := radioFirmwareClient.ListRadioFirmwares(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.RadioFirmwares...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.RadioFirmware{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	radioFirmwareListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	radioFirmwareListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	radioFirmwareListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	radioFirmwareListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.RadioFirmware{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	radioFirmwareListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	radioFirmwareListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	radioFirmwareListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.RadioFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	radioFirmwareListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.RadioFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	radioFirmwareCmd.AddCommand(radioFirmwareListCmd)

	radioFirmwareGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := radioFirmwareClient.ListRadioFirmwares(context.Background(), connect.NewRequest(&pb.ListRadioFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.RadioFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetRadioFirmwareRequest{Id: args[0], Columns: columns}
			res, err := radioFirmwareClient.GetRadioFirmware(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	radioFirmwareGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	radioFirmwareGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.RadioFirmware{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	radioFirmwareCmd.AddCommand(radioFirmwareGetCmd)
	radioFirmwareCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new radiofirmware",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.RadioFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse radiofirmware JSON: %w", err)
			}
			res, err := radioFirmwareClient.CreateRadioFirmware(context.Background(), connect.NewRequest(&pb.CreateRadioFirmwareRequest{RadioFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created radiofirmware '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	radioFirmwareCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	radioFirmwareUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing radiofirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := radioFirmwareClient.ListRadioFirmwares(context.Background(), connect.NewRequest(&pb.ListRadioFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.RadioFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.RadioFirmware
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse radiofirmware JSON: %w", err)
			}
			item.Id = args[0]
			res, err := radioFirmwareClient.UpdateRadioFirmware(context.Background(), connect.NewRequest(&pb.UpdateRadioFirmwareRequest{RadioFirmware: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated radiofirmware '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	radioFirmwareUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	radioFirmwareDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing radiofirmware",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := radioFirmwareClient.ListRadioFirmwares(context.Background(), connect.NewRequest(&pb.ListRadioFirmwaresRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.RadioFirmwares {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := radioFirmwareClient.DeleteRadioFirmware(context.Background(), connect.NewRequest(&pb.DeleteRadioFirmwareRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	radioFirmwareCmd.AddCommand(radioFirmwareCreateCmd)
	radioFirmwareCmd.AddCommand(radioFirmwareUpdateCmd)
	radioFirmwareCmd.AddCommand(radioFirmwareDeleteCmd)
	componentsSoftwareCmd.AddCommand(radioFirmwareCmd)

	// --- SoftwareTools ---
	softwareToolClient := apiconnect.NewSoftwareToolServiceClient(httpClient, targetURL)
	softwareToolCmd := &cobra.Command{Use: "software-tools", Aliases: []string{"softwaretools", "software-tool"}}
	softwareToolListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.SoftwareTool
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListSoftwareToolsRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := softwareToolClient.ListSoftwareTools(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.SoftwareTools...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.SoftwareTool{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	softwareToolListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	softwareToolListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	softwareToolListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	softwareToolListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.SoftwareTool{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	softwareToolListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	softwareToolListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	softwareToolListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.SoftwareTool{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	softwareToolListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.SoftwareTool{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	softwareToolCmd.AddCommand(softwareToolListCmd)

	softwareToolGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := softwareToolClient.ListSoftwareTools(context.Background(), connect.NewRequest(&pb.ListSoftwareToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.SoftwareTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetSoftwareToolRequest{Id: args[0], Columns: columns}
			res, err := softwareToolClient.GetSoftwareTool(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	softwareToolGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	softwareToolGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.SoftwareTool{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	softwareToolCmd.AddCommand(softwareToolGetCmd)
	softwareToolCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new softwaretool",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.SoftwareTool
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse softwaretool JSON: %w", err)
			}
			res, err := softwareToolClient.CreateSoftwareTool(context.Background(), connect.NewRequest(&pb.CreateSoftwareToolRequest{SoftwareTool: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created softwaretool '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	softwareToolCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	softwareToolUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing softwaretool",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := softwareToolClient.ListSoftwareTools(context.Background(), connect.NewRequest(&pb.ListSoftwareToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.SoftwareTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.SoftwareTool
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse softwaretool JSON: %w", err)
			}
			item.Id = args[0]
			res, err := softwareToolClient.UpdateSoftwareTool(context.Background(), connect.NewRequest(&pb.UpdateSoftwareToolRequest{SoftwareTool: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated softwaretool '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	softwareToolUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	softwareToolDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing softwaretool",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := softwareToolClient.ListSoftwareTools(context.Background(), connect.NewRequest(&pb.ListSoftwareToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.SoftwareTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := softwareToolClient.DeleteSoftwareTool(context.Background(), connect.NewRequest(&pb.DeleteSoftwareToolRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	softwareToolCmd.AddCommand(softwareToolCreateCmd)
	softwareToolCmd.AddCommand(softwareToolUpdateCmd)
	softwareToolCmd.AddCommand(softwareToolDeleteCmd)
	componentsSoftwareCmd.AddCommand(softwareToolCmd)

	// --- Transmitters ---
	transmitterClient := apiconnect.NewTransmitterServiceClient(httpClient, targetURL)
	transmitterCmd := &cobra.Command{Use: "transmitters", Aliases: []string{"transmitter", "radios", "radio"}}
	transmitterListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Transmitter
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListTransmittersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := transmitterClient.ListTransmitters(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Transmitters...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Transmitter{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	transmitterListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	transmitterListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	transmitterListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	transmitterListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Transmitter{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	transmitterListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	transmitterListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	transmitterListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Transmitter{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	transmitterListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Transmitter{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	transmitterCmd.AddCommand(transmitterListCmd)

	transmitterGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := transmitterClient.ListTransmitters(context.Background(), connect.NewRequest(&pb.ListTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Transmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetTransmitterRequest{Id: args[0], Columns: columns}
			res, err := transmitterClient.GetTransmitter(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	transmitterGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	transmitterGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Transmitter{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	transmitterCmd.AddCommand(transmitterGetCmd)
	transmitterCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new transmitter",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Transmitter
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse transmitter JSON: %w", err)
			}
			res, err := transmitterClient.CreateTransmitter(context.Background(), connect.NewRequest(&pb.CreateTransmitterRequest{Transmitter: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created transmitter '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	transmitterCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	transmitterUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing transmitter",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := transmitterClient.ListTransmitters(context.Background(), connect.NewRequest(&pb.ListTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Transmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Transmitter
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse transmitter JSON: %w", err)
			}
			item.Id = args[0]
			res, err := transmitterClient.UpdateTransmitter(context.Background(), connect.NewRequest(&pb.UpdateTransmitterRequest{Transmitter: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated transmitter '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	transmitterUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	transmitterDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing transmitter",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := transmitterClient.ListTransmitters(context.Background(), connect.NewRequest(&pb.ListTransmittersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Transmitters {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := transmitterClient.DeleteTransmitter(context.Background(), connect.NewRequest(&pb.DeleteTransmitterRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	transmitterCmd.AddCommand(transmitterCreateCmd)
	transmitterCmd.AddCommand(transmitterUpdateCmd)
	transmitterCmd.AddCommand(transmitterDeleteCmd)
	componentsGearCmd.AddCommand(transmitterCmd)

	// --- Goggles ---
	gogglesClient := apiconnect.NewGogglesServiceClient(httpClient, targetURL)
	gogglesCmd := &cobra.Command{Use: "goggles", Aliases: []string{"headsets"}}
	gogglesListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Goggles
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListGogglesRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := gogglesClient.ListGoggles(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Goggles...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Goggles{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	gogglesListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	gogglesListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	gogglesListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	gogglesListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Goggles{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gogglesListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	gogglesListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	gogglesListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Goggles{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gogglesListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Goggles{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gogglesCmd.AddCommand(gogglesListCmd)

	gogglesGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gogglesClient.ListGoggles(context.Background(), connect.NewRequest(&pb.ListGogglesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Goggles {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetGogglesRequest{Id: args[0], Columns: columns}
			res, err := gogglesClient.GetGoggles(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	gogglesGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	gogglesGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Goggles{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	gogglesCmd.AddCommand(gogglesGetCmd)
	gogglesCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new goggles",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Goggles
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse goggles JSON: %w", err)
			}
			res, err := gogglesClient.CreateGoggles(context.Background(), connect.NewRequest(&pb.CreateGogglesRequest{Goggles: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created goggles '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	gogglesCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	gogglesUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing goggles",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gogglesClient.ListGoggles(context.Background(), connect.NewRequest(&pb.ListGogglesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Goggles {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Goggles
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse goggles JSON: %w", err)
			}
			item.Id = args[0]
			res, err := gogglesClient.UpdateGoggles(context.Background(), connect.NewRequest(&pb.UpdateGogglesRequest{Goggles: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated goggles '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	gogglesUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	gogglesDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing goggles",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := gogglesClient.ListGoggles(context.Background(), connect.NewRequest(&pb.ListGogglesRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Goggles {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := gogglesClient.DeleteGoggles(context.Background(), connect.NewRequest(&pb.DeleteGogglesRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	gogglesCmd.AddCommand(gogglesCreateCmd)
	gogglesCmd.AddCommand(gogglesUpdateCmd)
	gogglesCmd.AddCommand(gogglesDeleteCmd)
	componentsGearCmd.AddCommand(gogglesCmd)

	// --- Chargers ---
	chargerClient := apiconnect.NewChargerServiceClient(httpClient, targetURL)
	chargerCmd := &cobra.Command{Use: "chargers", Aliases: []string{"charger"}}
	chargerListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.Charger
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListChargersRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := chargerClient.ListChargers(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Chargers...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.Charger{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	chargerListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	chargerListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	chargerListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	chargerListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Charger{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	chargerListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	chargerListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	chargerListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Charger{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	chargerListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Charger{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	chargerCmd.AddCommand(chargerListCmd)

	chargerGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := chargerClient.ListChargers(context.Background(), connect.NewRequest(&pb.ListChargersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Chargers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetChargerRequest{Id: args[0], Columns: columns}
			res, err := chargerClient.GetCharger(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	chargerGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	chargerGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Charger{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	chargerCmd.AddCommand(chargerGetCmd)
	chargerCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new charger",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Charger
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse charger JSON: %w", err)
			}
			res, err := chargerClient.CreateCharger(context.Background(), connect.NewRequest(&pb.CreateChargerRequest{Charger: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created charger '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	chargerCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	chargerUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing charger",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := chargerClient.ListChargers(context.Background(), connect.NewRequest(&pb.ListChargersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Chargers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.Charger
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse charger JSON: %w", err)
			}
			item.Id = args[0]
			res, err := chargerClient.UpdateCharger(context.Background(), connect.NewRequest(&pb.UpdateChargerRequest{Charger: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated charger '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	chargerUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	chargerDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing charger",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := chargerClient.ListChargers(context.Background(), connect.NewRequest(&pb.ListChargersRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Chargers {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := chargerClient.DeleteCharger(context.Background(), connect.NewRequest(&pb.DeleteChargerRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	chargerCmd.AddCommand(chargerCreateCmd)
	chargerCmd.AddCommand(chargerUpdateCmd)
	chargerCmd.AddCommand(chargerDeleteCmd)
	componentsGearCmd.AddCommand(chargerCmd)

	// --- FieldTools ---
	fieldToolClient := apiconnect.NewFieldToolServiceClient(httpClient, targetURL)
	fieldToolCmd := &cobra.Command{Use: "field-tools", Aliases: []string{"fieldtools", "field-tool"}}
	fieldToolListCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}
			var all []*pb.FieldTool
			var currentToken string
			for {
				pageSize := int32(100)
				if limit > 0 {
					remaining := limit - int32(len(all))
					if remaining <= 0 {
						break
					}
					if remaining < pageSize {
						pageSize = remaining
					}
				}
				req := &pb.ListFieldToolsRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := fieldToolClient.ListFieldTools(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.FieldTools...)
				if limit > 0 && int32(len(all)) >= limit {
					all = all[:limit]
					break
				}
				if res.Msg.NextPageToken == "" {
					break
				}
				currentToken = res.Msg.NextPageToken
			}
			if len(columns) == 0 {
				columns = GetDefaultColumns(&pb.FieldTool{})
			}
			yamlOut, _ = cmd.Flags().GetBool("yaml")
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	fieldToolListCmd.Flags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	fieldToolListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	fieldToolListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	fieldToolListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FieldTool{})
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range cols {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fieldToolListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	fieldToolListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	fieldToolListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FieldTool{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fieldToolListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FieldTool{})
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, cleanPrefix) {
				if isDesc {
					filtered = append(filtered, "^"+c)
				} else {
					filtered = append(filtered, c)
					filtered = append(filtered, "^"+c)
				}
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fieldToolCmd.AddCommand(fieldToolListCmd)

	fieldToolGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fieldToolClient.ListFieldTools(context.Background(), connect.NewRequest(&pb.ListFieldToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FieldTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetFieldToolRequest{Id: args[0], Columns: columns}
			res, err := fieldToolClient.GetFieldTool(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printGetOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	fieldToolGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	fieldToolGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.FieldTool{})
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range cols {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	fieldToolCmd.AddCommand(fieldToolGetCmd)
	fieldToolCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new fieldtool",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FieldTool
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse fieldtool JSON: %w", err)
			}
			res, err := fieldToolClient.CreateFieldTool(context.Background(), connect.NewRequest(&pb.CreateFieldToolRequest{FieldTool: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully created fieldtool '%s' (uuid: %s)\n", res.Msg.Id, res.Msg.Uuid)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	fieldToolCreateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component specification (or '-' for stdin)")
	fieldToolUpdateCmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing fieldtool",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fieldToolClient.ListFieldTools(context.Background(), connect.NewRequest(&pb.ListFieldToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FieldTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			var data []byte
			var err error
			if filePath == "-" || (filePath == "" && hasStdinData()) {
				data, err = io.ReadAll(os.Stdin)
			} else if filePath != "" {
				data, err = os.ReadFile(filePath)
			} else {
				return fmt.Errorf("--file flag or piped stdin is required")
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
			}
			var item pb.FieldTool
			if err := protojson.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to parse fieldtool JSON: %w", err)
			}
			item.Id = args[0]
			res, err := fieldToolClient.UpdateFieldTool(context.Background(), connect.NewRequest(&pb.UpdateFieldToolRequest{FieldTool: &item}))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated fieldtool '%s'\n", res.Msg.Id)
			return printGetOutput(cmd.OutOrStdout(), res.Msg, nil)
		},
	}
	fieldToolUpdateCmd.Flags().StringP("file", "f", "", "Path to JSON/YAML file containing component updates (or '-' for stdin)")
	fieldToolDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete or archive an existing fieldtool",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := fieldToolClient.ListFieldTools(context.Background(), connect.NewRequest(&pb.ListFieldToolsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.FieldTools {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := fieldToolClient.DeleteFieldTool(context.Background(), connect.NewRequest(&pb.DeleteFieldToolRequest{Id: args[0]}))
			if err != nil {
				return err
			}
			if res.Msg.IsArchived {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' has active references and was archived/delisted.\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Component '%s' was permanently deleted.\n", args[0])
			}
			return nil
		},
	}
	fieldToolCmd.AddCommand(fieldToolCreateCmd)
	fieldToolCmd.AddCommand(fieldToolUpdateCmd)
	fieldToolCmd.AddCommand(fieldToolDeleteCmd)
	componentsGearCmd.AddCommand(fieldToolCmd)

	// --- EVALUATOR ---
	evalClient := apiconnect.NewEvaluatorServiceClient(httpClient, targetURL)
	evalCmd := &cobra.Command{
		Use:     "evaluate [build-id]",
		Aliases: []string{"eval"},
		Short:   "Run physics estimation and compatibility checks on a build",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(&pb.ListBuildsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Builds {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			buildId := args[0]
			payload, _ := cmd.Flags().GetFloat32("payload")
			battery, _ := cmd.Flags().GetString("battery")
			if battery == "" {
				lRes, err := evalClient.GetBuildElectricalLimits(context.Background(), connect.NewRequest(&pb.GetBuildElectricalLimitsRequest{
					BuildSource: &pb.GetBuildElectricalLimitsRequest_BuildId{
						BuildId: buildId,
					},
				}))
				if err != nil {
					return fmt.Errorf("failed to determine electrical limits for build: %w", err)
				}
				if lRes.Msg == nil || lRes.Msg.DefaultBatteryId == "" {
					return fmt.Errorf("no compatible battery found in database for build")
				}
				battery = lRes.Msg.DefaultBatteryId
			}
			eReq := &pb.EvaluateBuildRequest{
				BuildSource: &pb.EvaluateBuildRequest_BuildId{
					BuildId: buildId,
				},
				PayloadWeightG: payload,
				BatteryId:      battery,
			}
			eRes, err := evalClient.EvaluateBuild(context.Background(), connect.NewRequest(eReq))
			if err != nil {
				return fmt.Errorf("evaluation failed: %w", err)
			}

			return printGetOutput(cmd.OutOrStdout(), eRes.Msg, nil)
		},
	}
	evalCmd.Flags().Float32P("payload", "p", 0, "Payload weight in grams")
	evalCmd.Flags().StringP("battery", "b", "", "Battery ID or UUID (defaults to lightest compatible battery)")
	evalCmd.RegisterFlagCompletionFunc("battery", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var filter string
		if len(args) > 0 && args[0] != "" {
			lRes, err := evalClient.GetBuildElectricalLimits(context.Background(), connect.NewRequest(&pb.GetBuildElectricalLimitsRequest{
				BuildSource: &pb.GetBuildElectricalLimitsRequest_BuildId{
					BuildId: args[0],
				},
			}))
			if err == nil && lRes.Msg != nil {
				var clauses []string
				if lRes.Msg.MinVoltage > 0 {
					clauses = append(clauses, fmt.Sprintf("min_voltage >= %.2f", lRes.Msg.MinVoltage))
				}
				if lRes.Msg.MaxVoltage > 0 {
					clauses = append(clauses, fmt.Sprintf("max_voltage <= %.2f", lRes.Msg.MaxVoltage))
				}
				if lRes.Msg.MaxCurrentA > 0 {
					clauses = append(clauses, fmt.Sprintf("max_current_a >= %.2f", lRes.Msg.MaxCurrentA))
				}
				if len(clauses) > 0 {
					filter = strings.Join(clauses, " && ")
				}
			}
		}
		res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(&pb.ListBatteriesRequest{
			Filter:   filter,
			PageSize: 100,
		}))
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var comps []string
		for _, item := range res.Msg.Batteries {
			if strings.HasPrefix(item.Id, toComplete) {
				comps = append(comps, item.Id)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	})
	buildCmd.AddCommand(evalCmd)

	// --- CUSTOM COMPLETION ---
	completionCmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate the autocompletion script for the specified shell",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				err := rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), false)
				if err == nil {
					// Dynamically bind to the exact path used to invoke the binary
					if os.Args[0] != "qs" {
						fmt.Fprintf(cmd.OutOrStdout(), "\ncomplete -o default -o nospace -F __start_qs %q\n", os.Args[0])
					}
					// Also support dynamic loading via bash-completion
					fmt.Fprintln(cmd.OutOrStdout(), "if [[ -n \"$1\" && \"$1\" != \"qs\" && \"$1\" != \"\" ]]; then complete -o default -o nospace -F __start_qs \"$1\"; fi")
				}
				return err
			case "zsh":
				return rootCmd.GenZshCompletionNoDesc(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), false)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
			default:
				return nil
			}
		},
	}
	rootCmd.AddCommand(completionCmd)

	// --- SEARCH ---
	searchClient := apiconnect.NewSearchServiceClient(httpClient, targetURL)
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Fuzzy text completion search across collections",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			limit, _ := cmd.Flags().GetInt32("limit")
			if limit <= 0 {
				limit = 10
			}

			selectorStrs, _ := cmd.Flags().GetStringArray("selector")
			pathFlag, _ := cmd.Flags().GetString("path")
			filterFlag, _ := cmd.Flags().GetString("filter")

			var selectors []*pb.SearchSelector
			for _, s := range selectorStrs {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				parts := strings.SplitN(s, ":", 2)
				sel := &pb.SearchSelector{Path: strings.TrimSpace(parts[0])}
				if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
					f := strings.TrimSpace(parts[1])
					sel.Filter = &f
				}
				selectors = append(selectors, sel)
			}

			if pathFlag != "" {
				sel := &pb.SearchSelector{Path: strings.TrimSpace(pathFlag)}
				if filterFlag != "" {
					f := strings.TrimSpace(filterFlag)
					sel.Filter = &f
				}
				selectors = append(selectors, sel)
			}

			sReq := &pb.SearchRequest{
				Query:     query,
				Selectors: selectors,
				Limit:     limit,
			}

			sRes, err := searchClient.Search(context.Background(), connect.NewRequest(sReq))
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if jsonOut {
				out, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(sRes.Msg)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(sRes.Msg.Results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No results found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "COLLECTION\tID\tNAME\tSCORE\tDETAILS")
			for _, item := range sRes.Msg.Results {
				scoreStr := fmt.Sprintf("%.2f", item.MatchScore)
				var details []string
				if m, ok := item.Metadata["manufacturer"]; ok && m != "" {
					details = append(details, m)
				}
				if wt, ok := item.Metadata["weight_g"]; ok && wt != "" {
					if f, err := strconv.ParseFloat(wt, 64); err == nil {
						details = append(details, fmt.Sprintf("%.2fg", f))
					} else {
						details = append(details, wt+"g")
					}
				}
				if c, ok := item.Metadata["cell_count_s"]; ok && c != "" {
					details = append(details, c+"S")
				}
				if capMah, ok := item.Metadata["capacity_mah"]; ok && capMah != "" {
					details = append(details, capMah+"mAh")
				}
				detailStr := strings.Join(details, " • ")
				if detailStr == "" && item.Description != "" {
					detailStr = item.Description
					if len(detailStr) > 40 {
						detailStr = detailStr[:37] + "..."
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", item.CollectionName, item.Id, item.Name, scoreStr, detailStr)
			}
			return w.Flush()
		},
	}
	searchCmd.Flags().StringArrayP("selector", "s", nil, "Target collection selector with optional filter (<path>[:<filter>])")
	searchCmd.Flags().StringP("path", "p", "", "Target single collection path (e.g. batteries, frames)")
	searchCmd.Flags().StringP("filter", "f", "", "CEL filter expression for single collection path")
	searchCmd.Flags().Int32P("limit", "n", 10, "Maximum number of results to return")
	rootCmd.AddCommand(searchCmd)

	initAuthCommands(rootCmd, targetURL, httpClient)
	initBlobCommands(rootCmd, targetURL, httpClient)
	initUserCommands(rootCmd, targetURL, httpClient)

	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func GetCollectionPath(m proto.Message) string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, common.E_CollectionPath) {
		if p, ok := proto.GetExtension(opts, common.E_CollectionPath).(string); ok && p != "" {
			return strings.Trim(p, "/")
		}
	}
	return ""
}

func GetDefaultColumns(m proto.Message) []string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, common.E_Frontend) {
		if front, ok := proto.GetExtension(opts, common.E_Frontend).(*common.FrontendOptions); ok && front != nil && len(front.DefaultColumns) > 0 {
			return front.DefaultColumns
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

func formatValue(val interface{}) string {
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

func printGetOutput(out io.Writer, data interface{}, cols []string) error {
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

func printOutput(out io.Writer, data interface{}, cols []string, startOffset ...int) error {
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

	printTableTo(out, data, cols, startOffset...)
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

	b, _ := dataToJSON(data)
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
		maxIdx := startIdx + len(val) - 1
		numWidth := len(fmt.Sprintf("%d", maxIdx))

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
		for idx, item := range val {
			fmt.Fprintf(w, "%*d", numWidth, startIdx+idx)
			if len(orderedKeys) > 0 {
				fmt.Fprintf(w, "\t")
			}
			if m, ok := item.(map[string]interface{}); ok {
				for i, k := range orderedKeys {
					fmt.Fprintf(w, "%s", formatValue(m[k]))
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
			fmt.Fprintf(w, "%s\t%s\n", k, formatValue(val[k]))
		}
	default:
		fmt.Fprintf(out, "%v\n", v)
	}
	w.Flush()
}
