package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"

	"quadsmith/shared/pkg/config"

	configpb "quadsmith/shared/gen/quadsmith/config"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

var (
	embeddedConfig *configpb.OpsCliConfig
	opsServerURL   string
	tokenFlag      string
	jsonOut        bool
)

func init() {
	var cfg configpb.OpsCliConfig
	opts := prototext.UnmarshalOptions{DiscardUnknown: false}
	if err := opts.Unmarshal(embeddedConfigRaw, &cfg); err != nil {
		panic(fmt.Sprintf("failed to parse embedded ops CLI configuration: %v", err))
	}
	if err := config.ValidateOpsCliConfig(&cfg); err != nil {
		panic(fmt.Sprintf("invalid embedded ops CLI configuration: %v", err))
	}
	embeddedConfig = &cfg
	opsServerURL = cfg.GetOpsServerUrl()
	if v := os.Getenv("QS_OPS_SERVER_URL"); v != "" {
		opsServerURL = v
	}
}

type SessionConfig struct {
	Token     string    `json:"token"`
	Operator  string    `json:"operator,omitempty"`
	Group     string    `json:"group,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func sessionFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".quadsmith", "ops_session.json")
}

func loadSessionToken() string {
	if tokenFlag != "" {
		return tokenFlag
	}
	if v := os.Getenv("QS_OPS_TOKEN"); v != "" {
		return v
	}

	path := sessionFilePath()
	data, err := os.ReadFile(path)
	if err == nil {
		var s SessionConfig
		if err := json.Unmarshal(data, &s); err == nil && s.Token != "" {
			return s.Token
		}
	}

	// Default fallback in sandbox environment
	if embeddedConfig.GetEnvironment() == "sandbox" {
		return "qs_op_sandbox_token_12345"
	}
	return ""
}

func saveSession(token, operator, group string, expiresAt time.Time) error {
	path := sessionFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(SessionConfig{
		Token:     token,
		Operator:  operator,
		Group:     group,
		ExpiresAt: expiresAt,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func clearSession() error {
	path := sessionFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}

func getOpsClient() opsconnect.OpsServiceClient {
	token := loadSessionToken()
	transport := &bearerTransport{
		token: token,
		base:  http.DefaultTransport,
	}
	timeout := 60 * time.Second
	if embeddedConfig.GetTimeout() != nil && embeddedConfig.GetTimeout().AsDuration() > 0 {
		timeout = embeddedConfig.GetTimeout().AsDuration()
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	return opsconnect.NewOpsServiceClient(httpClient, opsServerURL)
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "qsops",
		Short: "Quadsmith Developer Operations CLI",
		Long:  "Operational and developer tooling for system diagnostics, IAM, JIT elevation, DB maintenance, background tasks, and scheduled jobs.",
	}

	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "Operator session or bootstrap token (env: QS_OPS_TOKEN)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output results in JSON format")

	// Auth & IAM commands
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLoginQuadsmithCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newIamCmd())

	// Top-level aliases
	rootCmd.AddCommand(newJitCmd())

	// Domain operation commands
	rootCmd.AddCommand(newSystemCmd())
	rootCmd.AddCommand(newDbCmd())
	rootCmd.AddCommand(newTasksCmd())
	rootCmd.AddCommand(newJobsCmd())
	rootCmd.AddCommand(newAuditCmd())

	return rootCmd
}

// ---------------------------------------------------------
// Auth & Identity Commands
// ---------------------------------------------------------

func newLoginCmd() *cobra.Command {
	var code string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub OAuth to acquire a session token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if code == "" {
				code = "dev-sandbox-code"
			}
			client := getOpsClient()
			resp, err := client.ExchangeOAuthCode(cmd.Context(), connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
				Code: code,
			}))
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			token := resp.Msg.GetToken()
			operator := resp.Msg.GetOperator()
			group := resp.Msg.GetGroupName()
			expiresAt := resp.Msg.GetExpiresAt().AsTime()

			if err := saveSession(token, operator, group, expiresAt); err != nil {
				return fmt.Errorf("failed to save session: %w", err)
			}

			fmt.Printf("✓ Successfully authenticated as %s (group: %s)\n", operator, group)
			fmt.Printf("  Session token stored in ~/.quadsmith/ops_session.json (expires: %s)\n",
				expiresAt.Format(time.RFC1123))
			return nil
		},
	}
	cmd.Flags().StringVar(&code, "code", "", "OAuth authorization code (defaults to sandbox dev code)")
	return cmd
}

type PublicCLIConfig struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	UserUUID     string `json:"user_uuid,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Email        string `json:"email,omitempty"`
}

func publicCLIConfigPath(customPath string) string {
	if customPath != "" {
		return customPath
	}
	if custom := os.Getenv("QS_CONFIG"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".quadsmith", "config.json")
}

func newLoginQuadsmithCmd() *cobra.Command {
	var qsConfigPath string
	cmd := &cobra.Command{
		Use:   "login-quadsmith",
		Short: "Authenticate the public 'qs' CLI as the @quadsmith system user (requires active JIT elevation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.LoginAsQuadsmith(cmd.Context(), connect.NewRequest(&pb.LoginAsQuadsmithRequest{}))
			if err != nil {
				return fmt.Errorf("login-quadsmith failed: %w", err)
			}

			msg := resp.Msg
			targetPath := publicCLIConfigPath(qsConfigPath)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			cliCfg := PublicCLIConfig{
				AccessToken:  msg.GetAccessToken(),
				RefreshToken: msg.GetRefreshToken(),
				UserUUID:     msg.GetUserUuid(),
				UserID:       msg.GetUsername(),
			}
			data, err := json.MarshalIndent(cliCfg, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal CLI config: %w", err)
			}
			if err := os.WriteFile(targetPath, data, 0600); err != nil {
				return fmt.Errorf("failed to write CLI config to %s: %w", targetPath, err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
				fmt.Println(string(out))
				return nil
			}

			fmt.Printf("✓ Authenticated public 'qs' CLI as @%s (role: %s)\n", msg.GetUsername(), msg.GetRole())
			fmt.Printf("  Session written to %s (expires: %s)\n", targetPath, msg.GetExpiresAt().AsTime().Format(time.RFC1123))
			return nil
		},
	}
	cmd.Flags().StringVar(&qsConfigPath, "qs-config", "", "Path to public qs CLI config.json (defaults to $QS_CONFIG or ~/.quadsmith/config.json)")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and remove local operator credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := clearSession(); err != nil {
				return fmt.Errorf("failed to clear session: %w", err)
			}
			fmt.Println("✓ Logged out successfully")
			return nil
		},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Inspect current operator identity, role, and active JIT grants",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetCurrentOperator(cmd.Context(), connect.NewRequest(&pb.GetCurrentOperatorRequest{}))
			if err != nil {
				return fmt.Errorf("whoami failed: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "OPERATOR:\t%s\n", msg.GetOperator())
			fmt.Fprintf(w, "TYPE:\t%s\n", msg.GetOperatorType())
			fmt.Fprintf(w, "GROUP:\t%s\n", msg.GetGroupName())
			if msg.GetSessionExpiresAt() != nil {
				fmt.Fprintf(w, "EXPIRES:\t%s\n", msg.GetSessionExpiresAt().AsTime().Format(time.RFC3339))
			}
			if g := msg.GetActiveJitGrant(); g != nil {
				fmt.Fprintf(w, "ACTIVE JIT:\t%s (Operation: %s, Expires: %s, Reason: %s)\n",
					g.GetId(), g.GetOperation(), g.GetExpiresAt().AsTime().Format(time.RFC3339), g.GetReason())
			} else {
				fmt.Fprintf(w, "ACTIVE JIT:\tNone (Standard permissions)\n")
			}
			return w.Flush()
		},
	}
}

// ---------------------------------------------------------
// IAM Commands
// ---------------------------------------------------------

func newIamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iam",
		Short: "Manage operator identities, declarative groups, tokens, and JIT leases",
	}

	cmd.AddCommand(newWhoamiCmd())
	cmd.AddCommand(newIamUsersCmd())
	cmd.AddCommand(newIamGroupsCmd())
	cmd.AddCommand(newIamTokensCmd())
	cmd.AddCommand(newJitCmd())

	return cmd
}

func newIamUsersCmd() *cobra.Command {
	var filter string
	var sortOpts []string
	cmd := &cobra.Command{
		Use:   "users",
		Short: "List active developer identities and session tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListIamUsersRequest{Sort: sortOpts}
			if strings.TrimSpace(filter) != "" {
				f := strings.TrimSpace(filter)
				req.Filter = &f
			}
			resp, err := client.ListIamUsers(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list IAM users: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "OPERATOR\tTYPE\tGROUP\tACTIVE JIT\tLAST ACTIVE\n")
			for _, u := range resp.Msg.GetUsers() {
				lastActive := "-"
				if u.GetLastActiveAt() != nil {
					lastActive = u.GetLastActiveAt().AsTime().Format("2006-01-02 15:04:05")
				}
				jitStr := "No"
				if u.GetHasActiveJit() {
					jitStr = "YES"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					u.GetOperator(), u.GetType(), u.GetGroupName(), jitStr, lastActive)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "CEL filter expression (e.g. 'group_name == \"admins\" && has_active_jit')")
	cmd.Flags().StringSliceVarP(&sortOpts, "sort", "s", nil, "Columns to sort by (e.g. operator, ^group_name)")
	return cmd
}

func newIamGroupsCmd() *cobra.Command {
	var filter string
	var sortOpts []string
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List declarative IAM groups and allowed RPC methods",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListIamGroupsRequest{Sort: sortOpts}
			if strings.TrimSpace(filter) != "" {
				f := strings.TrimSpace(filter)
				req.Filter = &f
			}
			resp, err := client.ListIamGroups(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list IAM groups: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "GROUP\tDESCRIPTION\tALLOWED METHODS\n")
			for _, g := range resp.Msg.GetGroups() {
				methods := strings.Join(g.GetAllowedMethods(), ", ")
				fmt.Fprintf(w, "%s\t%s\t%s\n", g.GetName(), g.GetDescription(), methods)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "CEL filter expression (e.g. 'name.contains(\"admin\")')")
	cmd.Flags().StringSliceVarP(&sortOpts, "sort", "s", nil, "Columns to sort by (e.g. name, ^name)")
	return cmd
}

func newIamTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage persistent operator and robot tokens",
	}

	var tokenFilter string
	var tokenSort []string
	listTokensCmd := &cobra.Command{
		Use:   "list",
		Short: "List issued internal tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListInternalTokensRequest{Sort: tokenSort}
			if strings.TrimSpace(tokenFilter) != "" {
				f := strings.TrimSpace(tokenFilter)
				req.Filter = &f
			}
			resp, err := client.ListInternalTokens(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list tokens: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TOKEN ID\tNAME\tTYPE\tGROUP\tEXPIRES\tSTATUS\n")
			for _, t := range resp.Msg.GetTokens() {
				status := "ACTIVE"
				if t.GetRevokedAt() != nil {
					status = "REVOKED"
				} else if t.GetExpiresAt() != nil && t.GetExpiresAt().AsTime().Before(time.Now()) {
					status = "EXPIRED"
				}
				expires := "Never"
				if t.GetExpiresAt() != nil {
					expires = t.GetExpiresAt().AsTime().Format("2006-01-02 15:04")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					t.GetId(), t.GetName(), t.GetType(), t.GetGroupName(), expires, status)
			}
			return w.Flush()
		},
	}
	listTokensCmd.Flags().StringVarP(&tokenFilter, "filter", "f", "", "CEL filter expression (e.g. 'type == \"ROBOT\" && status == \"ACTIVE\"')")
	listTokensCmd.Flags().StringSliceVarP(&tokenSort, "sort", "s", nil, "Columns to sort by (e.g. name, ^created_at)")
	cmd.AddCommand(listTokensCmd)

	var name, tokenType, group string
	var days int
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new internal operator or robot token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("--name is required")
			}
			client := getOpsClient()
			resp, err := client.CreateInternalToken(cmd.Context(), connect.NewRequest(&pb.CreateInternalTokenRequest{
				Name:         name,
				Type:         tokenType,
				GroupName:    group,
				DurationDays: int32(days),
			}))
			if err != nil {
				return fmt.Errorf("failed to create token: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			fmt.Println("✓ Internal token generated successfully:")
			fmt.Printf("  Token ID:     %s\n", resp.Msg.GetTokenId())
			fmt.Printf("  Token Secret: %s\n", resp.Msg.GetTokenSecret())
			fmt.Println("  WARNING: Save the secret token immediately; it cannot be viewed again!")
			return nil
		},
	}
	createCmd.Flags().StringVar(&name, "name", "", "Descriptive name for the token")
	createCmd.Flags().StringVar(&tokenType, "type", "HUMAN", "Token type ('HUMAN' or 'ROBOT')")
	createCmd.Flags().StringVar(&group, "group", "operators", "IAM group for the token")
	createCmd.Flags().IntVar(&days, "days", 30, "Duration in days")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke [token_id]",
		Short: "Revoke an internal token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.RevokeInternalToken(cmd.Context(), connect.NewRequest(&pb.RevokeInternalTokenRequest{
				Id: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to revoke token: %w", err)
			}
			if resp.Msg.GetSuccess() {
				fmt.Printf("✓ Token %s successfully revoked\n", args[0])
			} else {
				fmt.Printf("Token %s not found or already revoked\n", args[0])
			}
			return nil
		},
	})

	return cmd
}

// ---------------------------------------------------------
// JIT Elevation Commands
// ---------------------------------------------------------

func newJitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jit",
		Short: "Request, inspect, extend, and revoke hour-based JIT elevations",
	}

	var hours int
	var reason string
	var operation string
	requestCmd := &cobra.Command{
		Use:   "request",
		Short: "Acquire a temporary JIT elevation grant (1-12 hours)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.AcquireJitGrant(cmd.Context(), connect.NewRequest(&pb.AcquireJitGrantRequest{
				Hours:     int32(hours),
				Reason:    reason,
				Operation: operation,
			}))
			if err != nil {
				return fmt.Errorf("failed to acquire JIT grant: %w", err)
			}

			g := resp.Msg.GetGrant()
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(g)
				fmt.Println(string(out))
				return nil
			}

			fmt.Println("✓ JIT elevation grant acquired successfully:")
			fmt.Printf("  Grant ID:  %s\n", g.GetId())
			fmt.Printf("  Operator:  %s\n", g.GetOperator())
			fmt.Printf("  Hours:     %d hours\n", g.GetHours())
			fmt.Printf("  Expires:   %s\n", g.GetExpiresAt().AsTime().Format(time.RFC3339))
			fmt.Printf("  Operation: %s\n", g.GetOperation())
			fmt.Printf("  Reason:    %s\n", g.GetReason())
			return nil
		},
	}
	requestCmd.Flags().IntVar(&hours, "hours", 2, "Lease duration in integer hours (1 to 12)")
	requestCmd.Flags().StringVar(&reason, "reason", "Operational maintenance", "Reason for elevation request")
	requestCmd.Flags().StringVar(&operation, "operation", "*", "Target operation or '*' for all sensitive methods")
	cmd.AddCommand(requestCmd)

	var extendHours int
	extendCmd := &cobra.Command{
		Use:   "extend [grant_id]",
		Short: "Extend an active JIT grant by additional hours (up to 12h from now)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ExtendJitGrant(cmd.Context(), connect.NewRequest(&pb.ExtendJitGrantRequest{
				GrantId: args[0],
				Hours:   int32(extendHours),
			}))
			if err != nil {
				return fmt.Errorf("failed to extend JIT grant: %w", err)
			}

			g := resp.Msg.GetGrant()
			fmt.Printf("✓ JIT grant %s extended:\n", g.GetId())
			fmt.Printf("  New Expiration: %s\n", g.GetExpiresAt().AsTime().Format(time.RFC3339))
			return nil
		},
	}
	extendCmd.Flags().IntVar(&extendHours, "hours", 2, "Additional hours to add (max 12h from current time)")
	cmd.AddCommand(extendCmd)

	var allGrants bool
	var jitFilter string
	var jitSort []string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List active or historic JIT elevation grants",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListJitGrantsRequest{
				ActiveOnly: !allGrants,
				Sort:       jitSort,
			}
			if strings.TrimSpace(jitFilter) != "" {
				f := strings.TrimSpace(jitFilter)
				req.Filter = &f
			}
			resp, err := client.ListJitGrants(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list JIT grants: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "GRANT ID\tOPERATOR\tOPERATION\tHOURS\tSTATUS\tEXPIRES\tREASON\n")
			for _, g := range resp.Msg.GetGrants() {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
					g.GetId(), g.GetOperator(), g.GetOperation(), g.GetHours(), g.GetStatus(),
					g.GetExpiresAt().AsTime().Format("15:04:05"), g.GetReason())
			}
			return w.Flush()
		},
	}
	listCmd.Flags().BoolVar(&allGrants, "all", false, "Include revoked and expired grants")
	listCmd.Flags().StringVarP(&jitFilter, "filter", "f", "", "CEL filter expression (e.g. 'hours >= 2 && status == \"ACTIVE\"')")
	listCmd.Flags().StringSliceVarP(&jitSort, "sort", "s", nil, "Columns to sort by (e.g. ^hours, operator)")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke [grant_id]",
		Short: "Revoke an active JIT grant immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.RevokeJitGrant(cmd.Context(), connect.NewRequest(&pb.RevokeJitGrantRequest{
				GrantId: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to revoke JIT grant: %w", err)
			}
			if resp.Msg.GetSuccess() {
				fmt.Printf("✓ JIT grant %s successfully revoked\n", args[0])
			} else {
				fmt.Printf("JIT grant %s not found or already revoked\n", args[0])
			}
			return nil
		},
	})

	return cmd
}

// ---------------------------------------------------------
// System Diagnostics & Database Commands
// ---------------------------------------------------------

func newSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System, service health, and telemetry diagnostics",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "diag",
		Short: "Display ops plane and API server runtime diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetSystemDiagnostics(cmd.Context(), connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to get diagnostics: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "VERSION:\t%s\n", msg.GetVersion())
			fmt.Fprintf(w, "ENVIRONMENT:\t%s\n", msg.GetEnvironment())
			fmt.Fprintf(w, "UPTIME:\t%s\n", time.Duration(msg.GetUptimeSeconds())*time.Second)
			fmt.Fprintf(w, "MEMORY ALLOCATED:\t%.2f MB\n", float64(msg.GetMemoryAllocatedBytes())/(1024*1024))
			fmt.Fprintf(w, "GOROUTINES:\t%d\n", msg.GetNumGoroutines())
			fmt.Fprintf(w, "PUBLIC DB CONNS:\t%d\n", msg.GetActiveDatabaseConnections())
			fmt.Fprintf(w, "OPS DB CONNS:\t%d\n", msg.GetActiveInternalConnections())
			return w.Flush()
		},
	})

	var healthRegion string
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Inspect deep dependency health across all 5 Quadsmith Go services",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			reqMsg := &pb.GetSystemHealthRequest{}
			if strings.TrimSpace(healthRegion) != "" {
				r := strings.TrimSpace(healthRegion)
				reqMsg.RegionFilter = &r
			}
			resp, err := client.GetSystemHealth(cmd.Context(), connect.NewRequest(reqMsg))
			if err != nil {
				return fmt.Errorf("failed to get system health: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			fmt.Printf("OVERALL STATUS: %s (region: %s)\n\n", msg.GetStatus(), msg.GetRegion())
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "SERVICE\tREGION\tINSTANCE\tSTATUS\tUPTIME\tCHECKS\n")
			for _, svc := range msg.GetServices() {
				var checkSummaries []string
				for _, c := range svc.GetChecks() {
					checkSummaries = append(checkSummaries, fmt.Sprintf("%s=%s(%.1fms)", c.GetName(), c.GetStatus(), c.GetLatencyMs()))
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%ds\t%s\n",
					svc.GetServiceName(),
					svc.GetRegion(),
					svc.GetInstanceId(),
					svc.GetStatus(),
					svc.GetUptimeSeconds(),
					strings.Join(checkSummaries, ", "))
			}
			return w.Flush()
		},
	}
	healthCmd.Flags().StringVar(&healthRegion, "region", "", "Filter by deployment region (e.g. us-central1, local)")
	cmd.AddCommand(healthCmd)

	var metricsRegion, metricsService, metricsRoute, metricsWindow, metricsFilter string
	var metricsSort []string
	var metricsFailedOnly bool
	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Query RED (Rate, Error, Duration) and resource metrics across services and routes",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			reqMsg := &pb.GetSystemMetricsRequest{
				Sort: metricsSort,
			}
			if strings.TrimSpace(metricsWindow) != "" {
				w := strings.TrimSpace(metricsWindow)
				reqMsg.Window = &w
			}
			if strings.TrimSpace(metricsRegion) != "" {
				r := strings.TrimSpace(metricsRegion)
				reqMsg.RegionFilter = &r
			}
			if strings.TrimSpace(metricsService) != "" {
				s := strings.TrimSpace(metricsService)
				reqMsg.ServiceFilter = &s
			}
			if strings.TrimSpace(metricsFilter) != "" {
				f := strings.TrimSpace(metricsFilter)
				reqMsg.Filter = &f
			}

			resp, err := client.GetSystemMetrics(cmd.Context(), connect.NewRequest(reqMsg))
			if err != nil {
				return fmt.Errorf("failed to get system metrics: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "SERVICE\tREGION\tTOTAL REQS\tOK\tERRORS\tAVAILABILITY\tP50 (MS)\tP95 (MS)\tP99 (MS)\n")
			for _, s := range msg.GetServices() {
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%.2f%%\t%.1f\t%.1f\t%.1f\n",
					s.GetServiceName(),
					s.GetRegion(),
					s.GetTotalRequests(),
					s.GetSuccessCount(),
					s.GetErrorCount(),
					s.GetAvailabilityPercent(),
					s.GetP50RttMs(),
					s.GetP95RttMs(),
					s.GetP99RttMs())
			}
			_ = w.Flush()

			fmt.Println()
			rw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(rw, "SERVICE\tREGION\tROUTE\tMETHOD\tREQS\tERRORS\tAVAILABILITY\tP95 (MS)\n")
			for _, s := range msg.GetServices() {
				for _, r := range s.GetRoutes() {
					if metricsRoute != "" && !strings.Contains(strings.ToLower(r.GetRoute()), strings.ToLower(metricsRoute)) {
						continue
					}
					if metricsFailedOnly && r.GetErrorCount() == 0 {
						continue
					}
					fmt.Fprintf(rw, "%s\t%s\t%s\t%s\t%d\t%d\t%.2f%%\t%.1f\n",
						r.GetServiceName(),
						r.GetRegion(),
						r.GetRoute(),
						r.GetMethod(),
						r.GetTotalRequests(),
						r.GetErrorCount(),
						r.GetAvailabilityPercent(),
						r.GetP95RttMs())
				}
			}
			return rw.Flush()
		},
	}
	metricsCmd.Flags().StringVar(&metricsRegion, "region", "", "Filter by deployment region (e.g. us-central1, local)")
	metricsCmd.Flags().StringVar(&metricsService, "service", "", "Filter by service name (e.g. server-api, server-ops-api)")
	metricsCmd.Flags().StringVar(&metricsRoute, "route", "", "Substring filter for route or RPC procedure")
	metricsCmd.Flags().StringVar(&metricsWindow, "window", "5m", "Aggregation window (5m, 1h, 24h)")
	metricsCmd.Flags().BoolVar(&metricsFailedOnly, "failed-only", false, "Only show routes with non-zero error counts")
	metricsCmd.Flags().StringVarP(&metricsFilter, "filter", "f", "", "CEL filter expression for routes (e.g. 'p95_rtt_ms > 10.0 || error_count > 0')")
	metricsCmd.Flags().StringSliceVarP(&metricsSort, "sort", "s", nil, "Columns to sort routes by (e.g. ^p95_rtt_ms, ^total_requests)")
	cmd.AddCommand(metricsCmd)

	traceCmd := &cobra.Command{
		Use:   "trace [trace_id]",
		Short: "Inspect the full end-to-end multi-hop trace and logs for a request Trace ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetTrace(cmd.Context(), connect.NewRequest(&pb.GetTraceRequest{
				TraceId: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to get trace %s: %w", args[0], err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			t := resp.Msg
			fmt.Printf("TRACE ID:\t%s\n", t.GetTraceId())
			fmt.Printf("ROOT SERVICE:\t%s (%s)\n", t.GetRootService(), t.GetRegion())
			fmt.Printf("ROOT ROUTE:\t%s %s\n", t.GetRootMethod(), t.GetRootRoute())
			fmt.Printf("STATUS:\t\t%d (success=%t)\n", t.GetStatusCode(), t.GetSuccess())
			fmt.Printf("TOTAL RTT:\t%.2f ms\n\n", t.GetTotalRttMs())

			sw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(sw, "SERVICE\tREGION\tMETHOD\tOPERATION\tSTATUS\tDURATION (MS)\n")
			for _, sp := range t.GetSpans() {
				fmt.Fprintf(sw, "%s\t%s\t%s\t%s\t%d\t%.2f\n",
					sp.GetServiceName(),
					sp.GetRegion(),
					sp.GetMethod(),
					sp.GetOperation(),
					sp.GetStatusCode(),
					sp.GetDurationMs())
			}
			_ = sw.Flush()

			fmt.Println()
			lw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(lw, "TIMESTAMP\tLEVEL\tSERVICE\tMESSAGE\n")
			for _, lg := range t.GetLogs() {
				fmt.Fprintf(lw, "%s\t%s\t%s\t%s\n",
					lg.GetTimestamp().AsTime().Format(time.RFC3339),
					lg.GetLevel(),
					lg.GetServiceName(),
					lg.GetMessage())
			}
			return lw.Flush()
		},
	}
	cmd.AddCommand(traceCmd)

	return cmd
}

func newDbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Public database statistics",
	}

	var dbFilter string
	var dbSort []string
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show table row counts and disk sizes in quadsmith_api",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.GetDatabaseStatsRequest{Sort: dbSort}
			if strings.TrimSpace(dbFilter) != "" {
				f := strings.TrimSpace(dbFilter)
				req.Filter = &f
			}
			resp, err := client.GetDatabaseStats(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to get database stats: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			msg := resp.Msg
			fmt.Printf("Database: %s (Total Size: %.2f MB)\n\n",
				msg.GetDatabaseName(), float64(msg.GetTotalSizeBytes())/(1024*1024))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TABLE NAME\tROWS\tSIZE (KB)\n")
			for _, t := range msg.GetTables() {
				fmt.Fprintf(w, "%s\t%d\t%.1f KB\n",
					t.GetTableName(), t.GetRowCount(), float64(t.GetTotalSizeBytes())/1024)
			}
			return w.Flush()
		},
	}
	statsCmd.Flags().StringVarP(&dbFilter, "filter", "f", "", "CEL filter expression (e.g. 'row_count > 0')")
	statsCmd.Flags().StringSliceVarP(&dbSort, "sort", "s", nil, "Columns to sort by (e.g. ^row_count, table_name)")
	cmd.AddCommand(statsCmd)

	return cmd
}

// ---------------------------------------------------------
// Tasks, Jobs & Audit Commands
// ---------------------------------------------------------

func newTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Inspect and trigger background operational tasks",
	}

	var taskFilter string
	var taskSort []string
	listTasksCmd := &cobra.Command{
		Use:   "list",
		Short: "List registered operational maintenance tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListTasksRequest{Sort: taskSort}
			if strings.TrimSpace(taskFilter) != "" {
				f := strings.TrimSpace(taskFilter)
				req.Filter = &f
			}
			resp, err := client.ListTasks(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TASK NAME\tDESCRIPTION\tPROCEDURE\tDEFAULT INTERVAL\n")
			for _, t := range resp.Msg.GetTasks() {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					t.GetName(), t.GetDescription(), t.GetProcedure(), t.GetDefaultInterval())
			}
			return w.Flush()
		},
	}
	listTasksCmd.Flags().StringVarP(&taskFilter, "filter", "f", "", "CEL filter expression (e.g. 'name.contains(\"cleanup\")')")
	listTasksCmd.Flags().StringSliceVarP(&taskSort, "sort", "s", nil, "Columns to sort by (e.g. name, ^default_interval)")
	cmd.AddCommand(listTasksCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [task_name]",
		Short: "Get metadata and description for a specific task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetTask(cmd.Context(), connect.NewRequest(&pb.GetTaskRequest{
				TaskName: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to get task %s: %w", args[0], err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			t := resp.Msg.GetTask()
			fmt.Printf("Task Name:        %s\n", t.GetName())
			fmt.Printf("Description:      %s\n", t.GetDescription())
			fmt.Printf("Procedure:        %s\n", t.GetProcedure())
			fmt.Printf("Default Interval: %s\n", t.GetDefaultInterval())
			return nil
		},
	})

	triggerCmd := &cobra.Command{
		Use:   "trigger [task_name]",
		Short: "Immediately trigger execution of a background task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown task subcommand %q; run 'qsops tasks trigger --help' to see available tasks", args[0])
			}
			return cmd.Help()
		},
	}

	triggerCmd.AddCommand(&cobra.Command{
		Use:   "cleanup_expired_otps",
		Short: "Purge used and expired OTPs",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.CleanupExpiredOtps(cmd.Context(), connect.NewRequest(&pb.CleanupExpiredOtpsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to trigger cleanup_expired_otps: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Cleaned %d expired OTPs (Job ID: %s)\n", resp.Msg.GetCleanedCount(), jobID)
			} else {
				fmt.Printf("✓ Cleaned %d expired OTPs\n", resp.Msg.GetCleanedCount())
			}
			return nil
		},
	})

	triggerCmd.AddCommand(&cobra.Command{
		Use:   "cleanup_expired_tokens",
		Short: "Purge expired refresh tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.CleanupExpiredTokens(cmd.Context(), connect.NewRequest(&pb.CleanupExpiredTokensRequest{}))
			if err != nil {
				return fmt.Errorf("failed to trigger cleanup_expired_tokens: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Cleaned %d expired tokens (Job ID: %s)\n", resp.Msg.GetCleanedCount(), jobID)
			} else {
				fmt.Printf("✓ Cleaned %d expired tokens\n", resp.Msg.GetCleanedCount())
			}
			return nil
		},
	})

	triggerCmd.AddCommand(&cobra.Command{
		Use:   "cleanup_orphaned_blobs",
		Short: "Delete unreferenced blob objects older than 10 minutes",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.CleanupOrphanedBlobs(cmd.Context(), connect.NewRequest(&pb.CleanupOrphanedBlobsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to trigger cleanup_orphaned_blobs: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Cleaned %d orphaned blobs (Job ID: %s)\n", resp.Msg.GetCleanedCount(), jobID)
			} else {
				fmt.Printf("✓ Cleaned %d orphaned blobs\n", resp.Msg.GetCleanedCount())
			}
			return nil
		},
	})

	triggerCmd.AddCommand(&cobra.Command{
		Use:   "purge_soft_deleted_accounts",
		Short: "Anonymize and permanently purge soft-deleted user accounts past grace period",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.PurgeSoftDeletedAccounts(cmd.Context(), connect.NewRequest(&pb.PurgeSoftDeletedAccountsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to trigger purge_soft_deleted_accounts: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Purged %d soft-deleted accounts (Job ID: %s)\n", resp.Msg.GetPurgedCount(), jobID)
			} else {
				fmt.Printf("✓ Purged %d soft-deleted accounts\n", resp.Msg.GetPurgedCount())
			}
			return nil
		},
	})

	triggerCmd.AddCommand(&cobra.Command{
		Use:   "send_username_release_warnings",
		Short: "Send release warning notifications for archived usernames approaching expiration",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.SendUsernameReleaseWarnings(cmd.Context(), connect.NewRequest(&pb.SendUsernameReleaseWarningsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to trigger send_username_release_warnings: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Sent %d username release warnings (Job ID: %s)\n", resp.Msg.GetWarningCount(), jobID)
			} else {
				fmt.Printf("✓ Sent %d username release warnings\n", resp.Msg.GetWarningCount())
			}
			return nil
		},
	})

	var triggerUser string
	var triggerRole string

	setUserRoleCmd := &cobra.Command{
		Use:   "set_user_role",
		Short: "Update a user account role ('USER' or 'ADMIN')",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(triggerUser) == "" {
				return errors.New("--user <username|uuid> is required for set_user_role")
			}
			roleUpper := strings.ToUpper(strings.TrimSpace(triggerRole))
			if roleUpper != "USER" && roleUpper != "ADMIN" {
				return errors.New("--role must be 'USER' or 'ADMIN'")
			}
			client := getOpsClient()
			resp, err := client.SetUserRole(cmd.Context(), connect.NewRequest(&pb.SetUserRoleRequest{
				UserId: triggerUser,
				Role:   roleUpper,
			}))
			if err != nil {
				return fmt.Errorf("failed to trigger set_user_role: %w", err)
			}
			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}
			jobID := resp.Header().Get("X-Job-Id")
			if jobID != "" {
				fmt.Printf("✓ Updated role for @%s (%s) from %s to %s (Job ID: %s)\n",
					resp.Msg.GetUsername(), resp.Msg.GetUserUuid(), resp.Msg.GetPreviousRole(), resp.Msg.GetRole(), jobID)
			} else {
				fmt.Printf("✓ Updated role for @%s (%s) from %s to %s\n",
					resp.Msg.GetUsername(), resp.Msg.GetUserUuid(), resp.Msg.GetPreviousRole(), resp.Msg.GetRole())
			}
			return nil
		},
	}
	setUserRoleCmd.Flags().StringVar(&triggerUser, "user", "", "Target user identifier (username or UUID)")
	setUserRoleCmd.Flags().StringVar(&triggerRole, "role", "", "Target role ('USER' or 'ADMIN')")
	_ = setUserRoleCmd.MarkFlagRequired("user")
	_ = setUserRoleCmd.MarkFlagRequired("role")
	triggerCmd.AddCommand(setUserRoleCmd)

	cmd.AddCommand(triggerCmd)

	return cmd
}

func newJobsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Inspect execution history of background jobs",
	}

	var limit int
	var taskFilter string
	var jobFilter string
	var jobSort []string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List historical executions of background tasks (latest first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			listReq := &pb.ListJobsRequest{
				Sort: jobSort,
			}
			if limit > 0 {
				l := int32(limit)
				listReq.Limit = &l
			}
			if taskFilter != "" {
				listReq.TaskName = &taskFilter
			}
			if strings.TrimSpace(jobFilter) != "" {
				f := strings.TrimSpace(jobFilter)
				listReq.Filter = &f
			}
			resp, err := client.ListJobs(cmd.Context(), connect.NewRequest(listReq))
			if err != nil {
				return fmt.Errorf("failed to list jobs: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "JOB ID\tTASK\tPROCEDURE\tCALLER\tSTATUS\tDURATION\tCREATED AT\n")
			for _, j := range resp.Msg.GetJobs() {
				createdAt := "-"
				if j.GetCreatedAt() != nil {
					createdAt = j.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d ms\t%s\n",
					j.GetJobId(), j.GetTaskName(), j.GetProcedure(), j.GetCaller(), j.GetStatus(), j.GetDurationMs(), createdAt)
			}
			return w.Flush()
		},
	}
	listCmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of job records to return")
	listCmd.Flags().StringVar(&taskFilter, "task", "", "Filter jobs by task name (e.g. cleanup_expired_otps)")
	listCmd.Flags().StringVarP(&jobFilter, "filter", "f", "", "CEL filter expression (e.g. 'status == \"ERROR\" || duration_ms >= 50')")
	listCmd.Flags().StringSliceVarP(&jobSort, "sort", "s", nil, "Columns to sort by (e.g. ^duration_ms, ^created_at)")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [job_id]",
		Short: "Retrieve execution details for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetJob(cmd.Context(), connect.NewRequest(&pb.GetJobRequest{
				JobId: args[0],
			}))
			if err != nil {
				return fmt.Errorf("failed to get job %s: %w", args[0], err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			j := resp.Msg.GetJob()
			createdAt := "-"
			if j.GetCreatedAt() != nil {
				createdAt = j.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
			}

			fmt.Printf("Job ID:        %s\n", j.GetJobId())
			fmt.Printf("UUID:          %s\n", j.GetUuid())
			fmt.Printf("Task Name:     %s\n", j.GetTaskName())
			fmt.Printf("Procedure:     %s\n", j.GetProcedure())
			fmt.Printf("Caller:        %s (%s)\n", j.GetCaller(), j.GetCallerType())
			fmt.Printf("Status:        %s\n", j.GetStatus())
			fmt.Printf("Duration:      %d ms\n", j.GetDurationMs())
			fmt.Printf("Created At:    %s\n", createdAt)
			if j.GetErrorMessage() != "" {
				fmt.Printf("Error Message: %s\n", j.GetErrorMessage())
			}
			return nil
		},
	})

	return cmd
}

func newAuditCmd() *cobra.Command {
	var limit int
	var auditFilter string
	var auditSort []string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit log of internal API calls across all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			req := &pb.ListInternalApiCallsRequest{
				Limit: int32(limit),
				Sort:  auditSort,
			}
			if strings.TrimSpace(auditFilter) != "" {
				f := strings.TrimSpace(auditFilter)
				req.Filter = &f
			}
			resp, err := client.ListInternalApiCalls(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("failed to fetch audit log: %w", err)
			}

			if jsonOut {
				out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
				fmt.Println(string(out))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TIMESTAMP\tCALLER\tGROUP\tMETHOD\tSTATUS\tDURATION\tJIT GRANT\n")
			for _, c := range resp.Msg.GetCalls() {
				ts := c.GetCreatedAt().AsTime().Format("15:04:05")
				jit := "-"
				if c.GetJitGrantId() != "" {
					jit = c.GetJitGrantId()
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d ms\t%s\n",
					ts, c.GetCaller(), c.GetGroupName(), c.GetMethod(), c.GetStatus(), c.GetDurationMs(), jit)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of audit records to return")
	cmd.Flags().StringVarP(&auditFilter, "filter", "f", "", "CEL filter expression (e.g. 'status != \"OK\" || duration_ms >= 20')")
	cmd.Flags().StringSliceVarP(&auditSort, "sort", "s", nil, "Columns to sort by (e.g. ^duration_ms, ^created_at)")
	return cmd
}
