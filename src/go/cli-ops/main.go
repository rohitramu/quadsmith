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
	jobspb "quadsmith/shared/gen/quadsmith/jobs"
	"quadsmith/shared/gen/quadsmith/jobs/jobsconnect"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

var (
	embeddedConfig *configpb.OpsCliConfig
	opsServerURL   string
	jobsServerURL  string
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
	jobsServerURL = cfg.GetJobsServerUrl()
	if v := os.Getenv("QS_JOBS_SERVER_URL"); v != "" {
		jobsServerURL = v
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

	// Default fallback in sandbox/test environments
	if embeddedConfig.GetEnvironment() == "sandbox" || embeddedConfig.GetEnvironment() == "test" {
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

func getJobsClient() jobsconnect.JobsServiceClient {
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
	url := jobsServerURL
	if url == "" {
		url = embeddedConfig.GetJobsServerUrl()
	}
	return jobsconnect.NewJobsServiceClient(httpClient, url)
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
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newIamCmd())

	// Top-level aliases
	rootCmd.AddCommand(newJitCmd())

	// Domain operation commands
	rootCmd.AddCommand(newSystemCmd())
	rootCmd.AddCommand(newDbCmd())
	rootCmd.AddCommand(newSeedCmd())
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
	return &cobra.Command{
		Use:   "users",
		Short: "List active developer identities and session tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListIamUsers(cmd.Context(), connect.NewRequest(&pb.ListIamUsersRequest{}))
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
}

func newIamGroupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "groups",
		Short: "List declarative IAM groups and allowed RPC methods",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListIamGroups(cmd.Context(), connect.NewRequest(&pb.ListIamGroupsRequest{}))
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
}

func newIamTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage persistent operator and robot tokens",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List issued internal tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListInternalTokens(cmd.Context(), connect.NewRequest(&pb.ListInternalTokensRequest{}))
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
	})

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
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List active or historic JIT elevation grants",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListJitGrants(cmd.Context(), connect.NewRequest(&pb.ListJitGrantsRequest{
				ActiveOnly: !allGrants,
			}))
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
		Short: "System and daemon diagnostics",
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

	return cmd
}

func newDbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Public database statistics and maintenance",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "stats",
		Short: "Show table row counts and disk sizes in quadsmith_api",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.GetDatabaseStats(cmd.Context(), connect.NewRequest(&pb.GetDatabaseStatsRequest{}))
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
	})

	var fullVacuum bool
	pruneCmd := &cobra.Command{
		Use:   "prune",
		Short: "Run VACUUM ANALYZE maintenance on the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.RunDatabaseMaintenance(cmd.Context(), connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{
				FullVacuum: fullVacuum,
			}))
			if err != nil {
				return fmt.Errorf("maintenance failed: %w", err)
			}
			fmt.Printf("✓ %s (duration: %d ms)\n", resp.Msg.GetMessage(), resp.Msg.GetDurationMs())
			return nil
		},
	}
	pruneCmd.Flags().BoolVar(&fullVacuum, "full", false, "Execute FULL vacuum (exclusive lock)")
	cmd.AddCommand(pruneCmd)

	return cmd
}

func newSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed hardware catalog and sample fixtures into public database",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Execute catalog seeding",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.SeedDatabase(cmd.Context(), connect.NewRequest(&pb.SeedDatabaseRequest{}))
			if err != nil {
				return fmt.Errorf("seeder failed: %w", err)
			}
			fmt.Printf("✓ %s\n", resp.Msg.GetMessage())
			return nil
		},
	})

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

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List registered operational maintenance tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListTasks(cmd.Context(), connect.NewRequest(&pb.ListTasksRequest{}))
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
	})

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

	cmd.AddCommand(&cobra.Command{
		Use:   "trigger [task_name]",
		Short: "Immediately trigger execution of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			ctx := cmd.Context()
			taskName := strings.ToLower(strings.TrimSpace(args[0]))

			switch taskName {
			case "cleanup_expired_otps", "cleanupexpiredotps":
				resp, err := client.CleanupExpiredOtps(ctx, connect.NewRequest(&pb.CleanupExpiredOtpsRequest{}))
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

			case "cleanup_expired_tokens", "cleanupexpiredtokens":
				resp, err := client.CleanupExpiredTokens(ctx, connect.NewRequest(&pb.CleanupExpiredTokensRequest{}))
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

			case "cleanup_orphaned_blobs", "cleanuporphanedblobs":
				resp, err := client.CleanupOrphanedBlobs(ctx, connect.NewRequest(&pb.CleanupOrphanedBlobsRequest{}))
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

			case "purge_soft_deleted_accounts", "purgesoftdeletedaccounts", "purge_accounts":
				resp, err := client.PurgeSoftDeletedAccounts(ctx, connect.NewRequest(&pb.PurgeSoftDeletedAccountsRequest{}))
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

			case "send_username_release_warnings", "sendusernamereleasewarnings", "warn_usernames":
				resp, err := client.SendUsernameReleaseWarnings(ctx, connect.NewRequest(&pb.SendUsernameReleaseWarningsRequest{}))
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

			case "run_database_maintenance", "rundatabasemaintenance", "vacuum":
				resp, err := client.RunDatabaseMaintenance(ctx, connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{
					FullVacuum: false,
				}))
				if err != nil {
					return fmt.Errorf("failed to trigger database maintenance: %w", err)
				}
				if jsonOut {
					out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
					fmt.Println(string(out))
					return nil
				}
				jobID := resp.Header().Get("X-Job-Id")
				if jobID != "" {
					fmt.Printf("✓ %s (duration: %d ms, Job ID: %s)\n", resp.Msg.GetMessage(), resp.Msg.GetDurationMs(), jobID)
				} else {
					fmt.Printf("✓ %s (duration: %d ms)\n", resp.Msg.GetMessage(), resp.Msg.GetDurationMs())
				}
				return nil

			case "seed_database", "seeddatabase", "seed":
				resp, err := client.SeedDatabase(ctx, connect.NewRequest(&pb.SeedDatabaseRequest{
					Force: false,
				}))
				if err != nil {
					return fmt.Errorf("failed to trigger seed_database: %w", err)
				}
				if jsonOut {
					out, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
					fmt.Println(string(out))
					return nil
				}
				jobID := resp.Header().Get("X-Job-Id")
				if jobID != "" {
					fmt.Printf("✓ %s (records seeded: %d, Job ID: %s)\n", resp.Msg.GetMessage(), resp.Msg.GetRecordsSeeded(), jobID)
				} else {
					fmt.Printf("✓ %s (records seeded: %d)\n", resp.Msg.GetMessage(), resp.Msg.GetRecordsSeeded())
				}
				return nil

			default:
				return fmt.Errorf("unknown task %q; run 'qsops tasks list' to see available tasks", args[0])
			}
		},
	})

	return cmd
}

func newJobsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Inspect execution history and schedules of jobs",
	}

	var limit int
	var taskFilter string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List historical executions of background tasks (latest first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getJobsClient()
			listReq := &jobspb.ListJobsRequest{}
			if limit > 0 {
				l := int32(limit)
				listReq.Limit = &l
			}
			if taskFilter != "" {
				listReq.TaskName = &taskFilter
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
			fmt.Fprintf(w, "JOB ID\tTASK\tPROCEDURE\tSTATUS\tDURATION\tCREATED AT\tSCHEDULE\n")
			for _, j := range resp.Msg.GetJobs() {
				createdAt := "-"
				if j.GetCreatedAt() != nil {
					createdAt = j.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
				}
				sched := "-"
				if j.GetScheduleInterval() != "" {
					sched = j.GetScheduleInterval()
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d ms\t%s\t%s\n",
					j.GetJobId(), j.GetTaskName(), j.GetProcedure(), j.GetStatus(), j.GetDurationMs(), createdAt, sched)
			}
			return w.Flush()
		},
	}
	listCmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of job records to return")
	listCmd.Flags().StringVar(&taskFilter, "task", "", "Filter jobs by task name (e.g. cleanup_expired_otps)")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [job_id]",
		Short: "Retrieve schedule information and execution details for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getJobsClient()
			resp, err := client.GetJob(cmd.Context(), connect.NewRequest(&jobspb.GetJobRequest{
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
			sched := "None"
			if j.GetScheduleInterval() != "" {
				sched = j.GetScheduleInterval()
			}

			fmt.Printf("Job ID:            %s\n", j.GetJobId())
			fmt.Printf("Task Name:         %s\n", j.GetTaskName())
			fmt.Printf("Procedure:         %s\n", j.GetProcedure())
			fmt.Printf("Caller:            %s (%s)\n", j.GetCaller(), j.GetCallerType())
			fmt.Printf("Status:            %s\n", j.GetStatus())
			fmt.Printf("Duration:          %d ms\n", j.GetDurationMs())
			fmt.Printf("Created At:        %s\n", createdAt)
			fmt.Printf("Schedule Interval: %s\n", sched)
			if j.GetErrorMessage() != "" {
				fmt.Printf("Error Message:     %s\n", j.GetErrorMessage())
			}
			return nil
		},
	})

	return cmd
}

func newAuditCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit log of internal API calls across all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpsClient()
			resp, err := client.ListInternalApiCalls(cmd.Context(), connect.NewRequest(&pb.ListInternalApiCallsRequest{
				Limit: int32(limit),
			}))
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
	return cmd
}
