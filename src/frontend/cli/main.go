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
	rootCmd.PersistentFlags().BoolVar(&yamlOut, "yaml", false, "Output format as YAML")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	}

	// --- Motors ---
	motorClient := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, targetURL)
	motorsCmd := &cobra.Command{Use: "motors"}
	motorsListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	motorsListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	motorsListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	motorsListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	motorsListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	motorsListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	motorsListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	motorsListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	motorsCmd.AddCommand(motorsListCmd)

	motorsGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	motorsGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	motorsGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	motorsCmd.AddCommand(motorsGetCmd)

	rootCmd.AddCommand(motorsCmd)

	// --- Frames ---
	frameClient := quadsmithconnect.NewFrameServiceClient(http.DefaultClient, targetURL)
	framesCmd := &cobra.Command{Use: "frames"}
	framesListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	framesListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	framesListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	framesListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	framesListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	framesListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	framesListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	framesListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	framesCmd.AddCommand(framesListCmd)

	framesGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	framesGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	framesGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	framesCmd.AddCommand(framesGetCmd)

	rootCmd.AddCommand(framesCmd)

	// --- Batteries ---
	batteryClient := quadsmithconnect.NewBatteryServiceClient(http.DefaultClient, targetURL)
	batteriesCmd := &cobra.Command{Use: "batteries"}
	batteriesListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	batteriesListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	batteriesListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	batteriesListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	batteriesListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	batteriesListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	batteriesListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	batteriesListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	batteriesCmd.AddCommand(batteriesListCmd)

	batteriesGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	batteriesGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	batteriesGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	batteriesCmd.AddCommand(batteriesGetCmd)

	rootCmd.AddCommand(batteriesCmd)

	// --- Escs ---
	escClient := quadsmithconnect.NewEscServiceClient(http.DefaultClient, targetURL)
	escsCmd := &cobra.Command{Use: "escs"}
	escsListCmd := &cobra.Command{
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
			var all []*pb.Esc
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
				req := &pb.ListEscsRequest{Filter: filter, Columns: columns, Sort: sortOpts, PageSize: pageSize, PageToken: currentToken}
				res, err := escClient.ListEscs(context.Background(), connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Escs...)
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
				columns = GetDefaultColumns(&pb.Esc{})
			}
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	escsListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	escsListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	escsListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Esc{})
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
	escsListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	escsListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	escsListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Esc{})
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
	escsListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Esc{})
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
	escsCmd.AddCommand(escsListCmd)

	escsGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := escClient.ListEscs(context.Background(), connect.NewRequest(&pb.ListEscsRequest{PageSize: 100}))
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var comps []string
			for _, item := range res.Msg.Escs {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns, _ := cmd.Flags().GetStringSlice("column")
			req := &pb.GetEscRequest{Id: args[0], Columns: columns}
			res, err := escClient.GetEsc(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	escsGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	escsGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cols := GetColumns(&pb.Esc{})
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
	escsCmd.AddCommand(escsGetCmd)

	rootCmd.AddCommand(escsCmd)

	// --- FlightControllers ---
	flightcontrollerClient := quadsmithconnect.NewFlightControllerServiceClient(http.DefaultClient, targetURL)
	flightcontrollersCmd := &cobra.Command{Use: "flightcontrollers"}
	flightcontrollersListCmd := &cobra.Command{
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
				res, err := flightcontrollerClient.ListFlightControllers(context.Background(), connect.NewRequest(req))
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flightcontrollersListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	flightcontrollersListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	flightcontrollersListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	flightcontrollersListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	flightcontrollersListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	flightcontrollersListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	flightcontrollersListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	flightcontrollersCmd.AddCommand(flightcontrollersListCmd)

	flightcontrollersGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := flightcontrollerClient.ListFlightControllers(context.Background(), connect.NewRequest(&pb.ListFlightControllersRequest{PageSize: 100}))
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
			res, err := flightcontrollerClient.GetFlightController(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	flightcontrollersGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	flightcontrollersGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	flightcontrollersCmd.AddCommand(flightcontrollersGetCmd)

	rootCmd.AddCommand(flightcontrollersCmd)

	// --- Receivers ---
	receiverClient := quadsmithconnect.NewReceiverServiceClient(http.DefaultClient, targetURL)
	receiversCmd := &cobra.Command{Use: "receivers"}
	receiversListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	receiversListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	receiversListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	receiversListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	receiversListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	receiversListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	receiversListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	receiversListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	receiversCmd.AddCommand(receiversListCmd)

	receiversGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	receiversGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	receiversGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	receiversCmd.AddCommand(receiversGetCmd)

	rootCmd.AddCommand(receiversCmd)

	// --- VideoTransmitters ---
	videotransmitterClient := quadsmithconnect.NewVideoTransmitterServiceClient(http.DefaultClient, targetURL)
	videotransmittersCmd := &cobra.Command{Use: "videotransmitters"}
	videotransmittersListCmd := &cobra.Command{
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
				res, err := videotransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(req))
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	videotransmittersListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	videotransmittersListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	videotransmittersListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	videotransmittersListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	videotransmittersListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	videotransmittersListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	videotransmittersListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	videotransmittersCmd.AddCommand(videotransmittersListCmd)

	videotransmittersGetCmd := &cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			res, err := videotransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(&pb.ListVideoTransmittersRequest{PageSize: 100}))
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
			res, err := videotransmitterClient.GetVideoTransmitter(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	videotransmittersGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	videotransmittersGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	videotransmittersCmd.AddCommand(videotransmittersGetCmd)

	rootCmd.AddCommand(videotransmittersCmd)

	// --- Antennas ---
	antennaClient := quadsmithconnect.NewAntennaServiceClient(http.DefaultClient, targetURL)
	antennasCmd := &cobra.Command{Use: "antennas"}
	antennasListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	antennasListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	antennasListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	antennasListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	antennasListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	antennasListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	antennasListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	antennasListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	antennasCmd.AddCommand(antennasListCmd)

	antennasGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	antennasGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	antennasGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	antennasCmd.AddCommand(antennasGetCmd)

	rootCmd.AddCommand(antennasCmd)

	// --- Cameras ---
	cameraClient := quadsmithconnect.NewCameraServiceClient(http.DefaultClient, targetURL)
	camerasCmd := &cobra.Command{Use: "cameras"}
	camerasListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	camerasListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	camerasListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	camerasListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	camerasListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	camerasListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	camerasListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	camerasListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	camerasCmd.AddCommand(camerasListCmd)

	camerasGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	camerasGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	camerasGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	camerasCmd.AddCommand(camerasGetCmd)

	rootCmd.AddCommand(camerasCmd)

	// --- Propellers ---
	propellerClient := quadsmithconnect.NewPropellerServiceClient(http.DefaultClient, targetURL)
	propellersCmd := &cobra.Command{Use: "propellers"}
	propellersListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	propellersListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	propellersListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	propellersListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	propellersListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	propellersListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	propellersListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	propellersListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	propellersCmd.AddCommand(propellersListCmd)

	propellersGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	propellersGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	propellersGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	propellersCmd.AddCommand(propellersGetCmd)

	rootCmd.AddCommand(propellersCmd)

	// --- Builds ---
	buildClient := quadsmithconnect.NewBuildServiceClient(http.DefaultClient, targetURL)
	buildsCmd := &cobra.Command{Use: "builds"}
	buildsListCmd := &cobra.Command{
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
			err := printOutput(cmd.OutOrStdout(), all, columns)
			if err != nil {
				return err
			}
			return nil
		},
	}
	buildsListCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	buildsListCmd.Flags().Int32P("limit", "l", 0, "Maximum number of items to return")
	buildsListCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	buildsListCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	buildsListCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^kv)")
	buildsListCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	buildsListCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	buildsCmd.AddCommand(buildsListCmd)

	buildsGetCmd := &cobra.Command{
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
			return printOutput(cmd.OutOrStdout(), res.Msg, columns)
		},
	}
	buildsGetCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	buildsGetCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	buildsCmd.AddCommand(buildsGetCmd)

	rootCmd.AddCommand(buildsCmd)

	// --- EVALUATOR ---
	evalClient := quadsmithconnect.NewEvaluatorServiceClient(http.DefaultClient, targetURL)
	evalCmd := &cobra.Command{
		Use:   "evaluate [build-id]",
		Short: "Run physics estimation and compatibility checks on a build",
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
			bReq := &pb.GetBuildRequest{Id: args[0]}
			bRes, err := buildClient.GetBuild(context.Background(), connect.NewRequest(bReq))
			if err != nil {
				return fmt.Errorf("failed to fetch build: %w", err)
			}

			payload, _ := cmd.Flags().GetFloat32("payload")
			eReq := &pb.EvaluateBuildRequest{
				Build:          bRes.Msg,
				PayloadWeightG: payload,
			}
			eRes, err := evalClient.EvaluateBuild(context.Background(), connect.NewRequest(eReq))
			if err != nil {
				return fmt.Errorf("evaluation failed: %w", err)
			}

			return printOutput(cmd.OutOrStdout(), eRes.Msg, nil)
		},
	}
	evalCmd.Flags().Float32("payload", 0, "Payload weight in grams")
	rootCmd.AddCommand(evalCmd)

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
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
		return nil
	} else if yamlOut {
		if isNilSlice {
			fmt.Fprintln(out, "[]")
			return nil
		}
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
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
