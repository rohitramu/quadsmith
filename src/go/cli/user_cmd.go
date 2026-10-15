package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"
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
		banActiveOnly bool
		banLimit      int32
		banJSON       bool
		banYAML       bool
	)

	runGet := func(cmd *cobra.Command, target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			return errors.New("target username or uuid required")
		}

		ctx := context.Background()
		res, err := userClient.GetUserBanStatus(ctx, connect.NewRequest(&pb.GetUserBanStatusRequest{
			UserId: target,
		}))
		if err != nil {
			return err
		}

		if banJSON {
			m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
			data, err := m.Marshal(res.Msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		if banYAML {
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

		if !res.Msg.IsBanned || res.Msg.ActiveBan == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "User @%s is not currently banned.\n", target)
			return nil
		}

		b := res.Msg.ActiveBan
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "BAN ID:\t%s\n", b.Uuid)
		fmt.Fprintf(w, "USER UUID:\t%s\n", b.UserUuid)
		fmt.Fprintf(w, "REASON:\t%s\n", b.Reason)
		if b.InternalNotes != "" {
			fmt.Fprintf(w, "NOTES:\t%s\n", b.InternalNotes)
		}
		if b.CreatedAt != nil {
			fmt.Fprintf(w, "ENACTED:\t%s\n", time.Unix(b.CreatedAt.Seconds, 0).Format(time.RFC3339))
		}
		if b.ExpiresAt != nil {
			fmt.Fprintf(w, "EXPIRES:\t%s\n", time.Unix(b.ExpiresAt.Seconds, 0).Format(time.RFC3339))
			fmt.Fprintf(w, "TYPE:\tTemporary\n")
		} else {
			fmt.Fprintf(w, "EXPIRES:\tNever\n")
			fmt.Fprintf(w, "TYPE:\tPermanent\n")
		}
		if b.BannedByUserUuid != "" {
			fmt.Fprintf(w, "BANNED BY:\t%s\n", b.BannedByUserUuid)
		}
		return w.Flush()
	}

	runList := func(cmd *cobra.Command, target string) error {
		target = strings.TrimSpace(target)
		var userFilter *string
		if target != "" {
			userFilter = &target
		}

		ctx := context.Background()
		res, err := userClient.ListBans(ctx, connect.NewRequest(&pb.ListBansRequest{
			UserId:     userFilter,
			ActiveOnly: banActiveOnly,
			PageSize:   banLimit,
		}))
		if err != nil {
			return err
		}

		if banJSON {
			m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
			data, err := m.Marshal(res.Msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		if banYAML {
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

		if len(res.Msg.Bans) == 0 {
			if target != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "No ban records found for user @%s.\n", target)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No ban records found.")
			}
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUSER\tSTATUS\tREASON\tENACTED\tEXPIRES")
		for _, b := range res.Msg.Bans {
			status := "ACTIVE"
			if b.RevokedAt != nil {
				status = "REVOKED"
			} else if b.ExpiresAt != nil && time.Unix(b.ExpiresAt.Seconds, 0).Before(time.Now()) {
				status = "EXPIRED"
			}

			enacted := "recent"
			if b.CreatedAt != nil {
				enacted = time.Unix(b.CreatedAt.Seconds, 0).Format("2006-01-02")
			}

			expires := "permanent"
			if b.ExpiresAt != nil {
				expires = time.Unix(b.ExpiresAt.Seconds, 0).Format("2006-01-02")
			}

			reason := b.Reason
			if len(reason) > 30 {
				reason = reason[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", b.Uuid, b.UserUuid, status, reason, enacted, expires)
		}
		return w.Flush()
	}

	runCreate := func(cmd *cobra.Command, target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			return errors.New("target username or uuid required")
		}
		if strings.TrimSpace(banReason) == "" {
			return errors.New("--reason is required")
		}

		var expiresAt *timestamppb.Timestamp
		if strings.TrimSpace(banExpiresIn) != "" {
			dur, err := parseDurationOrDays(banExpiresIn)
			if err != nil {
				return fmt.Errorf("invalid --expires-in value: %w", err)
			}
			expiresAt = timestamppb.New(time.Now().Add(dur))
		}

		ctx := context.Background()
		res, err := userClient.BanUser(ctx, connect.NewRequest(&pb.BanUserRequest{
			UserId:        target,
			Reason:        banReason,
			InternalNotes: banNotes,
			ExpiresAt:     expiresAt,
		}))
		if err != nil {
			return err
		}

		if banJSON {
			m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
			data, err := m.Marshal(res.Msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
		return nil
	}

	runUpdate := func(cmd *cobra.Command, target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			return errors.New("target username or uuid required")
		}

		var reasonPtr *string
		if strings.TrimSpace(banReason) != "" {
			r := strings.TrimSpace(banReason)
			reasonPtr = &r
		}

		var notesPtr *string
		if strings.TrimSpace(banNotes) != "" {
			n := strings.TrimSpace(banNotes)
			notesPtr = &n
		}

		var expiresAt *timestamppb.Timestamp
		if strings.TrimSpace(banExpiresIn) != "" {
			dur, err := parseDurationOrDays(banExpiresIn)
			if err != nil {
				return fmt.Errorf("invalid --expires-in value: %w", err)
			}
			expiresAt = timestamppb.New(time.Now().Add(dur))
		}

		ctx := context.Background()
		res, err := userClient.UpdateBan(ctx, connect.NewRequest(&pb.UpdateBanRequest{
			UserId:         target,
			Reason:         reasonPtr,
			InternalNotes:  notesPtr,
			ExpiresAt:      expiresAt,
			ClearExpiresAt: banPermanent,
		}))
		if err != nil {
			return err
		}

		if banJSON {
			m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
			data, err := m.Marshal(res.Msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
		return nil
	}

	runDelete := func(cmd *cobra.Command, target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			return errors.New("target username or uuid required")
		}

		reason := strings.TrimSpace(banReason)
		if reason == "" {
			reason = "Ban revoked by administrator"
		}

		ctx := context.Background()
		res, err := userClient.UnbanUser(ctx, connect.NewRequest(&pb.UnbanUserRequest{
			UserId:       target,
			RevokeReason: reason,
		}))
		if err != nil {
			return err
		}

		if banJSON {
			m := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: false}
			data, err := m.Marshal(res.Msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
		return nil
	}

	bansCmd := &cobra.Command{
		Use:     "bans <username|uuid> [get|list|create|update|delete]",
		Aliases: []string{"ban"},
		Short:   "Manage user bans and account restrictions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			first := strings.TrimSpace(args[0])
			switch strings.ToLower(first) {
			case "get":
				target := ""
				if len(args) > 1 {
					target = args[1]
				}
				return runGet(cmd, target)
			case "list":
				target := ""
				if len(args) > 1 {
					target = args[1]
				}
				return runList(cmd, target)
			case "create":
				target := ""
				if len(args) > 1 {
					target = args[1]
				}
				return runCreate(cmd, target)
			case "update":
				target := ""
				if len(args) > 1 {
					target = args[1]
				}
				return runUpdate(cmd, target)
			case "delete":
				target := ""
				if len(args) > 1 {
					target = args[1]
				}
				return runDelete(cmd, target)
			}

			target := first
			if len(args) == 1 {
				return runGet(cmd, target)
			}

			action := strings.ToLower(strings.TrimSpace(args[1]))
			switch action {
			case "get":
				return runGet(cmd, target)
			case "list":
				return runList(cmd, target)
			case "create":
				return runCreate(cmd, target)
			case "update":
				return runUpdate(cmd, target)
			case "delete":
				return runDelete(cmd, target)
			default:
				return fmt.Errorf("unknown action %q for user %s (valid: get, list, create, update, delete)", action, target)
			}
		},
	}

	bansCmd.PersistentFlags().StringVar(&banReason, "reason", "", "Infraction reason (or revocation reason for delete)")
	bansCmd.PersistentFlags().StringVar(&banNotes, "notes", "", "Internal administrator moderation notes")
	bansCmd.PersistentFlags().StringVar(&banExpiresIn, "expires-in", "", "Ban duration (e.g. 24h, 7d, 30d; omit for permanent)")
	bansCmd.PersistentFlags().BoolVar(&banPermanent, "permanent", false, "Convert ban to permanent (remove expiration on update)")
	bansCmd.PersistentFlags().BoolVar(&banActiveOnly, "active-only", false, "Filter list to only active bans")
	bansCmd.PersistentFlags().Int32Var(&banLimit, "limit", 50, "Maximum number of bans to list")
	bansCmd.PersistentFlags().BoolVar(&banJSON, "json", false, "Output in JSON format")
	bansCmd.PersistentFlags().BoolVar(&banYAML, "yaml", false, "Output in YAML format")

	getCmd := &cobra.Command{
		Use:   "get [username|uuid]",
		Short: "Get active ban details for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			return runGet(cmd, target)
		},
	}
	bansCmd.AddCommand(getCmd)

	listCmd := &cobra.Command{
		Use:   "list [username|uuid]",
		Short: "List ban history for a user (or all system bans)",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			return runList(cmd, target)
		},
	}
	bansCmd.AddCommand(listCmd)

	createCmd := &cobra.Command{
		Use:   "create [username|uuid]",
		Short: "Ban a user and suspend all linked email addresses",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			return runCreate(cmd, target)
		},
	}
	bansCmd.AddCommand(createCmd)

	updateCmd := &cobra.Command{
		Use:   "update [username|uuid]",
		Short: "Update an active ban's duration, reason, or internal notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			return runUpdate(cmd, target)
		},
	}
	bansCmd.AddCommand(updateCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete [username|uuid]",
		Short: "Unban a user and restore suspended account access",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			return runDelete(cmd, target)
		},
	}
	bansCmd.AddCommand(deleteCmd)

	userCmd.AddCommand(bansCmd)
	rootCmd.AddCommand(userCmd)
}
