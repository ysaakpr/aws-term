#!/bin/bash
#
# AWS-Term Installer
# 
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ysaakpr/aws-term/main/install.sh | bash
#
# Or download and run:
#   chmod +x install.sh && ./install.sh
#

set -e

# Configuration
REPO="ysaakpr/aws-term"
BINARY_NAME="aws-term"
INSTALL_DIR="${AWS_TERM_INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

print_banner() {
    echo -e "${CYAN}${BOLD}"
    echo "╔══════════════════════════════════════════╗"
    echo "║        AWS-Term Installer                ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Darwin) echo "darwin" ;;
        Linux) echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $os" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        # Fallback: try to get any release
        version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" 2>/dev/null | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    fi
    
    echo "$version"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Get current installed version
get_installed_version() {
    if command_exists "$BINARY_NAME"; then
        "$BINARY_NAME" --version 2>/dev/null | awk '{print $NF}' || echo ""
    else
        echo ""
    fi
}

# Download and install
install() {
    local os="$1"
    local arch="$2"
    local version="$3"
    
    # Construct download URL
    local filename="${BINARY_NAME}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        filename="${filename}.zip"
    else
        filename="${filename}.tar.gz"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    
    info "Downloading ${BINARY_NAME} ${version} for ${os}/${arch}..."
    
    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    # Download
    if command_exists curl; then
        curl -fsSL "$download_url" -o "${tmp_dir}/${filename}" || error "Failed to download from ${download_url}"
    elif command_exists wget; then
        wget -q "$download_url" -O "${tmp_dir}/${filename}" || error "Failed to download from ${download_url}"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
    
    success "Downloaded successfully"
    
    # Extract
    info "Extracting..."
    cd "$tmp_dir"
    
    if [ "$os" = "windows" ]; then
        unzip -q "$filename" || error "Failed to extract archive"
    else
        tar -xzf "$filename" || error "Failed to extract archive"
    fi
    
    # Find the binary
    local binary_file="${BINARY_NAME}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        binary_file="${binary_file}.exe"
    fi
    
    if [ ! -f "$binary_file" ]; then
        error "Binary not found in archive"
    fi
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Install binary
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    if [ "$os" = "windows" ]; then
        install_path="${install_path}.exe"
    fi
    
    mv "$binary_file" "$install_path"
    chmod +x "$install_path"
    
    success "Installed to ${install_path}"
}

# Add to PATH instructions
print_path_instructions() {
    local shell_name
    shell_name=$(basename "$SHELL")
    
    # Check if already in PATH
    if echo "$PATH" | grep -q "$INSTALL_DIR"; then
        return
    fi
    
    echo ""
    warn "Add ${INSTALL_DIR} to your PATH"
    echo ""
    
    case "$shell_name" in
        zsh)
            echo -e "  Add this line to your ${CYAN}~/.zshrc${NC}:"
            echo ""
            echo -e "    ${YELLOW}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
            echo ""
            echo "  Then run:"
            echo ""
            echo -e "    ${YELLOW}source ~/.zshrc${NC}"
            ;;
        bash)
            echo -e "  Add this line to your ${CYAN}~/.bashrc${NC} or ${CYAN}~/.bash_profile${NC}:"
            echo ""
            echo -e "    ${YELLOW}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
            echo ""
            echo "  Then run:"
            echo ""
            echo -e "    ${YELLOW}source ~/.bashrc${NC}"
            ;;
        *)
            echo -e "  Add ${CYAN}${INSTALL_DIR}${NC} to your shell's PATH configuration."
            ;;
    esac
}

# Verify installation
verify_installation() {
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    
    if [ -f "$install_path" ]; then
        success "Installation verified!"
        echo ""
        info "Version: $("$install_path" --version 2>/dev/null || echo 'unknown')"
        return 0
    else
        error "Installation verification failed"
        return 1
    fi
}

# Main
main() {
    print_banner
    
    # Detect system
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    
    info "Detected: ${os}/${arch}"
    
    # Check current version
    local current_version
    current_version=$(get_installed_version)
    
    if [ -n "$current_version" ]; then
        info "Current version: ${current_version}"
    fi
    
    # Get latest version
    info "Checking for latest version..."
    local latest_version
    latest_version=$(get_latest_version)
    
    if [ -z "$latest_version" ]; then
        error "Could not determine latest version. Please check https://github.com/${REPO}/releases"
    fi
    
    info "Latest version: ${latest_version}"
    
    # Check if update needed
    if [ "$current_version" = "$latest_version" ] || [ "$current_version" = "${latest_version#v}" ]; then
        success "Already up to date!"
        echo ""
        info "Run '${BINARY_NAME}' to get started"
        exit 0
    fi
    
    # Install
    echo ""
    install "$os" "$arch" "$latest_version"
    
    # Verify
    echo ""
    verify_installation
    
    # PATH instructions
    print_path_instructions
    
    echo ""
    success "Installation complete!"
    echo ""
    info "Run '${BINARY_NAME} --help' to get started"
    echo ""
}

main "$@"

