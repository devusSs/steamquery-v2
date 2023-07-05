package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/types"
	"github.com/devusSs/steamquery-v2/utils"
)

const (
	updateURL = "https://api.github.com/repos/devusSs/steamquery-v2/releases/latest"
)

var (
	BuildDate      string
	BuildMode      string
	BuildVersion   string
	buildArch      = runtime.GOARCH
	buildOS        = runtime.GOOS
	buildGoVersion = runtime.Version()
)

func PrintBuildInfo() {
	fmt.Printf("Build date: \t  %s\n", BuildDate)
	fmt.Printf("Build mode: \t  %s\n", BuildMode)
	fmt.Printf("Build version: \t  %s\n", BuildVersion)
	fmt.Printf("Build OS: \t  %s\n", buildOS)
	fmt.Printf("Build arch: \t  %s\n", buildArch)
	fmt.Printf("Build GO version: %s\n", buildGoVersion)
}

func CheckForUpdatesAndApply() error {
	updateURL, newVersion, changelog, err := findLatestReleaseURL()
	if err != nil {
		return err
	}

	newVersionAvailable, err := newerVersionAvailable(newVersion)
	if err != nil {
		return err
	}

	if newVersionAvailable {
		if err := doUpdate(updateURL); err != nil {
			return err
		}

		fmt.Printf("Update changelog (%s): %s\n", newVersion, changelog)

		fmt.Println("Update succeeded, please restart the app")

		os.Exit(0)
	}

	return nil
}

// Minimum version increased to v1.0.8 due to major fixes.
func CheckMinVersion() error {
	currentVersion, err := semver.NewVersion(BuildVersion)
	if err != nil {
		return err
	}

	minVersion, err := semver.NewVersion("v1.2.2")
	if err != nil {
		return err
	}

	if currentVersion.LessThan(minVersion) {
		return fmt.Errorf(
			"unsupported version (%s), please update to at least (%s)",
			BuildVersion,
			"v1.2.2",
		)
	}

	return nil
}

// Function for watchdog mode to notify user for potentially new version.
func PeriodicUpdateCheck(stopCheck chan bool) {
	logging.LogInfo("Setup periodic update checking, every 6 hours")

	interval := time.NewTicker(6 * time.Hour)

	for {
		select {
		case <-interval.C:
			_, newVersion, changelog, err := findLatestReleaseURL()
			if err != nil {
				logging.LogFatal(err.Error())
			}

			newVersionAvailable, err := newerVersionAvailable(newVersion)
			if err != nil {
				logging.LogFatal(err.Error())
			}

			if newVersionAvailable {
				mailData := utils.EmailData{}
				mailData.Subject = "steamquery-v2 new version available"
				mailData.Data = fmt.Sprintf(
					"A new version of steamquery-v2 is available.<br>Version: %s<br>Changelog: %s<br>Timestamp: %s",
					newVersion,
					changelog,
					time.Now().Local().String(),
				)
				if err := utils.SendMail(&mailData); err != nil {
					logging.LogFatal(err.Error())
				}
			}
		case <-stopCheck:
			interval.Stop()
			logging.LogDebug("Stopping periodic update check")
			return
		}
	}
}

// Queries the latest release from Github repo.
func findLatestReleaseURL() (string, string, string, error) {
	resp, err := http.Get(updateURL)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var release types.GithubRelease

	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return "", "", "", err
	}

	// Fix versions / architecture to match Github releases.
	if buildArch == "amd64" {
		buildArch = "x86_64"
	}

	if buildArch == "386" {
		buildArch = "i386"
	}

	// Find matching release for our OS & architecture.
	for _, asset := range release.Assets {
		releaseName := strings.ToLower(asset.Name)

		if strings.Contains(releaseName, buildArch) && strings.Contains(releaseName, buildOS) {
			// Format the changelog body accordingly.
			changeSplit := strings.Split(
				strings.ReplaceAll(strings.TrimSpace(release.Body), "## Changelog", ""),
				"\n",
			)
			for i, line := range changeSplit {
				changeSplit[i] = strings.ReplaceAll(fmt.Sprintf("\t\t\t%s", line), "*", "-")
			}
			changelog := strings.Join(changeSplit, "\n")
			return asset.BrowserDownloadURL, release.TagName, changelog, nil
		}
	}

	return "", "", "", errors.New("no matching release found")
}

// Compare current version with latest version
func newerVersionAvailable(newVersion string) (bool, error) {
	currentBuild := strings.ReplaceAll(BuildVersion, "v", "")
	newBuild := strings.ReplaceAll(newVersion, "v", "")

	vOld, err := semver.NewVersion(currentBuild)
	if err != nil {
		return false, err
	}

	vNew, err := semver.NewVersion(newBuild)
	if err != nil {
		return false, err
	}

	return vOld.LessThan(vNew), nil
}

// Perform the actual patch.
func doUpdate(url string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if err := selfupdate.UpdateTo(url, exe); err != nil {
		return err
	}

	return nil
}
