package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Browser represents a detected browser
type Browser struct {
	Name string
	Path string
}

// DetectBrowsers finds available browsers on the system
func DetectBrowsers() []string {
	var browsers []string

	switch runtime.GOOS {
	case "darwin":
		// macOS browser detection
		chromePaths := []string{
			"/Applications/Google Chrome.app",
			"/Applications/Google Chrome Canary.app",
			"/Applications/Chromium.app",
		}
		for _, path := range chromePaths {
			if _, err := os.Stat(path); err == nil {
				browsers = append(browsers, "Chrome")
				break
			}
		}

		if _, err := os.Stat("/Applications/Safari.app"); err == nil {
			browsers = append(browsers, "Safari")
		}

		// Also check for Firefox and Brave
		if _, err := os.Stat("/Applications/Firefox.app"); err == nil {
			browsers = append(browsers, "Firefox")
		}
		if _, err := os.Stat("/Applications/Brave Browser.app"); err == nil {
			browsers = append(browsers, "Brave")
		}

	case "linux":
		// Linux browser detection
		if _, err := exec.LookPath("google-chrome"); err == nil {
			browsers = append(browsers, "Chrome")
		} else if _, err := exec.LookPath("chromium"); err == nil {
			browsers = append(browsers, "Chromium")
		} else if _, err := exec.LookPath("chromium-browser"); err == nil {
			browsers = append(browsers, "Chromium")
		}

		if _, err := exec.LookPath("firefox"); err == nil {
			browsers = append(browsers, "Firefox")
		}

	case "windows":
		// Windows browser detection
		chromePaths := []string{
			os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("PROGRAMFILES") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("PROGRAMFILES(X86)") + "\\Google\\Chrome\\Application\\chrome.exe",
		}
		for _, path := range chromePaths {
			if _, err := os.Stat(path); err == nil {
				browsers = append(browsers, "Chrome")
				break
			}
		}

		// Edge is always available on modern Windows
		browsers = append(browsers, "Edge")
	}

	return browsers
}

// OpenURL opens a URL in the specified browser
func OpenURL(browserName, url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		switch browserName {
		case "Chrome":
			cmd = exec.Command("open", "-a", "Google Chrome", url)
		case "Safari":
			cmd = exec.Command("open", "-a", "Safari", url)
		case "Firefox":
			cmd = exec.Command("open", "-a", "Firefox", url)
		case "Brave":
			cmd = exec.Command("open", "-a", "Brave Browser", url)
		default:
			cmd = exec.Command("open", url)
		}

	case "linux":
		switch browserName {
		case "Chrome":
			cmd = exec.Command("google-chrome", url)
		case "Chromium":
			if _, err := exec.LookPath("chromium"); err == nil {
				cmd = exec.Command("chromium", url)
			} else {
				cmd = exec.Command("chromium-browser", url)
			}
		case "Firefox":
			cmd = exec.Command("firefox", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}

	case "windows":
		switch browserName {
		case "Chrome":
			chromePaths := []string{
				os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\Application\\chrome.exe",
				os.Getenv("PROGRAMFILES") + "\\Google\\Chrome\\Application\\chrome.exe",
				os.Getenv("PROGRAMFILES(X86)") + "\\Google\\Chrome\\Application\\chrome.exe",
			}
			for _, path := range chromePaths {
				if _, err := os.Stat(path); err == nil {
					cmd = exec.Command(path, url)
					break
				}
			}
		case "Edge":
			cmd = exec.Command("cmd", "/c", "start", "microsoft-edge:"+url)
		default:
			cmd = exec.Command("cmd", "/c", "start", url)
		}

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if cmd == nil {
		return fmt.Errorf("could not find browser: %s", browserName)
	}

	return cmd.Start()
}

// GetBrowserAppPath returns the application path for a browser on macOS
func GetBrowserAppPath(browserName string) string {
	if runtime.GOOS != "darwin" {
		return ""
	}

	switch browserName {
	case "Chrome":
		return "/Applications/Google Chrome.app"
	case "Safari":
		return "/Applications/Safari.app"
	case "Firefox":
		return "/Applications/Firefox.app"
	case "Brave":
		return "/Applications/Brave Browser.app"
	default:
		return ""
	}
}

