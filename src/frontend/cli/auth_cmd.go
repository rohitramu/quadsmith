package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
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

type authTransport struct {
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
	authClient := quadsmithconnect.NewAuthServiceClient(httpClient, targetURL)

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
		Short: "Display the currently authenticated pilot account",
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

			fmt.Printf("Pilot Handle: @%s\n", res.Msg.Id)
			fmt.Printf("Email:        %s\n", res.Msg.Email)
			fmt.Printf("Role:         %s\n", res.Msg.Role)
			fmt.Printf("UUID:         %s\n", res.Msg.Uuid)
			return nil
		},
	}
	rootCmd.AddCommand(whoamiCmd)

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
