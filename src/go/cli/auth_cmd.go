package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

type CLIConfig struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	UserUUID     string `json:"user_uuid,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Email        string `json:"email,omitempty"`
}

func configFilePath() string {
	if custom := os.Getenv("QS_CONFIG"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".quadsmith", "config.json")
}

func loadCLIConfig() (*CLIConfig, error) {
	path := configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &CLIConfig{}, nil
	}
	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &CLIConfig{}, nil
	}
	return &cfg, nil
}

func saveCLIConfig(cfg *CLIConfig) error {
	path := configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func clearCLIConfig() error {
	path := configFilePath()
	_ = os.Remove(path)
	return nil
}

var activeTargetURL string

func SetActiveTargetURL(url string) {
	activeTargetURL = url
}

func GetActiveTargetURL() string {
	return activeTargetURL
}

type authTransport struct {
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if activeTargetURL != "" {
		if target, err := url.Parse(activeTargetURL); err == nil {
			req = req.Clone(req.Context())
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			if target.Path != "" && target.Path != "/" {
				req.URL.Path = strings.TrimRight(target.Path, "/") + "/" + strings.TrimLeft(req.URL.Path, "/")
			}
		}
	}
	if req.Header.Get("Authorization") == "" {
		if envToken := os.Getenv("QS_ACCESS_TOKEN"); envToken != "" {
			req.Header.Set("Authorization", "Bearer "+envToken)
		} else if cfg, err := loadCLIConfig(); err == nil && cfg.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
		}
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func newCLIHTTPClient() *http.Client {
	return &http.Client{
		Transport: &authTransport{base: http.DefaultTransport},
	}
}

func initAuthCommands(rootCmd *cobra.Command, targetURL string, httpClient *http.Client) {
	authClient := apiconnect.NewAuthServiceClient(httpClient, targetURL)

	// login
	loginCmd := &cobra.Command{
		Use:   "login [email]",
		Short: "Authenticate with Quadsmith using passwordless email OTP",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			var email string
			if len(args) > 0 {
				email = strings.TrimSpace(args[0])
			} else {
				fmt.Print("Enter your email address: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				email = strings.TrimSpace(input)
			}

			if email == "" {
				return errors.New("email address cannot be empty")
			}

			ctx := context.Background()
			fmt.Printf("Sending verification code to %s...\n", email)
			sendRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{Email: email}))
			if err != nil {
				return fmt.Errorf("failed to send OTP: %w", err)
			}

			fmt.Println(sendRes.Msg.Message)
			fmt.Print("Enter the 6-digit verification code: ")
			codeInput, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			code := strings.TrimSpace(codeInput)
			if code == "" {
				return errors.New("verification code cannot be empty")
			}

			verifyRes, err := authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
				Email: email,
				Code:  code,
			}))
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			cfg := &CLIConfig{
				AccessToken:  verifyRes.Msg.AccessToken,
				RefreshToken: verifyRes.Msg.RefreshToken,
				UserUUID:     verifyRes.Msg.User.Uuid,
				UserID:       verifyRes.Msg.User.Id,
				Email:        verifyRes.Msg.User.Email,
			}
			if err := saveCLIConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Successfully authenticated as @%s (%s)\n", cfg.UserID, cfg.Email)
			return nil
		},
	}
	rootCmd.AddCommand(loginCmd)

	// whoami
	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display the currently authenticated user account",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCLIConfig()
			hasToken := (err == nil && cfg.AccessToken != "") || os.Getenv("QS_ACCESS_TOKEN") != ""
			if !hasToken {
				fmt.Println("Not logged in. Run 'qs login' to authenticate.")
				return nil
			}

			ctx := context.Background()
			res, err := authClient.GetCurrentUser(ctx, connect.NewRequest(&pb.GetCurrentUserRequest{}))
			if err != nil {
				return fmt.Errorf("failed to get user info (session may have expired, run 'qs login'): %w", err)
			}

			fmt.Printf("Username:     @%s\n", res.Msg.Id)
			fmt.Printf("Email:        %s\n", res.Msg.Email)
			fmt.Printf("Role:         %s\n", res.Msg.Role)
			return nil
		},
	}
	rootCmd.AddCommand(whoamiCmd)

	// email root command
	emailCmd := &cobra.Command{
		Use:   "email",
		Short: "Manage associated login emails for your Quadsmith account",
	}

	var jsonOutput bool

	// email list
	emailListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all email addresses associated with your account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			res, err := authClient.ListUserEmails(ctx, connect.NewRequest(&pb.ListUserEmailsRequest{}))
			if err != nil {
				return fmt.Errorf("failed to list emails: %w", err)
			}

			if jsonOutput {
				data, err := json.MarshalIndent(res.Msg.Emails, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "EMAIL\tPRIMARY\tSTATUS\tCREATED")
			for _, em := range res.Msg.Emails {
				isPrimary := "no"
				if em.IsPrimary {
					isPrimary = "yes"
				}
				status := "verified"
				created := "recent"
				if em.CreatedAt != nil {
					created = time.Unix(em.CreatedAt.Seconds, 0).Format("2006-01-02")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", em.Email, isPrimary, status, created)
			}
			return w.Flush()
		},
	}
	emailListCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	emailCmd.AddCommand(emailListCmd)

	// email add
	emailAddCmd := &cobra.Command{
		Use:   "add [email]",
		Short: "Associate a new email address with your account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			var email string
			if len(args) > 0 {
				email = strings.TrimSpace(args[0])
			} else {
				fmt.Print("Enter the email address to associate: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				email = strings.TrimSpace(input)
			}
			if email == "" {
				return errors.New("email address cannot be empty")
			}

			ctx := context.Background()
			fmt.Printf("Sending verification code to %s...\n", email)
			sendRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
				Email:   email,
				Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
			}))
			if err != nil {
				return fmt.Errorf("failed to send OTP: %w", err)
			}

			fmt.Println(sendRes.Msg.Message)
			fmt.Print("Enter the 6-digit verification code: ")
			codeInput, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			code := strings.TrimSpace(codeInput)
			if code == "" {
				return errors.New("verification code cannot be empty")
			}

			verifyRes, err := authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
				Email:   email,
				Code:    code,
				Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
			}))
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			if verifyRes.Msg.MergeRequired {
				fmt.Printf("\n⚠️  Notice: This email is already associated with another account (@%s).\n", verifyRes.Msg.OtherUserUsername)
				fmt.Printf("To merge that account's builds and data into your current account, run:\n")
				fmt.Printf("  qs account merge %s\n\n", email)
				return nil
			}

			fmt.Printf("✓ Successfully added %s to your account.\n", email)
			return nil
		},
	}
	emailCmd.AddCommand(emailAddCmd)

	// email set-primary
	emailSetPrimaryCmd := &cobra.Command{
		Use:   "set-primary <email>",
		Short: "Set an associated email address as the primary email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := strings.TrimSpace(args[0])
			ctx := context.Background()
			_, err := authClient.SetPrimaryEmail(ctx, connect.NewRequest(&pb.SetPrimaryEmailRequest{Email: email}))
			if err != nil {
				return fmt.Errorf("failed to set primary email: %w", err)
			}
			fmt.Printf("✓ %s is now your primary account email.\n", email)
			return nil
		},
	}
	emailCmd.AddCommand(emailSetPrimaryCmd)

	// email remove
	emailRemoveCmd := &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove an associated email address from your account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := strings.TrimSpace(args[0])
			ctx := context.Background()
			_, err := authClient.RemoveUserEmail(ctx, connect.NewRequest(&pb.RemoveUserEmailRequest{Email: email}))
			if err != nil {
				return fmt.Errorf("failed to remove email: %w", err)
			}
			fmt.Printf("✓ Removed %s from your account.\n", email)
			return nil
		},
	}
	emailCmd.AddCommand(emailRemoveCmd)
	rootCmd.AddCommand(emailCmd)

	// account root command
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "Manage your Quadsmith account settings and consolidations",
	}

	// account merge
	accountMergeCmd := &cobra.Command{
		Use:   "merge [email]",
		Short: "Merge another Quadsmith account into your current account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			var email string
			if len(args) > 0 {
				email = strings.TrimSpace(args[0])
			} else {
				fmt.Print("Enter the email address of the account to merge: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				email = strings.TrimSpace(input)
			}
			if email == "" {
				return errors.New("email address cannot be empty")
			}

			ctx := context.Background()
			fmt.Printf("Sending verification code to %s...\n", email)
			sendRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
				Email:   email,
				Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
			}))
			if err != nil {
				return fmt.Errorf("failed to send OTP: %w", err)
			}

			fmt.Println(sendRes.Msg.Message)
			fmt.Printf("\n⚠️  WARNING: Merging will permanently transfer all drone builds, flight logs,\n")
			fmt.Printf("and data from the other account into your currently authenticated account.\n")
			fmt.Printf("The address %s will be added as a login email.\n\n", email)

			fmt.Print("Enter the 6-digit verification code sent to that address: ")
			codeInput, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			code := strings.TrimSpace(codeInput)
			if code == "" {
				return errors.New("verification code cannot be empty")
			}

			fmt.Print("Are you sure you want to proceed with this merge? (y/N): ")
			confirmInput, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			confirm := strings.ToLower(strings.TrimSpace(confirmInput))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Merge cancelled.")
				return nil
			}

			mergeRes, err := authClient.MergeAccount(ctx, connect.NewRequest(&pb.MergeAccountRequest{
				Email: email,
				Code:  code,
			}))
			if err != nil {
				return fmt.Errorf("merge failed: %w", err)
			}

			fmt.Printf("\n✓ Accounts merged successfully!\n")
			fmt.Printf("Transferred builds: %d\n", mergeRes.Msg.TransferredBuildCount)
			fmt.Printf("Consolidated email: %s\n", email)
			return nil
		},
	}
	var exportOutput string
	var exportCode string

	accountExportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export a complete GDPR personal data archive (.zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			cfg, err := loadCLIConfig()
			if err != nil || (cfg.AccessToken == "" && os.Getenv("QS_ACCESS_TOKEN") == "") {
				return errors.New("authentication required. Please run 'qs login' first")
			}

			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Initiating GDPR data archive export...")
			sendRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
				Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA,
			}))
			if err != nil {
				return fmt.Errorf("failed to initiate data export challenge: %w", err)
			}

			challengeID := sendRes.Msg.ChallengeId
			primaryEmail := sendRes.Msg.Email
			if primaryEmail == "" {
				primaryEmail = "your primary email"
			}

			code := strings.TrimSpace(exportCode)
			if code == "" {
				fmt.Printf("Verification code sent to %s.\n", primaryEmail)
				fmt.Print("Enter the 6-digit verification code: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				code = strings.TrimSpace(input)
			}
			if code == "" {
				return errors.New("verification code cannot be empty")
			}

			verifyRes, err := authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
				Email:       sendRes.Msg.Email,
				ChallengeId: challengeID,
				Code:        code,
				Purpose:     pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_EXPORT_DATA,
			}))
			if err != nil {
				return fmt.Errorf("failed to verify data export: %w", err)
			}

			exportToken := verifyRes.Msg.GetExportToken()
			downloadURL := verifyRes.Msg.GetDownloadUrl()
			suggestedFilename := verifyRes.Msg.GetFilename()
			if suggestedFilename == "" {
				suggestedFilename = fmt.Sprintf("quadsmith-data-%s.zip", time.Now().UTC().Format("2006-01-02"))
			}

			outputPath := exportOutput
			if outputPath == "" {
				outputPath = suggestedFilename
			}

			baseURL := strings.TrimRight(targetURL, "/")
			if active := GetActiveTargetURL(); active != "" {
				baseURL = strings.TrimRight(active, "/")
			}
			fullURL := fmt.Sprintf("%s%s", baseURL, downloadURL)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create download request: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+exportToken)

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to download archive: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("download failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
			}

			outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outFile.Close()

			written, err := io.Copy(outFile, resp.Body)
			if err != nil {
				return fmt.Errorf("failed to save archive: %w", err)
			}

			fmt.Printf("✓ Data archive saved to %s (%d bytes)\n", outputPath, written)
			return nil
		},
	}
	accountExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Destination file path for the .zip archive")
	accountExportCmd.Flags().StringVar(&exportCode, "code", "", "6-digit verification code sent to your primary email")
	accountCmd.AddCommand(accountExportCmd)

	accountCmd.AddCommand(accountMergeCmd)
	rootCmd.AddCommand(accountCmd)

	// logout
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear stored session credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := loadCLIConfig()
			if cfg != nil && cfg.RefreshToken != "" {
				ctx := context.Background()
				_, _ = authClient.Logout(ctx, connect.NewRequest(&pb.LogoutRequest{
					RefreshToken: &cfg.RefreshToken,
				}))
			}

			_ = clearCLIConfig()
			fmt.Println("✓ Successfully logged out.")
			return nil
		},
	}
	rootCmd.AddCommand(logoutCmd)
}
