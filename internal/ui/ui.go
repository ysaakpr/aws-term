package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ysaakpr/aws-term/internal/config"
	"golang.org/x/term"
)

const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ClearLine   = "\033[2K"
	MoveUp      = "\033[1A"
	HideCursor  = "\033[?25l"
	ShowCursor  = "\033[?25h"
)

// PrintHeader prints the application header
func PrintHeader() {
	fmt.Println()
	fmt.Printf("%s%s╔══════════════════════════════════════════╗%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s║          AWS Terminal Session            ║%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════╝%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Println()
}

// PromptInput prompts the user for text input
func PromptInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s%s%s: ", ColorYellow, prompt, ColorReset)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// PromptSSOUrl prompts the user for an AWS SSO URL
func PromptSSOUrl() string {
	fmt.Printf("\n%sNo AWS SSO configuration found.%s\n", ColorYellow, ColorReset)
	fmt.Println("Please enter your AWS SSO start URL:")
	fmt.Println("(e.g., https://my-company.awsapps.com/start)")
	fmt.Println()
	return PromptInput("SSO URL")
}

// PromptProfileName prompts the user for a profile name
func PromptProfileName() string {
	return PromptInput("Profile name")
}

// SelectProfile displays a list of profiles and allows the user to select one
func SelectProfile(profiles []config.Profile) (*config.Profile, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles available")
	}

	if len(profiles) == 1 {
		return &profiles[0], nil
	}

	fmt.Printf("\n%s%sSelect a profile:%s\n\n", ColorBold, ColorCyan, ColorReset)

	selectedIndex := 0

	// Find default profile index
	for i, p := range profiles {
		if p.Default {
			selectedIndex = i
			break
		}
	}

	// Try to enable raw mode for arrow key navigation
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return selectProfileFallback(profiles)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return selectProfileFallback(profiles)
	}
	defer term.Restore(fd, oldState)

	// Hide cursor during selection
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	for {
		// Print profiles
		for i, p := range profiles {
			defaultMarker := ""
			if p.Default {
				defaultMarker = fmt.Sprintf(" %s(default)%s", ColorGreen, ColorReset)
			}

			if i == selectedIndex {
				fmt.Printf("%s  ▸ %s%s%s\r\n", ClearLine, ColorBold, p.Name, ColorReset)
				fmt.Printf("%s    %s%s%s%s\r\n", ClearLine, ColorBlue, p.SSOUrl, ColorReset, defaultMarker)
			} else {
				fmt.Printf("%s    %s\r\n", ClearLine, p.Name)
				fmt.Printf("%s    %s%s%s%s\r\n", ClearLine, ColorBlue, p.SSOUrl, ColorReset, defaultMarker)
			}
		}

		fmt.Printf("\r\n%sUse ↑/↓ arrows to navigate, Enter to select, q to quit%s\r\n", ColorYellow, ColorReset)

		// Read key
		var buf [3]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return nil, err
		}

		// Move cursor up to redraw
		for i := 0; i <= len(profiles)*2+1; i++ {
			fmt.Print(MoveUp + ClearLine)
		}

		if n == 1 {
			switch buf[0] {
			case 'q', 'Q', 3: // q, Q, or Ctrl+C
				fmt.Print(ShowCursor)
				term.Restore(fd, oldState)
				fmt.Println("\r\nCancelled.")
				os.Exit(0)
			case 13, 10: // Enter (CR or LF)
				fmt.Print(ShowCursor)
				fmt.Printf("\r\n")
				return &profiles[selectedIndex], nil
			case 'j': // vim down
				if selectedIndex < len(profiles)-1 {
					selectedIndex++
				}
			case 'k': // vim up
				if selectedIndex > 0 {
					selectedIndex--
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if selectedIndex > 0 {
					selectedIndex--
				}
			case 66: // Down arrow
				if selectedIndex < len(profiles)-1 {
					selectedIndex++
				}
			}
		}
	}
}

// selectProfileFallback is a fallback for when raw mode is not available
func selectProfileFallback(profiles []config.Profile) (*config.Profile, error) {
	fmt.Printf("\n%s%sAvailable profiles:%s\n\n", ColorBold, ColorCyan, ColorReset)

	for i, p := range profiles {
		defaultMarker := ""
		if p.Default {
			defaultMarker = fmt.Sprintf(" %s(default)%s", ColorGreen, ColorReset)
		}
		fmt.Printf("  %d. %s%s\n", i+1, p.Name, defaultMarker)
		fmt.Printf("     %s%s%s\n", ColorBlue, p.SSOUrl, ColorReset)
	}

	fmt.Println()
	input := PromptInput("Enter profile number")

	var index int
	_, err := fmt.Sscanf(input, "%d", &index)
	if err != nil || index < 1 || index > len(profiles) {
		return nil, fmt.Errorf("invalid selection")
	}

	return &profiles[index-1], nil
}

// SelectBrowser prompts the user to select a browser
func SelectBrowser(browsers []string) (string, error) {
	if len(browsers) == 0 {
		return "", fmt.Errorf("no browsers found")
	}

	if len(browsers) == 1 {
		fmt.Printf("%sUsing %s...%s\n", ColorCyan, browsers[0], ColorReset)
		return browsers[0], nil
	}

	fmt.Printf("\n%s%sSelect a browser:%s\n\n", ColorBold, ColorCyan, ColorReset)

	selectedIndex := 0

	// Try to enable raw mode for arrow key navigation
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return selectBrowserFallback(browsers)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return selectBrowserFallback(browsers)
	}
	defer term.Restore(fd, oldState)

	// Hide cursor during selection
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	for {
		// Print browsers
		for i, b := range browsers {
			if i == selectedIndex {
				fmt.Printf("%s  ▸ %s%s%s\r\n", ClearLine, ColorBold, b, ColorReset)
			} else {
				fmt.Printf("%s    %s\r\n", ClearLine, b)
			}
		}

		fmt.Printf("\r\n%sUse ↑/↓ arrows to navigate, Enter to select%s\r\n", ColorYellow, ColorReset)

		// Read key
		var buf [3]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return "", err
		}

		// Move cursor up to redraw
		for i := 0; i <= len(browsers)+1; i++ {
			fmt.Print(MoveUp + ClearLine)
		}

		if n == 1 {
			switch buf[0] {
			case 'q', 'Q', 3: // q, Q, or Ctrl+C
				fmt.Print(ShowCursor)
				term.Restore(fd, oldState)
				fmt.Println("\r\nCancelled.")
				os.Exit(0)
			case 13, 10: // Enter (CR or LF)
				fmt.Print(ShowCursor)
				fmt.Printf("\r\n")
				return browsers[selectedIndex], nil
			case 'j': // vim down
				if selectedIndex < len(browsers)-1 {
					selectedIndex++
				}
			case 'k': // vim up
				if selectedIndex > 0 {
					selectedIndex--
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if selectedIndex > 0 {
					selectedIndex--
				}
			case 66: // Down arrow
				if selectedIndex < len(browsers)-1 {
					selectedIndex++
				}
			}
		}
	}
}

// selectBrowserFallback is a fallback for when raw mode is not available
func selectBrowserFallback(browsers []string) (string, error) {
	for i, b := range browsers {
		fmt.Printf("  %d. %s\n", i+1, b)
	}

	fmt.Println()
	input := PromptInput("Enter browser number")

	var index int
	_, err := fmt.Sscanf(input, "%d", &index)
	if err != nil || index < 1 || index > len(browsers) {
		return "", fmt.Errorf("invalid selection")
	}

	return browsers[index-1], nil
}

// ConfirmSetDefault asks user if they want to set this profile as default
func ConfirmSetDefault() bool {
	input := PromptInput("Set as default profile? (y/N)")
	return strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("\n%s✓ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Printf("\n%s✗ %s%s\n", "\033[31m", message, ColorReset)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("%s%s%s\n", ColorCyan, message, ColorReset)
}

// PrintCredentials prints the export commands for the user
func PrintCredentials(accessKeyId, secretAccessKey, sessionToken string) {
	fmt.Println()
	fmt.Printf("%s%sAWS Credentials obtained successfully!%s\n", ColorBold, ColorGreen, ColorReset)
	fmt.Println()
	fmt.Printf("%sCopy and paste these commands to set your environment:%s\n\n", ColorYellow, ColorReset)
	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", accessKeyId)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", secretAccessKey)
	fmt.Printf("export AWS_SESSION_TOKEN=%s\n", sessionToken)
	fmt.Println()
}

// SelectFromList displays a generic list and allows the user to select one item
func SelectFromList(title string, items []string) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items available")
	}

	if len(items) == 1 {
		return 0, nil
	}

	fmt.Printf("\n%s%s%s%s\n\n", ColorBold, ColorCyan, title, ColorReset)

	selectedIndex := 0

	// Try to enable raw mode for arrow key navigation
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return selectFromListFallback(items)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return selectFromListFallback(items)
	}
	defer term.Restore(fd, oldState)

	// Hide cursor during selection
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	for {
		// Print items
		for i, item := range items {
			if i == selectedIndex {
				fmt.Printf("%s  ▸ %s%s%s\r\n", ClearLine, ColorBold, item, ColorReset)
			} else {
				fmt.Printf("%s    %s\r\n", ClearLine, item)
			}
		}

		fmt.Printf("\r\n%sUse ↑/↓ arrows to navigate, Enter to select, q to quit%s\r\n", ColorYellow, ColorReset)

		// Read key
		var buf [3]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return -1, err
		}

		// Move cursor up to redraw
		for i := 0; i <= len(items)+1; i++ {
			fmt.Print(MoveUp + ClearLine)
		}

		if n == 1 {
			switch buf[0] {
			case 'q', 'Q', 3: // q, Q, or Ctrl+C
				fmt.Print(ShowCursor)
				term.Restore(fd, oldState)
				fmt.Println("\r\nCancelled.")
				os.Exit(0)
			case 13, 10: // Enter (CR or LF)
				fmt.Print(ShowCursor)
				fmt.Printf("\r\n")
				return selectedIndex, nil
			case 'j': // vim down
				if selectedIndex < len(items)-1 {
					selectedIndex++
				}
			case 'k': // vim up
				if selectedIndex > 0 {
					selectedIndex--
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if selectedIndex > 0 {
					selectedIndex--
				}
			case 66: // Down arrow
				if selectedIndex < len(items)-1 {
					selectedIndex++
				}
			}
		}
	}
}

// selectFromListFallback is a fallback for when raw mode is not available
func selectFromListFallback(items []string) (int, error) {
	for i, item := range items {
		fmt.Printf("  %d. %s\n", i+1, item)
	}

	fmt.Println()
	input := PromptInput("Enter number")

	var index int
	_, err := fmt.Sscanf(input, "%d", &index)
	if err != nil || index < 1 || index > len(items) {
		return -1, fmt.Errorf("invalid selection")
	}

	return index - 1, nil
}
