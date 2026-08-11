package cmd

import (
	"aem/internal/selfupdate"
	"aem/pkg/errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var checkOnly bool
	var force bool
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the aem executable to the newest release",
		Long: "Check the aem source-control releases for a newer version, download the\n" +
			"matching release archive, verify it against the published checksums, and\n" +
			"replace the running executable in place.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tempDir, err := fs.GetTempDir()
			if err != nil {
				return err
			}
			service := selfupdate.NewService(log, tempDir)

			if checkOnly {
				return runUpdateCheck(service, targetVersion)
			}

			exePath, err := os.Executable()
			if err != nil {
				return errors.NewFileSystemError("failed to locate the aem executable", err)
			}

			result, err := service.Update(Version, targetVersion, exePath, force)
			if err != nil {
				return err
			}
			if result.AlreadyInstalled {
				fmt.Printf("aem is already up to date (%s)\n", result.To)
				return nil
			}
			fmt.Printf("Updated aem %s -> %s (%s)\n", result.From, result.To, result.Path)
			fmt.Println("Run `aem version` to confirm the new build.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for a newer release without downloading or updating")
	cmd.Flags().StringVar(&targetVersion, "version", "", "update to a specific release instead of the latest (e.g. 1.2.3)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "reinstall the same version, downgrade, or replace a dev build")
	return cmd
}

func runUpdateCheck(service *selfupdate.Service, targetVersion string) error {
	result, err := service.Check(Version, targetVersion)
	if err != nil {
		return err
	}

	switch {
	case result.CurrentIsDev:
		fmt.Printf("aem %s (no release version bundled); available release: %s\n", result.Current, result.Target)
		fmt.Println("Run `aem update --force` to install it.")
	case result.UpdateAvailable:
		fmt.Printf("Update available: %s -> %s\n", result.Current, result.Target)
		fmt.Println("Run `aem update` to install it.")
	case result.UpToDate:
		fmt.Printf("aem is already up to date (%s)\n", result.Current)
	default:
		fmt.Printf("aem %s is newer than the requested release %s\n", result.Current, result.Target)
		fmt.Println("Rerun with --force to downgrade.")
	}
	return nil
}
