package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sigs.k8s.io/yaml"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

func parseDurationOrDays(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration: %w", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func initUserCommands(rootCmd *cobra.Command, targetURL string, httpClient *http.Client) {
	userClient := apiconnect.NewUserServiceClient(httpClient, targetURL)

	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users and perform administrative moderation",
	}

	var (
		banReason     string
		banNotes      string
		banExpiresIn  string
		banPermanent  bool
		banRevoke     bool
		banActiveOnly bool
		banLimit      int32
		banYAML       bool
	)

	banColumns := []string{"uuid", "user_uuid", "banned_by_user_uuid", "reason", "internal_notes", "expires_at", "created_at", "revoked_at", "revoked_by_user_uuid", "revoke_reason"}

	getCmd := &cobra.Command{
		Use:   "get <username|uuid>",
		Short: "Get active ban details for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			columns, _ := cmd.Flags().GetStringSlice("column")
			yamlFlag, _ := cmd.Flags().GetBool("yaml")

			ctx := context.Background()
			res, err := userClient.GetUserBanStatus(ctx, connect.NewRequest(&pb.GetUserBanStatusRequest{
				UserId: target,
			}))
			if err != nil {
				return err
			}

			if !res.Msg.IsBanned || res.Msg.ActiveBan == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "User @%s is not currently banned.\n", target)
				return nil
			}

			yamlOut = yamlFlag
			return printGetOutput(cmd.OutOrStdout(), res.Msg.ActiveBan, columns)
		},
	}
	getCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	getCmd.Flags().BoolVar(&banYAML, "yaml", false, "Output format as YAML")
	_ = getCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range banColumns {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})

	listCmd := &cobra.Command{
		Use:   "list [username|uuid]",
		Short: "List ban history for a user (or all system bans)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = strings.TrimSpace(args[0])
			}
			var userFilter *string
			if target != "" {
				userFilter = &target
			}

			filter, _ := cmd.Flags().GetString("filter")
			columns, _ := cmd.Flags().GetStringSlice("column")
			sortOpts, _ := cmd.Flags().GetStringSlice("sort")
			limit, _ := cmd.Flags().GetInt32("limit")
			activeOnly, _ := cmd.Flags().GetBool("active-only")
			yamlFlag, _ := cmd.Flags().GetBool("yaml")

			if limit < 0 {
				cmd.SilenceUsage = false
				return fmt.Errorf("limit cannot be negative")
			}

			pageSize := int32(50)
			if limit > 0 && limit < pageSize {
				pageSize = limit
			}

			ctx := context.Background()
			var all []*pb.Ban
			var currentToken string
			for {
				req := &pb.ListBansRequest{
					UserId:     userFilter,
					ActiveOnly: activeOnly,
					PageSize:   pageSize,
					PageToken:  currentToken,
					Filter:     filter,
					Sort:       sortOpts,
					Columns:    columns,
				}
				res, err := userClient.ListBans(ctx, connect.NewRequest(req))
				if err != nil {
					return err
				}
				all = append(all, res.Msg.Bans...)
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
				columns = []string{"uuid", "user_uuid", "reason", "created_at", "expires_at"}
			}

			yamlOut = yamlFlag
			return printOutput(cmd.OutOrStdout(), all, columns)
		},
	}
	listCmd.Flags().StringP("filter", "f", "", "CEL filter string")
	listCmd.Flags().StringSliceP("column", "c", nil, "Columns to select")
	listCmd.Flags().StringSliceP("sort", "s", nil, "Columns to sort by (e.g. ^created_at)")
	listCmd.Flags().Int32VarP(&banLimit, "limit", "l", 50, "Maximum number of bans to list")
	listCmd.Flags().BoolVar(&banActiveOnly, "active-only", false, "Filter list to only active bans")
	listCmd.Flags().BoolVar(&banYAML, "yaml", false, "Output in YAML format")

	_ = listCmd.RegisterFlagCompletionFunc("column", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		selected, _ := cmd.Flags().GetStringSlice("column")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[s] = true
		}
		var filtered []string
		for _, c := range banColumns {
			if !selectedMap[c] && strings.HasPrefix(c, toComplete) {
				filtered = append(filtered, c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})
	_ = listCmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		selected, _ := cmd.Flags().GetStringSlice("sort")
		selectedMap := make(map[string]bool)
		for _, s := range selected {
			selectedMap[strings.TrimPrefix(s, "^")] = true
		}
		isDesc := strings.HasPrefix(toComplete, "^")
		cleanPrefix := strings.TrimPrefix(toComplete, "^")
		var filtered []string
		for _, c := range banColumns {
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
	_ = listCmd.RegisterFlagCompletionFunc("filter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		re := regexp.MustCompile(`([a-zA-Z_]+)$`)
		match := re.FindStringSubmatch(toComplete)
		prefix := ""
		base := toComplete
		if len(match) > 0 {
			prefix = match[1]
			base = toComplete[:len(toComplete)-len(prefix)]
		}
		var filtered []string
		for _, c := range banColumns {
			if len(prefix) == 0 || strings.HasPrefix(c, prefix) {
				filtered = append(filtered, base+c)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	})

	createCmd := &cobra.Command{
		Use:   "create <username|uuid>",
		Short: "Ban a user and suspend all linked email addresses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			reason, _ := cmd.Flags().GetString("reason")
			if strings.TrimSpace(reason) == "" {
				return errors.New("--reason is required")
			}
			notes, _ := cmd.Flags().GetString("notes")
			expiresIn, _ := cmd.Flags().GetString("expires-in")
			yamlFlag, _ := cmd.Flags().GetBool("yaml")

			var expiresAt *timestamppb.Timestamp
			if strings.TrimSpace(expiresIn) != "" {
				dur, err := parseDurationOrDays(expiresIn)
				if err != nil {
					return fmt.Errorf("invalid --expires-in value: %w", err)
				}
				expiresAt = timestamppb.New(time.Now().Add(dur))
			}

			ctx := context.Background()
			res, err := userClient.BanUser(ctx, connect.NewRequest(&pb.BanUserRequest{
				UserId:        target,
				Reason:        reason,
				InternalNotes: notes,
				ExpiresAt:     expiresAt,
			}))
			if err != nil {
				return err
			}

			yamlOut = yamlFlag
			if jsonOut {
				m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
				data, err := m.Marshal(res.Msg)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if yamlOut {
				m := protojson.MarshalOptions{EmitUnpopulated: false}
				data, err := m.Marshal(res.Msg)
				if err != nil {
					return err
				}
				y, err := yaml.JSONToYAML(data)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), string(y))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
			return nil
		},
	}
	createCmd.Flags().StringVar(&banReason, "reason", "", "Infraction reason (required)")
	_ = createCmd.MarkFlagRequired("reason")
	createCmd.Flags().StringVar(&banNotes, "notes", "", "Internal administrator moderation notes")
	createCmd.Flags().StringVar(&banExpiresIn, "expires-in", "", "Ban duration (e.g. 24h, 7d, 30d; omit for permanent)")
	createCmd.Flags().BoolVar(&banYAML, "yaml", false, "Output in YAML format")

	updateCmd := &cobra.Command{
		Use:   "update <username|uuid>",
		Short: "Update an active ban's duration, reason, or internal notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			reason, _ := cmd.Flags().GetString("reason")
			notes, _ := cmd.Flags().GetString("notes")
			expiresIn, _ := cmd.Flags().GetString("expires-in")
			permanent, _ := cmd.Flags().GetBool("permanent")
			revoke, _ := cmd.Flags().GetBool("revoke")
			yamlFlag, _ := cmd.Flags().GetBool("yaml")

			var reasonPtr *string
			if strings.TrimSpace(reason) != "" {
				r := strings.TrimSpace(reason)
				reasonPtr = &r
			}

			var notesPtr *string
			if strings.TrimSpace(notes) != "" {
				n := strings.TrimSpace(notes)
				notesPtr = &n
			}

			var expiresAt *timestamppb.Timestamp
			if strings.TrimSpace(expiresIn) != "" {
				dur, err := parseDurationOrDays(expiresIn)
				if err != nil {
					return fmt.Errorf("invalid --expires-in value: %w", err)
				}
				expiresAt = timestamppb.New(time.Now().Add(dur))
			}

			var revokePtr *bool
			if revoke {
				revokePtr = &revoke
			}

			ctx := context.Background()
			res, err := userClient.UpdateBan(ctx, connect.NewRequest(&pb.UpdateBanRequest{
				UserId:         target,
				Reason:         reasonPtr,
				InternalNotes:  notesPtr,
				ExpiresAt:      expiresAt,
				ClearExpiresAt: permanent,
				Revoke:         revokePtr,
				RevokeReason:   reasonPtr,
			}))
			if err != nil {
				return err
			}

			yamlOut = yamlFlag
			if jsonOut {
				m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
				data, err := m.Marshal(res.Msg)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if yamlOut {
				m := protojson.MarshalOptions{EmitUnpopulated: false}
				data, err := m.Marshal(res.Msg)
				if err != nil {
					return err
				}
				y, err := yaml.JSONToYAML(data)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), string(y))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
			return nil
		},
	}
	updateCmd.Flags().StringVar(&banReason, "reason", "", "Updated infraction reason")
	updateCmd.Flags().StringVar(&banNotes, "notes", "", "Updated internal administrator moderation notes")
	updateCmd.Flags().StringVar(&banExpiresIn, "expires-in", "", "Updated ban duration (e.g. 24h, 7d, 30d)")
	updateCmd.Flags().BoolVar(&banPermanent, "permanent", false, "Convert ban to permanent (remove expiration)")
	updateCmd.Flags().BoolVar(&banRevoke, "revoke", false, "Revoke/lift the ban immediately")
	updateCmd.Flags().BoolVar(&banYAML, "yaml", false, "Output in YAML format")

	bansCmd := &cobra.Command{
		Use:     "bans",
		Aliases: []string{"ban"},
		Short:   "Manage user bans and moderation history",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	bansCmd.AddCommand(getCmd)
	bansCmd.AddCommand(listCmd)
	bansCmd.AddCommand(createCmd)
	bansCmd.AddCommand(updateCmd)

	userCmd.AddCommand(bansCmd)
	rootCmd.AddCommand(userCmd)
}
