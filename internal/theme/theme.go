package theme

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

func Detect(ctx context.Context) string {
	switch runtime.GOOS {
	case "windows":
		return windowsSystemTheme(ctx)
	case "darwin":
		return darwinSystemTheme(ctx)
	default:
		return linuxSystemTheme(ctx)
	}
}

func windowsSystemTheme(ctx context.Context) string {
	out, ok := runThemeCommand(ctx, "reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "/v", "AppsUseLightTheme")
	if !ok {
		return ""
	}
	fields := strings.Fields(strings.ToLower(out))
	if len(fields) == 0 {
		return ""
	}
	switch fields[len(fields)-1] {
	case "0x0", "0":
		return ThemeDark
	case "0x1", "1":
		return ThemeLight
	default:
		return ""
	}
}

func darwinSystemTheme(ctx context.Context) string {
	out, ok := runThemeCommand(ctx, "defaults", "read", "-g", "AppleInterfaceStyle")
	if ok && strings.Contains(strings.ToLower(out), ThemeDark) {
		return ThemeDark
	}
	return ThemeLight
}

func linuxSystemTheme(ctx context.Context) string {
	for _, env := range []string{"GTK_THEME", "QT_STYLE_OVERRIDE"} {
		if theme := themeFromName(os.Getenv(env)); theme != "" {
			return theme
		}
	}

	if out, ok := runThemeCommand(ctx, "gsettings", "get", "org.gnome.desktop.interface", "color-scheme"); ok {
		lower := strings.ToLower(out)
		if strings.Contains(lower, "prefer-dark") {
			return ThemeDark
		}
		if strings.Contains(lower, "prefer-light") {
			return ThemeLight
		}
	}

	if out, ok := runThemeCommand(ctx, "gsettings", "get", "org.gnome.desktop.interface", "gtk-theme"); ok {
		if theme := themeFromName(out); theme != "" {
			return theme
		}
	}

	for _, cmd := range []string{"kreadconfig6", "kreadconfig5"} {
		if out, ok := runThemeCommand(ctx, cmd, "--group", "General", "--key", "ColorScheme"); ok {
			if theme := themeFromName(out); theme != "" {
				return theme
			}
		}
	}

	if theme := themeFromKDEGlobals(); theme != "" {
		return theme
	}

	return ""
}

func runThemeCommand(ctx context.Context, name string, args ...string) (string, bool) {
	if _, err := exec.LookPath(name); err != nil {
		return "", false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	commandCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(commandCtx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", false
	}
	return strings.TrimSpace(out.String()), true
}

func themeFromName(name string) string {
	lower := strings.ToLower(strings.Trim(name, "'\" \n\t"))
	if lower == "" {
		return ""
	}
	if strings.Contains(lower, ThemeDark) || strings.Contains(lower, "black") {
		return ThemeDark
	}
	if strings.Contains(lower, ThemeLight) {
		return ThemeLight
	}
	return ""
}

func themeFromKDEGlobals() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "kdeglobals"))
	if err != nil {
		return ""
	}

	inGeneral := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inGeneral = strings.EqualFold(line, "[General]")
			continue
		}
		if !inGeneral || !strings.HasPrefix(line, "ColorScheme=") {
			continue
		}
		return themeFromName(strings.TrimPrefix(line, "ColorScheme="))
	}

	return ""
}
