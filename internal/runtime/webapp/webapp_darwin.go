package webapp

import (
	"context"
	"os"
	"path/filepath"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/configs"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
	"github.com/pardnchiu/agenvoy/page"
)

const chromePath = "/Applications/Google Chrome.app"

func Install(ctx context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.UserHomeDir", "", err)
	}

	bundle := filepath.Join(home, "Applications", configs.APP_NAME+".app")
	if go_pkg_filesystem_reader.Exists(bundle) {
		return bundle, nil
	}

	if !go_pkg_filesystem_reader.Exists(chromePath) {
		return "", utilsmsg.New(utilsmsg.NotFound, chromePath, "not installed")
	}

	icns, err := page.FS.ReadFile("public/icon.icns")
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "page.FS.ReadFile", "", err)
	}

	binDir := filepath.Join(bundle, "Contents", "MacOS")
	resDir := filepath.Join(bundle, "Contents", "Resources")
	for _, dir := range []string{binDir, resDir} {
		if err := go_pkg_filesystem.CheckDir(dir, true); err != nil {
			return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.CheckDir", "", err)
		}
	}

	if err := go_pkg_filesystem.WriteFile(filepath.Join(resDir, configs.APP_NAME+".icns"), string(icns), 0644); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", "", err)
	}
	if err := go_pkg_filesystem.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), infoPlist(), 0644); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", "", err)
	}
	if err := go_pkg_filesystem.WriteFile(filepath.Join(binDir, configs.APP_NAME), launcher(), 0755); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", "", err)
	}

	now := time.Now()
	if err := os.Chtimes(bundle, now, now); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.Chtimes", "", err)
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
	<string>` + configs.APP_NAME + `</string>
	<key>CFBundleDisplayName</key>
	<string>` + configs.APP_NAME + `</string>
	<key>CFBundleIdentifier</key>
	<string>com.pardnchiu.agenvoy.webapp</string>
	<key>CFBundleExecutable</key>
	<string>` + configs.APP_NAME + `</string>
	<key>CFBundleIconFile</key>
	<string>` + configs.APP_NAME + `</string>
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
