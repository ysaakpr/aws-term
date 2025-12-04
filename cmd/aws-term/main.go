package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ysaakpr/aws-term/internal/browser"
	"github.com/ysaakpr/aws-term/internal/config"
	"github.com/ysaakpr/aws-term/internal/sso"
	"github.com/ysaakpr/aws-term/internal/ui"
)

var (
	version = "0.2.0"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	addProfile := flag.Bool("add", false, "Add a new SSO profile")
	listProfiles := flag.Bool("list", false, "List all configured profiles")
	setDefault := flag.String("set-default", "", "Set a profile as default")
	regionFlag := flag.String("region", "", "AWS region for SSO (default: us-east-1)")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("aws-term version %s\n", version)
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Print header
	ui.PrintHeader()

	// Load or create configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{Profiles: []config.Profile{}}
	}

	// Handle list profiles flag
	if *listProfiles {
		listAllProfiles(cfg)
		os.Exit(0)
	}

	// Handle set default flag
	if *setDefault != "" {
		if profile := cfg.GetProfileByName(*setDefault); profile != nil {
			cfg.SetDefault(*setDefault)
			if err := cfg.Save(); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to save config: %v", err))
				os.Exit(1)
			}
			ui.PrintSuccess(fmt.Sprintf("Set '%s' as the default profile", *setDefault))
		} else {
			ui.PrintError(fmt.Sprintf("Profile '%s' not found", *setDefault))
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle add profile flag
	if *addProfile {
		addNewProfile(cfg)
		os.Exit(0)
	}

	// Get profile name from positional argument
	profileName := ""
	if flag.NArg() > 0 {
		profileName = flag.Arg(0)
	}

	// Select profile to use
	var selectedProfile *config.Profile

	if profileName != "" {
		// User specified a profile name
		selectedProfile = cfg.GetProfileByName(profileName)
		if selectedProfile == nil {
			ui.PrintError(fmt.Sprintf("Profile '%s' not found", profileName))
			ui.PrintInfo("Use --list to see available profiles or --add to create a new one")
			os.Exit(1)
		}
	} else if len(cfg.Profiles) == 0 {
		// No profiles configured, prompt for new one
		selectedProfile = promptNewProfile(cfg)
	} else if len(cfg.Profiles) == 1 {
		// Only one profile, use it
		selectedProfile = &cfg.Profiles[0]
		ui.PrintInfo(fmt.Sprintf("Using profile: %s", selectedProfile.Name))
	} else {
		// Multiple profiles, check for default or prompt selection
		defaultProfile := cfg.GetDefaultProfile()
		if defaultProfile != nil {
			ui.PrintInfo(fmt.Sprintf("Using default profile: %s", defaultProfile.Name))
			selectedProfile = defaultProfile
		} else {
			// No default, show selection
			selected, err := ui.SelectProfile(cfg.Profiles)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to select profile: %v", err))
				os.Exit(1)
			}
			selectedProfile = selected
		}
	}

	// Determine region
	region := selectedProfile.Region
	if *regionFlag != "" {
		region = *regionFlag
	}
	if region == "" {
		region = sso.ExtractRegionFromURL(selectedProfile.SSOUrl)
	}

	// Detect available browsers
	browsers := browser.DetectBrowsers()
	if len(browsers) == 0 {
		ui.PrintError("No supported browsers found (Chrome, Safari, Firefox)")
		os.Exit(1)
	}

	// Select browser
	selectedBrowser, err := ui.SelectBrowser(browsers)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to select browser: %v", err))
		os.Exit(1)
	}

	// Create SSO client and authenticate
	ctx := context.Background()
	ssoClient := sso.NewSSOClient(selectedProfile.SSOUrl, region)

	// Authenticate using device authorization flow
	if err := ssoClient.Authenticate(ctx, selectedBrowser); err != nil {
		ui.PrintError(fmt.Sprintf("Authentication failed: %v", err))
		os.Exit(1)
	}

	// List available accounts
	ui.PrintInfo("Fetching available accounts...")
	accounts, err := ssoClient.ListAccounts(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list accounts: %v", err))
		os.Exit(1)
	}

	if len(accounts) == 0 {
		ui.PrintError("No accounts available for this SSO configuration")
		os.Exit(1)
	}

	// Select account
	selectedAccount, err := sso.SelectAccount(accounts)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to select account: %v", err))
		os.Exit(1)
	}

	// List roles for the selected account
	ui.PrintInfo(fmt.Sprintf("Fetching roles for %s...", selectedAccount.AccountName))
	roles, err := ssoClient.ListRoles(ctx, selectedAccount.AccountId)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list roles: %v", err))
		os.Exit(1)
	}

	if len(roles) == 0 {
		ui.PrintError("No roles available for this account")
		os.Exit(1)
	}

	// Select role
	selectedRole, err := sso.SelectRole(roles)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to select role: %v", err))
		os.Exit(1)
	}

	// Get credentials for the selected role
	ui.PrintInfo("Getting credentials...")
	creds, err := ssoClient.GetRoleCredentials(ctx, selectedAccount.AccountId, selectedRole.RoleName)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get credentials: %v", err))
		os.Exit(1)
	}

	// Save credentials to a file for sourcing
	credFile, err := sso.WriteCredentialsToFile(creds)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to write credentials file: %v", err))
	}

	// Print success and show how to use credentials
	ui.PrintSuccess("Credentials obtained successfully!")
	fmt.Println()
	fmt.Printf("  %sAccount:%s  %s (%s)\n", ui.ColorBold, ui.ColorReset, selectedAccount.AccountName, selectedAccount.AccountId)
	fmt.Printf("  %sRole:%s     %s\n", ui.ColorBold, ui.ColorReset, selectedRole.RoleName)
	fmt.Printf("  %sExpires:%s  %s\n", ui.ColorBold, ui.ColorReset, creds.Expiration.Local().Format(time.RFC1123))
	fmt.Println()

	// Determine the user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	fmt.Printf("To use these credentials, you can either:\n\n")
	if credFile != "" {
		fmt.Printf("  1. Source the credentials file:\n")
		fmt.Printf("     %ssource %s%s\n\n", ui.ColorCyan, credFile, ui.ColorReset)
		fmt.Printf("  2. Or copy these export commands:\n\n")
	} else {
		fmt.Printf("  Copy these export commands:\n\n")
	}
	fmt.Printf("     export AWS_ACCESS_KEY_ID=\"%s\"\n", creds.AccessKeyId)
	fmt.Printf("     export AWS_SECRET_ACCESS_KEY=\"%s\"\n", creds.SecretAccessKey)
	fmt.Printf("     export AWS_SESSION_TOKEN=\"%s\"\n\n", creds.SessionToken)

	// Print helpful verification commands
	fmt.Printf("%s%s─── Verify your session ───%s\n\n", ui.ColorBold, ui.ColorYellow, ui.ColorReset)
	fmt.Printf("  After setting credentials, run:\n\n")
	fmt.Printf("    %saws sts get-caller-identity%s\n", ui.ColorCyan, ui.ColorReset)
	fmt.Printf("    # Shows: Account ID, User ID, and ARN\n\n")
	fmt.Printf("    %saws s3 ls%s\n", ui.ColorCyan, ui.ColorReset)
	fmt.Printf("    # Lists S3 buckets (if you have permission)\n\n")

	// Ask if user wants to spawn a new shell with credentials
	response := ui.PromptInput("Open a new shell with these credentials? (Y/n)")
	if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		spawnShellWithCredentials(shell, creds, selectedAccount.AccountName, selectedRole.RoleName)
	}
}

func printHelp() {
	fmt.Printf(`aws-term - AWS SSO Terminal Session Manager

Usage:
  aws-term [options] [profile-name]

Options:
  --help            Show this help message
  --version         Show version information
  --add             Add a new SSO profile
  --list            List all configured profiles
  --set-default     Set a profile as the default
  --region          AWS region for SSO (default: auto-detect or us-east-1)

Examples:
  aws-term                    # Use default profile or show selection
  aws-term production         # Use the 'production' profile
  aws-term --add              # Add a new profile
  aws-term --set-default dev  # Set 'dev' as the default profile
  aws-term --region eu-west-1 # Use a specific region

Workflow:
  1. Select an SSO profile (or create one)
  2. Choose a browser for authentication
  3. Sign in via AWS SSO in your browser
  4. Select an AWS account
  5. Select a role
  6. Get temporary credentials

Configuration:
  Profiles are stored in ~/.aws-terminal/config.json
`)
}

func listAllProfiles(cfg *config.Config) {
	if len(cfg.Profiles) == 0 {
		ui.PrintInfo("No profiles configured. Use --add to create one.")
		return
	}

	fmt.Printf("\n%sConfigured profiles:%s\n\n", ui.ColorBold, ui.ColorReset)
	for _, p := range cfg.Profiles {
		defaultMarker := ""
		if p.Default {
			defaultMarker = fmt.Sprintf(" %s(default)%s", ui.ColorGreen, ui.ColorReset)
		}
		regionInfo := ""
		if p.Region != "" {
			regionInfo = fmt.Sprintf(" [%s]", p.Region)
		}
		fmt.Printf("  • %s%s%s%s%s\n", ui.ColorBold, p.Name, ui.ColorReset, regionInfo, defaultMarker)
		fmt.Printf("    %s%s%s\n", ui.ColorBlue, p.SSOUrl, ui.ColorReset)
	}
	fmt.Println()
}

func addNewProfile(cfg *config.Config) {
	profile := promptNewProfile(cfg)
	if profile == nil {
		return
	}

	cfg.AddProfile(*profile)
	if err := cfg.Save(); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save config: %v", err))
		os.Exit(1)
	}

	ui.PrintSuccess(fmt.Sprintf("Profile '%s' added successfully!", profile.Name))
}

func promptNewProfile(cfg *config.Config) *config.Profile {
	ssoUrl := ui.PromptSSOUrl()
	if ssoUrl == "" {
		ui.PrintError("SSO URL cannot be empty")
		return nil
	}

	// Clean up URL (remove trailing # or /#)
	ssoUrl = strings.TrimSuffix(ssoUrl, "#")
	ssoUrl = strings.TrimSuffix(ssoUrl, "/#")
	ssoUrl = strings.TrimSuffix(ssoUrl, "/")

	// Validate URL
	if err := sso.ValidateSSOUrl(ssoUrl); err != nil {
		ui.PrintError(fmt.Sprintf("Invalid SSO URL: %v", err))
		return nil
	}

	// Check if URL already exists
	if cfg.ProfileExists(ssoUrl) {
		ui.PrintInfo("This SSO URL already exists in your configuration.")
		for _, p := range cfg.Profiles {
			if p.SSOUrl == ssoUrl {
				return &p
			}
		}
	}

	// Prompt for profile name
	profileName := ui.PromptProfileName()
	if profileName == "" {
		ui.PrintError("Profile name cannot be empty")
		return nil
	}

	// Check if name already exists
	if cfg.GetProfileByName(profileName) != nil {
		ui.PrintError(fmt.Sprintf("Profile '%s' already exists", profileName))
		return nil
	}

	// Prompt for region (optional)
	region := ui.PromptInput("AWS Region (press Enter for us-east-1)")
	if region == "" {
		region = "us-east-1"
	}

	// Ask about default
	setAsDefault := len(cfg.Profiles) == 0 || ui.ConfirmSetDefault()

	profile := &config.Profile{
		Name:    profileName,
		SSOUrl:  ssoUrl,
		Region:  region,
		Default: setAsDefault,
	}

	cfg.AddProfile(*profile)
	if err := cfg.Save(); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save configuration: %v", err))
		return nil
	}

	ui.PrintSuccess(fmt.Sprintf("Profile '%s' saved!", profileName))
	return profile
}

func spawnShellWithCredentials(shell string, creds *sso.Credentials, accountName, roleName string) {
	// Set environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", creds.SessionToken)

	// Add markers to show we're in an AWS session
	os.Setenv("AWS_TERM_SESSION", "1")
	os.Setenv("AWS_TERM_ACCOUNT", accountName)
	os.Setenv("AWS_TERM_ROLE", roleName)

	fmt.Printf("\n%sStarting new shell with AWS credentials...%s\n", ui.ColorCyan, ui.ColorReset)
	fmt.Printf("%sAccount: %s | Role: %s%s\n", ui.ColorYellow, accountName, roleName, ui.ColorReset)
	fmt.Printf("%sType 'exit' to return to your original shell.%s\n\n", ui.ColorYellow, ui.ColorReset)

	// Spawn new shell
	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// On macOS/Linux, we can set a custom prompt indicator
	if runtime.GOOS != "windows" {
		// Try to prepend AWS indicator to existing prompt
		for i, env := range cmd.Env {
			if strings.HasPrefix(env, "PS1=") {
				cmd.Env[i] = fmt.Sprintf("PS1=[aws:%s] %s", roleName, strings.TrimPrefix(env, "PS1="))
				break
			}
		}
	}

	if err := cmd.Run(); err != nil {
		ui.PrintError(fmt.Sprintf("Shell exited with error: %v", err))
	} else {
		fmt.Printf("\n%sAWS session ended.%s\n", ui.ColorCyan, ui.ColorReset)
	}
}
