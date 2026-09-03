package webapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/page"
)

const chromePath = "/Applications/Google Chrome.app"

func Install(ctx context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("os.UserHomeDir: %w", err)
	}

	bundle := filepath.Join(home, "Applications", appName+".app")
	if go_pkg_filesystem_reader.Exists(bundle) {
		return bundle, nil
	}

	if !go_pkg_filesystem_reader.Exists(chromePath) {
		return "", fmt.Errorf("%s not installed", chromePath)
	}

	icns, err := page.FS.ReadFile("public/icon.icns")
	if err != nil {
		return "", fmt.Errorf("page.FS.ReadFile: %w", err)
	}

	binDir := filepath.Join(bundle, "Contents", "MacOS")
	resDir := filepath.Join(bundle, "Contents", "Resources")
	for _, dir := range []string{binDir, resDir} {
		if err := go_pkg_filesystem.CheckDir(dir, true); err != nil {
			return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: CheckDir: %w", err)
		}
	}

	if err := go_pkg_filesystem.WriteFile(filepath.Join(resDir, appName+".icns"), string(icns), 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}
	if err := go_pkg_filesystem.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), infoPlist(), 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}
	if err := go_pkg_filesystem.WriteFile(filepath.Join(binDir, appName), launcher(), 0755); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}

	now := time.Now()
	if err := os.Chtimes(bundle, now, now); err != nil {
		return "", fmt.Errorf("os.Chtimes: %w", err)
	}
	return bundle, nil
}

func launcher() string {
	return "#!/bin/sh\nexec /usr/bin/open -na \"Google Chrome\" --args --app=" + appURL() + "\n"
}

func infoPlist() string {
	version := appVersion()
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>` + appName + `</string>
	<key>CFBundleDisplayName</key>
	<string>` + appName + `</string>
	<key>CFBundleIdentifier</key>
	<string>com.pardnchiu.agenvoy.webapp</string>
	<key>CFBundleExecutable</key>
	<string>` + appName + `</string>
	<key>CFBundleIconFile</key>
	<string>` + appName + `</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>` + version + `</string>
	<key>CFBundleVersion</key>
	<string>` + version + `</string>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
`
}
