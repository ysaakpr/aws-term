# aws-term

A CLI tool to manage AWS SSO sessions and set up terminal environment variables for AWS access. Uses the official AWS SDK for secure device authorization flow.

## Features

- 🔐 **AWS SSO Device Authorization** - Secure OAuth 2.0 device flow (like `aws sso login`)
- 📋 **Multiple Profiles** - Store and manage multiple AWS SSO configurations
- 🏢 **Account Selection** - Choose from available AWS accounts
- 👤 **Role Selection** - Pick the IAM role to assume
- ⌨️ **Arrow Key Navigation** - Use arrow keys to navigate selections
- 🖥️ **Shell Integration** - Spawn a new shell with credentials pre-loaded
- ⏰ **Session Expiry** - Shows credential expiration time

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/ysaakpr/aws-term.git
cd aws-term

# Build
make build

# Install to /usr/local/bin (optional)
make install
```

### Manual Installation

```bash
go build -o aws-term ./cmd/aws-term
mv aws-term /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Run with default profile or show profile selection
aws-term

# Use a specific profile
aws-term production

# Add a new SSO profile
aws-term --add

# List all configured profiles
aws-term --list

# Set a profile as default
aws-term --set-default production

# Use a specific AWS region
aws-term --region eu-west-1
```

### First Time Setup

When you run `aws-term` for the first time, it will prompt you to:

1. Enter your AWS SSO start URL (e.g., `https://my-company.awsapps.com/start`)
2. Give the profile a name
3. Specify the AWS region (default: us-east-1)
4. Optionally set it as the default profile

### Authentication Workflow

1. **Select Profile** - Choose an SSO profile (or use default)
2. **Select Browser** - Pick Chrome, Safari, or another detected browser
3. **Device Authorization** - Browser opens with a verification code
4. **Sign In** - Complete SSO login in your browser
5. **Select Account** - Choose from available AWS accounts
6. **Select Role** - Pick the IAM role to assume
7. **Get Credentials** - Receive temporary AWS credentials

### Using Credentials

After authentication, you have two options:

**Option 1: Source the credentials file**
```bash
source ~/.aws-terminal/credentials.sh
```

**Option 2: Start a new shell with credentials**
When prompted, select 'y' to open a new shell session with the AWS credentials already set.

**Option 3: Copy export commands**
Copy the displayed export commands and paste them in your terminal.

## Configuration

Configuration is stored in `~/.aws-terminal/config.json`:

```json
{
  "profiles": [
    {
      "name": "production",
      "sso_url": "https://my-company.awsapps.com/start",
      "region": "us-east-1",
      "default": true
    },
    {
      "name": "development",
      "sso_url": "https://my-dev.awsapps.com/start",
      "region": "eu-west-1"
    }
  ]
}
```

## Command Line Options

| Option | Description |
|--------|-------------|
| `--help` | Show help information |
| `--version` | Show version information |
| `--add` | Add a new SSO profile interactively |
| `--list` | List all configured profiles |
| `--set-default <name>` | Set a profile as the default |
| `--region <region>` | Override the AWS region |

## How It Works

```
┌─────────────────┐
│  aws-term CLI   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ Register Client │────▶│ AWS SSO OIDC     │
└────────┬────────┘     └──────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ Device Auth     │────▶│ Browser Login    │
└────────┬────────┘     └──────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ Poll for Token  │────▶│ Get Access Token │
└────────┬────────┘     └──────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ List Accounts   │────▶│ AWS SSO          │
└────────┬────────┘     └──────────────────┘
         │
         ▼
┌─────────────────┐
│ Get Role Creds  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Export to Shell │
└─────────────────┘
```

## Requirements

- macOS, Linux, or Windows
- One of: Chrome, Safari, Firefox, Brave, Edge
- Go 1.21+ (for building from source)

## Comparison with AWS CLI

| Feature | aws-term | aws sso login |
|---------|----------|---------------|
| Interactive account/role selection | ✅ | ❌ (requires pre-configuration) |
| Arrow key navigation | ✅ | ❌ |
| Multiple SSO URLs | ✅ | ✅ |
| Spawn shell with credentials | ✅ | ❌ |
| No AWS CLI required | ✅ | ❌ |

## License

MIT License
