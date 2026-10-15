package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	// qs user ban <username|uuid> --reason="..." [--notes="..."] [--expires-in="..."]
	var banReason, banNotes, banExpiresIn string
	banCmd := &cobra.Command{
		Use:   "ban <username|uuid>",
		Short: "Ban a user and suspend all linked email addresses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
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

			fmt.Printf("✓ %s (Ban ID: %s)\n", res.Msg.Message, res.Msg.Ban.Uuid)
			return nil
		},
	}
	banCmd.Flags().StringVar(&banReason, "reason", "", "Public reason for ban (required)")
	banCmd.Flags().StringVar(&banNotes, "notes", "", "Internal admin notes")
	banCmd.Flags().StringVar(&banExpiresIn, "expires-in", "", "Ban duration (e.g. 24h, 7d, 30d; omit for permanent)")
	_ = banCmd.MarkFlagRequired("reason")
	userCmd.AddCommand(banCmd)

	// qs user unban <username|uuid> --reason="..."
	var unbanReason string
	unbanCmd := &cobra.Command{
		Use:   "unban <username|uuid>",
		Short: "Unban a user and restore suspended account access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			if target == "" {
				return errors.New("target username or uuid required")
			}
			if strings.TrimSpace(unbanReason) == "" {
				return errors.New("--reason is required")
			}

			ctx := context.Background()
			res, err := userClient.UnbanUser(ctx, connect.NewRequest(&pb.UnbanUserRequest{
				UserId:       target,
				RevokeReason: unbanReason,
			}))
			if err != nil {
				return err
			}

			fmt.Printf("✓ %s\n", res.Msg.Message)
			return nil
		},
	}
	unbanCmd.Flags().StringVar(&unbanReason, "reason", "", "Reason for lifting the ban (required)")
	_ = unbanCmd.MarkFlagRequired("reason")
	userCmd.AddCommand(unbanCmd)

	// qs user bans [--user=<username|uuid>] [--active-only]
	var listUserFilter string
	var listActiveOnly bool
	bansCmd := &cobra.Command{
		Use:   "bans",
		Short: "List administrative bans",
		RunE: func(cmd *cobra.Command, args []string) error {
			var filterUser *string
			if strings.TrimSpace(listUserFilter) != "" {
				u := strings.TrimSpace(listUserFilter)
				filterUser = &u
			}

			ctx := context.Background()
			res, err := userClient.ListBans(ctx, connect.NewRequest(&pb.ListBansRequest{
				UserId:     filterUser,
				ActiveOnly: listActiveOnly,
			}))
			if err != nil {
				return err
			}

			if len(res.Msg.Bans) == 0 {
				fmt.Println("No bans found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "BAN ID\tTARGET USER\tBANNED BY\tREASON\tEXPIRES\tSTATUS")
			for _, b := range res.Msg.Bans {
				status := "ACTIVE"
				if b.RevokedAt != nil {
					status = "REVOKED"
				} else if b.ExpiresAt != nil && time.Now().After(time.Unix(b.ExpiresAt.Seconds, 0)) {
					status = "EXPIRED"
				}

				expires := "Permanent"
				if b.ExpiresAt != nil {
					expires = time.Unix(b.ExpiresAt.Seconds, 0).Format("2006-01-02 15:04")
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					b.Uuid, b.UserUuid, b.BannedByUserUuid, b.Reason, expires, status)
			}
			return w.Flush()
		},
	}
	bansCmd.Flags().StringVar(&listUserFilter, "user", "", "Filter bans by user ID or handle")
	bansCmd.Flags().BoolVar(&listActiveOnly, "active-only", false, "Only list active bans")
	userCmd.AddCommand(bansCmd)

	// qs user ban-status [username|uuid] [--json]
	var banStatusJSON bool
	banStatusCmd := &cobra.Command{
		Use:   "ban-status [username|uuid]",
		Short: "Check the ban status of a user or the current account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var target string
			if len(args) > 0 {
				target = strings.TrimSpace(args[0])
			}

			ctx := context.Background()
			res, err := userClient.GetUserBanStatus(ctx, connect.NewRequest(&pb.GetUserBanStatusRequest{
				UserId: target,
			}))
			if err != nil {
				return err
			}

			if banStatusJSON {
				data, err := json.MarshalIndent(res.Msg, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			if !res.Msg.IsBanned || res.Msg.ActiveBan == nil {
				fmt.Println("Status: ACTIVE (Not banned)")
				return nil
			}

			b := res.Msg.ActiveBan
			expires := "Permanent"
			if b.ExpiresAt != nil {
				expires = time.Unix(b.ExpiresAt.Seconds, 0).Format("2006-01-02 15:04:05 UTC")
			}

			fmt.Println("Status: BANNED")
			fmt.Printf("Ban ID: %s\n", b.Uuid)
			fmt.Printf("Reason: %s\n", b.Reason)
			fmt.Printf("Expires: %s\n", expires)
			fmt.Printf("Banned By: %s\n", b.BannedByUserUuid)
			return nil
		},
	}
	banStatusCmd.Flags().BoolVar(&banStatusJSON, "json", false, "Output in JSON format")
	userCmd.AddCommand(banStatusCmd)

	rootCmd.AddCommand(userCmd)
}
