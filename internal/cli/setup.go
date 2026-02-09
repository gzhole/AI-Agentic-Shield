package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gzhole/agentshield/internal/config"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up AgentShield for your environment",
	Long: `Set up AgentShield integration with IDE agents and install default policy packs.

Run without arguments to see integration instructions:
  agentshield setup

Install the wrapper script and default packs automatically:
  agentshield setup --install`,
	RunE: setupCommand,
}

var installFlag bool

func init() {
	setupCmd.Flags().BoolVar(&installFlag, "install", false, "Install wrapper script and default policy packs")
	rootCmd.AddCommand(setupCmd)
}

func setupCommand(cmd *cobra.Command, args []string) error {
	if installFlag {
		return runSetupInstall()
	}
	printSetupInstructions()
	return nil
}

func runSetupInstall() error {
	cfg, err := config.Load("", "", "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Ensure packs directory exists
	packsDir := filepath.Join(cfg.ConfigDir, "packs")
	if err := os.MkdirAll(packsDir, 0755); err != nil {
		return fmt.Errorf("failed to create packs dir: %w", err)
	}

	// Find wrapper script source
	wrapperSrc := findWrapperSource()
	if wrapperSrc == "" {
		fmt.Fprintln(os.Stderr, "⚠  Could not find agentshield-wrapper.sh source.")
		fmt.Fprintln(os.Stderr, "   The wrapper will need to be installed manually.")
	} else {
		// Install wrapper to share directory
		shareDir := getShareDir()
		if err := os.MkdirAll(shareDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  Could not create %s: %v\n", shareDir, err)
			fmt.Fprintln(os.Stderr, "   Try: sudo agentshield setup --install")
		} else {
			wrapperDst := filepath.Join(shareDir, "agentshield-wrapper.sh")
			data, err := os.ReadFile(wrapperSrc)
			if err != nil {
				return fmt.Errorf("failed to read wrapper source: %w", err)
			}
			if err := os.WriteFile(wrapperDst, data, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "⚠  Could not write to %s: %v\n", wrapperDst, err)
				fmt.Fprintln(os.Stderr, "   Try: sudo agentshield setup --install")
			} else {
				fmt.Printf("✅ Wrapper installed: %s\n", wrapperDst)
			}
		}
	}

	// Copy bundled packs if packs dir is empty
	packsSrc := findPacksSource()
	if packsSrc != "" {
		installed := installPacks(packsSrc, packsDir)
		if installed > 0 {
			fmt.Printf("✅ %d policy packs installed to %s\n", installed, packsDir)
		} else {
			fmt.Printf("✅ Policy packs already present in %s\n", packsDir)
		}
	}

	fmt.Println()
	printSetupInstructions()
	return nil
}

func printSetupInstructions() {
	wrapperPath := filepath.Join(getShareDir(), "agentshield-wrapper.sh")

	// Check if wrapper exists
	wrapperExists := false
	if _, err := os.Stat(wrapperPath); err == nil {
		wrapperExists = true
	}

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("  AgentShield Setup Guide")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	if !wrapperExists {
		fmt.Println("⚠  Wrapper not installed. Run: agentshield setup --install")
		fmt.Println()
	}

	fmt.Println("─── IDE Agent Integration (Windsurf, Cursor, etc.) ───")
	fmt.Println()
	fmt.Println("Configure your IDE to use AgentShield as the agent's shell:")
	fmt.Println()
	fmt.Printf("  Shell path: %s\n", wrapperPath)
	fmt.Println("  Shell args: -c")
	fmt.Println()
	fmt.Println("The wrapper intercepts every command the agent runs,")
	fmt.Println("evaluates it against your policy, and blocks dangerous actions.")
	fmt.Println()

	fmt.Println("─── Claude Code ───────────────────────────────────────")
	fmt.Println()
	fmt.Println("In your Claude Code settings, set the shell override:")
	fmt.Println()
	fmt.Printf("  \"shell\": \"%s\"\n", wrapperPath)
	fmt.Println()

	fmt.Println("─── Direct CLI Usage ─────────────────────────────────")
	fmt.Println()
	fmt.Println("  agentshield run -- <command>         # evaluate & run")
	fmt.Println("  agentshield run --shell -- \"cmd\"     # shell string mode")
	fmt.Println("  agentshield log                      # view audit trail")
	fmt.Println("  agentshield log --summary            # audit summary")
	fmt.Println("  agentshield pack list                # show policy packs")
	fmt.Println()

	fmt.Println("─── Bypass Mode ──────────────────────────────────────")
	fmt.Println()
	fmt.Println("  export AGENTSHIELD_BYPASS=1    # temporarily disable")
	fmt.Println("  unset AGENTSHIELD_BYPASS       # re-enable")
	fmt.Println()

	// Show current status
	fmt.Println("─── Current Status ───────────────────────────────────")
	fmt.Println()
	printStatus()
}

func printStatus() {
	cfg, _ := config.Load("", "", "")

	// Check binary
	binPath, err := exec.LookPath("agentshield")
	if err != nil {
		fmt.Println("  Binary:  ⚠  not found in PATH")
	} else {
		fmt.Printf("  Binary:  ✅ %s\n", binPath)
	}

	// Check wrapper
	wrapperPath := filepath.Join(getShareDir(), "agentshield-wrapper.sh")
	if _, err := os.Stat(wrapperPath); err == nil {
		fmt.Printf("  Wrapper: ✅ %s\n", wrapperPath)
	} else {
		fmt.Println("  Wrapper: ⚠  not installed")
	}

	// Check packs
	if cfg != nil {
		packsDir := filepath.Join(cfg.ConfigDir, "packs")
		entries, err := os.ReadDir(packsDir)
		if err == nil {
			count := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") && !strings.HasPrefix(e.Name(), "_") {
					count++
				}
			}
			fmt.Printf("  Packs:   ✅ %d enabled in %s\n", count, packsDir)
		} else {
			fmt.Printf("  Packs:   ⚠  %s not found\n", packsDir)
		}

		// Check policy
		if _, err := os.Stat(cfg.PolicyPath); err == nil {
			fmt.Printf("  Policy:  ✅ %s\n", cfg.PolicyPath)
		} else {
			fmt.Println("  Policy:  ℹ  using built-in defaults")
		}
	}

	fmt.Println()
}

// getShareDir returns the platform-appropriate share directory for AgentShield.
func getShareDir() string {
	// Check if installed via Homebrew
	brewPrefix := os.Getenv("HOMEBREW_PREFIX")
	if brewPrefix == "" {
		if runtime.GOARCH == "arm64" && runtime.GOOS == "darwin" {
			brewPrefix = "/opt/homebrew"
		} else {
			brewPrefix = "/usr/local"
		}
	}
	return filepath.Join(brewPrefix, "share", "agentshield")
}

// findWrapperSource looks for the wrapper script in known locations.
func findWrapperSource() string {
	candidates := []string{
		filepath.Join(getShareDir(), "agentshield-wrapper.sh"),
	}

	// Check relative to binary
	if binPath, err := exec.LookPath("agentshield"); err == nil {
		binDir := filepath.Dir(binPath)
		candidates = append(candidates,
			filepath.Join(binDir, "..", "share", "agentshield", "agentshield-wrapper.sh"),
			filepath.Join(binDir, "..", "scripts", "agentshield-wrapper.sh"),
		)
	}

	// Check current working directory (for dev)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "scripts", "agentshield-wrapper.sh"))
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// findPacksSource looks for bundled policy packs in known locations.
func findPacksSource() string {
	candidates := []string{
		filepath.Join(getShareDir(), "packs"),
	}

	if binPath, err := exec.LookPath("agentshield"); err == nil {
		binDir := filepath.Dir(binPath)
		candidates = append(candidates,
			filepath.Join(binDir, "..", "share", "agentshield", "packs"),
			filepath.Join(binDir, "..", "packs"),
		)
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "packs"))
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}

// installPacks copies pack files from src to dst, skipping files that already exist.
func installPacks(srcDir, dstDir string) int {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0
	}

	installed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		dstPath := filepath.Join(dstDir, e.Name())
		if _, err := os.Stat(dstPath); err == nil {
			continue // already exists
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			continue
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  Failed to install pack %s: %v\n", e.Name(), err)
			continue
		}
		installed++
	}
	return installed
}
