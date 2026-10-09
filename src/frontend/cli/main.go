package main

import (
	"context"
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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

var (
	apiURL  = "http://localhost:8080"
	jsonOut bool
	yamlOut bool
)

func newRootCmd() *cobra.Command {
	jsonOut = false
	yamlOut = false
	if _, ok := os.LookupEnv("QS_COMPLETION_DESCRIPTIONS"); !ok {
		_ = os.Setenv("QS_COMPLETION_DESCRIPTIONS", "false")
	}
	targetURL := apiURL
	if url := os.Getenv("QS_API_URL"); url != "" {
		targetURL = url
	}

	rootCmd := &cobra.Command{
		Use:   "qs",
		Short: "Quadsmith CLI",
	}
	rootCmd.CompletionOptions.DisableDescriptions = true

	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output format as JSON")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	}

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

	// --- Antennas ---
	antennaClient := quadsmithconnect.NewAntennaServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(antennaCmd)

	// --- Batteries ---
	batteryClient := quadsmithconnect.NewBatteryServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(batteryCmd)

	// --- Builds ---
	buildClient := quadsmithconnect.NewBuildServiceClient(http.DefaultClient, targetURL)
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
	rootCmd.AddCommand(buildCmd)

	// --- Cameras ---
	cameraClient := quadsmithconnect.NewCameraServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(cameraCmd)

	// --- ElectronicSpeedControllers ---
	electronicSpeedControllerClient := quadsmithconnect.NewElectronicSpeedControllerServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(electronicSpeedControllerCmd)

	// --- FlightControllers ---
	flightControllerClient := quadsmithconnect.NewFlightControllerServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(flightControllerCmd)

	// --- Frames ---
	frameClient := quadsmithconnect.NewFrameServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(frameCmd)

	// --- GpsReceivers ---
	gpsReceiverClient := quadsmithconnect.NewGpsReceiverServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(gpsReceiverCmd)

	// --- Motors ---
	motorClient := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(motorCmd)

	// --- Propellers ---
	propellerClient := quadsmithconnect.NewPropellerServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(propellerCmd)

	// --- Receivers ---
	receiverClient := quadsmithconnect.NewReceiverServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(receiverCmd)

	// --- VideoTransmitters ---
	videoTransmitterClient := quadsmithconnect.NewVideoTransmitterServiceClient(http.DefaultClient, targetURL)
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
	componentsHardwareCmd.AddCommand(videoTransmitterCmd)

	// --- EVALUATOR ---
	evalClient := quadsmithconnect.NewEvaluatorServiceClient(http.DefaultClient, targetURL)
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
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, targetURL)
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
					details = append(details, wt+"g")
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

	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func GetCollectionPath(m proto.Message) string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, pb.E_CollectionPath) {
		if p, ok := proto.GetExtension(opts, pb.E_CollectionPath).(string); ok && p != "" {
			return strings.Trim(p, "/")
		}
	}
	return ""
}

func GetDefaultColumns(m proto.Message) []string {
	opts := m.ProtoReflect().Descriptor().Options()
	if proto.HasExtension(opts, pb.E_Frontend) {
		if front, ok := proto.GetExtension(opts, pb.E_Frontend).(*pb.FrontendOptions); ok && front != nil && len(front.DefaultColumns) > 0 {
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
