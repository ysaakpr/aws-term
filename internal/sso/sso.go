package sso

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/ysaakpr/aws-term/internal/browser"
	"github.com/ysaakpr/aws-term/internal/ui"
)

const (
	ClientName = "aws-term"
	ClientType = "public"
	GrantType  = "urn:ietf:params:oauth:grant-type:device_code"
)

// Credentials represents AWS credentials
type Credentials struct {
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// Account represents an AWS account
type Account struct {
	AccountId   string
	AccountName string
	EmailAddress string
}

// Role represents an AWS role
type Role struct {
	RoleName  string
	AccountId string
}

// SSOClient handles AWS SSO operations using the SDK
type SSOClient struct {
	StartURL    string
	Region      string
	oidcClient  *ssooidc.Client
	ssoClient   *sso.Client
	accessToken string
}

// NewSSOClient creates a new SSO client
func NewSSOClient(startURL, region string) *SSOClient {
	// Create OIDC client for device authorization
	oidcClient := ssooidc.New(ssooidc.Options{
		Region: region,
	})

	// Create SSO client for account/role listing and credentials
	ssoClient := sso.New(sso.Options{
		Region: region,
	})

	return &SSOClient{
		StartURL:   startURL,
		Region:     region,
		oidcClient: oidcClient,
		ssoClient:  ssoClient,
	}
}

// Authenticate performs the SSO device authorization flow
func (c *SSOClient) Authenticate(ctx context.Context, browserName string) error {
	// Step 1: Register the client
	ui.PrintInfo("Registering client with AWS SSO...")
	
	registerOutput, err := c.oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(ClientName),
		ClientType: aws.String(ClientType),
		Scopes:     []string{"sso:account:access"},
	})
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}

	clientId := aws.ToString(registerOutput.ClientId)
	clientSecret := aws.ToString(registerOutput.ClientSecret)

	// Step 2: Start device authorization
	ui.PrintInfo("Starting device authorization...")
	
	deviceAuthOutput, err := c.oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(clientId),
		ClientSecret: aws.String(clientSecret),
		StartUrl:     aws.String(c.StartURL),
	})
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %w", err)
	}

	verificationUri := aws.ToString(deviceAuthOutput.VerificationUriComplete)
	userCode := aws.ToString(deviceAuthOutput.UserCode)
	deviceCode := aws.ToString(deviceAuthOutput.DeviceCode)
	expiresIn := deviceAuthOutput.ExpiresIn
	interval := deviceAuthOutput.Interval

	// Step 3: Open browser for user to authorize
	fmt.Println()
	fmt.Printf("%s%s════════════════════════════════════════════%s\n", ui.ColorBold, ui.ColorCyan, ui.ColorReset)
	fmt.Printf("%s  Opening browser for AWS SSO login...%s\n", ui.ColorYellow, ui.ColorReset)
	fmt.Printf("%s════════════════════════════════════════════%s\n", ui.ColorCyan, ui.ColorReset)
	fmt.Println()
	fmt.Printf("  If browser doesn't open, visit:\n")
	fmt.Printf("  %s%s%s\n", ui.ColorBlue, verificationUri, ui.ColorReset)
	fmt.Println()
	fmt.Printf("  Verification code: %s%s%s\n", ui.ColorBold, userCode, ui.ColorReset)
	fmt.Println()

	if err := browser.OpenURL(browserName, verificationUri); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to open browser: %v", err))
		fmt.Println("Please open the URL manually in your browser.")
	}

	// Step 4: Poll for the token
	ui.PrintInfo("Waiting for authorization... (press Ctrl+C to cancel)")
	
	pollInterval := time.Duration(interval) * time.Second
	if pollInterval < 1*time.Second {
		pollInterval = 5 * time.Second
	}
	
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for time.Now().Before(deadline) {
		tokenOutput, err := c.oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     aws.String(clientId),
			ClientSecret: aws.String(clientSecret),
			GrantType:    aws.String(GrantType),
			DeviceCode:   aws.String(deviceCode),
		})
		
		if err != nil {
			// Check if it's an authorization pending error
			if strings.Contains(err.Error(), "AuthorizationPendingException") ||
			   strings.Contains(err.Error(), "authorization_pending") {
				fmt.Print(".")
				time.Sleep(pollInterval)
				continue
			}
			
			// Check if it's a slow down error
			if strings.Contains(err.Error(), "SlowDownException") ||
			   strings.Contains(err.Error(), "slow_down") {
				pollInterval = pollInterval * 2
				time.Sleep(pollInterval)
				continue
			}
			
			return fmt.Errorf("failed to get token: %w", err)
		}

		c.accessToken = aws.ToString(tokenOutput.AccessToken)
		fmt.Println()
		ui.PrintSuccess("Authorization successful!")
		return nil
	}

	return fmt.Errorf("authorization timed out")
}

// ListAccounts lists all AWS accounts available to the user
func (c *SSOClient) ListAccounts(ctx context.Context) ([]Account, error) {
	var accounts []Account
	var nextToken *string

	for {
		output, err := c.ssoClient.ListAccounts(ctx, &sso.ListAccountsInput{
			AccessToken: aws.String(c.accessToken),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, acc := range output.AccountList {
			accounts = append(accounts, Account{
				AccountId:    aws.ToString(acc.AccountId),
				AccountName:  aws.ToString(acc.AccountName),
				EmailAddress: aws.ToString(acc.EmailAddress),
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return accounts, nil
}

// ListRoles lists all roles for a specific account
func (c *SSOClient) ListRoles(ctx context.Context, accountId string) ([]Role, error) {
	var roles []Role
	var nextToken *string

	for {
		output, err := c.ssoClient.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
			AccessToken: aws.String(c.accessToken),
			AccountId:   aws.String(accountId),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}

		for _, role := range output.RoleList {
			roles = append(roles, Role{
				RoleName:  aws.ToString(role.RoleName),
				AccountId: aws.ToString(role.AccountId),
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return roles, nil
}

// GetRoleCredentials gets credentials for a specific role
func (c *SSOClient) GetRoleCredentials(ctx context.Context, accountId, roleName string) (*Credentials, error) {
	output, err := c.ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(c.accessToken),
		AccountId:   aws.String(accountId),
		RoleName:    aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	creds := output.RoleCredentials
	return &Credentials{
		AccessKeyId:     aws.ToString(creds.AccessKeyId),
		SecretAccessKey: aws.ToString(creds.SecretAccessKey),
		SessionToken:    aws.ToString(creds.SessionToken),
		Expiration:      time.UnixMilli(creds.Expiration),
	}, nil
}

// SelectAccount prompts the user to select an account
func SelectAccount(accounts []Account) (*Account, error) {
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	if len(accounts) == 1 {
		fmt.Printf("%sUsing account: %s (%s)%s\n", ui.ColorCyan, accounts[0].AccountName, accounts[0].AccountId, ui.ColorReset)
		return &accounts[0], nil
	}

	// Convert to a format suitable for selection
	items := make([]string, len(accounts))
	for i, acc := range accounts {
		items[i] = fmt.Sprintf("%s (%s)", acc.AccountName, acc.AccountId)
	}

	idx, err := ui.SelectFromList("Select an AWS account:", items)
	if err != nil {
		return nil, err
	}

	return &accounts[idx], nil
}

// SelectRole prompts the user to select a role
func SelectRole(roles []Role) (*Role, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles available")
	}

	if len(roles) == 1 {
		fmt.Printf("%sUsing role: %s%s\n", ui.ColorCyan, roles[0].RoleName, ui.ColorReset)
		return &roles[0], nil
	}

	items := make([]string, len(roles))
	for i, role := range roles {
		items[i] = role.RoleName
	}

	idx, err := ui.SelectFromList("Select a role:", items)
	if err != nil {
		return nil, err
	}

	return &roles[idx], nil
}

// ValidateSSOUrl validates if the provided URL is a valid AWS SSO start URL
func ValidateSSOUrl(ssoUrl string) error {
	parsed, err := url.Parse(ssoUrl)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("SSO URL must use HTTPS")
	}

	if parsed.Host == "" {
		return fmt.Errorf("SSO URL must have a valid host")
	}

	return nil
}

// ExtractRegionFromURL tries to extract the region from an AWS SSO URL
func ExtractRegionFromURL(ssoUrl string) string {
	// Default region
	defaultRegion := "us-east-1"

	parsed, err := url.Parse(ssoUrl)
	if err != nil {
		return defaultRegion
	}

	host := parsed.Host
	
	// Try to extract region from URL patterns like:
	// https://d-xxxxxxxxxx.awsapps.com/start
	// or regional URLs
	
	if strings.Contains(host, ".awsapps.com") {
		// For standard SSO URLs, we need to check if there's a regional pattern
		// The default SSO region can be specified, but often it's us-east-1
		return defaultRegion
	}

	// Check for regional patterns in the URL
	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
	}

	for _, region := range regions {
		if strings.Contains(host, region) {
			return region
		}
	}

	return defaultRegion
}

// ExportCredentialsScript generates a shell script for exporting credentials
func ExportCredentialsScript(creds *Credentials) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("export AWS_ACCESS_KEY_ID=\"%s\"\n", creds.AccessKeyId))
	sb.WriteString(fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=\"%s\"\n", creds.SecretAccessKey))
	sb.WriteString(fmt.Sprintf("export AWS_SESSION_TOKEN=\"%s\"\n", creds.SessionToken))

	return sb.String()
}

// WriteCredentialsToFile writes credentials to a temporary file for sourcing
func WriteCredentialsToFile(creds *Credentials) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/.aws-terminal/credentials.sh", homeDir)

	content := ExportCredentialsScript(creds)

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

// ConvertAccountsToTypes converts internal Account type to AWS SDK types
func ConvertAccountsToTypes(accounts []Account) []types.AccountInfo {
	result := make([]types.AccountInfo, len(accounts))
	for i, acc := range accounts {
		result[i] = types.AccountInfo{
			AccountId:    aws.String(acc.AccountId),
			AccountName:  aws.String(acc.AccountName),
			EmailAddress: aws.String(acc.EmailAddress),
		}
	}
	return result
}
