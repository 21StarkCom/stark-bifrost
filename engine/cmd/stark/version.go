package main

import (
	"fmt"
	"os"

	"github.com/GetEvinced/stark-marketplace/engine/internal/indexio"
	"github.com/spf13/cobra"
)

// Version is overridden at release time via -ldflags "-X main.Version=...".
var Version = "dev"

func versionString() string {
	return fmt.Sprintf("stark %s (index schemaVersion %d..%d)",
		Version, indexio.SchemaVersionMin, indexio.SchemaVersionMax)
}

// checkIndexSupported maps an index schemaVersion to an exit code (spec §9.7).
func checkIndexSupported(v int) int {
	if err := indexio.AssertSchemaVersion(v); err != nil {
		return ExitSchemaVersion
	}
	return ExitOK
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version + supported index schemaVersion range",
		Run:   func(c *cobra.Command, args []string) { fmt.Println(versionString()) },
	}
}

func newSelfUpdateCmd() *cobra.Command {
	var indexPath string
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update the stark binary (asserts supported schemaVersion range)",
		RunE: func(c *cobra.Command, args []string) error {
			if indexPath != "" {
				idx, err := indexio.LoadIndex(indexPath)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(ExitSchemaVersion)
				}
				if code := checkIndexSupported(idx.SchemaVersion); code != ExitOK {
					os.Exit(code)
				}
			}
			fmt.Println(versionString())
			fmt.Println("self-update: download signed release binary from GitHub Releases (see docs §9.7)")
			return nil
		},
	}
	cmd.Flags().StringVar(&indexPath, "index", "", "optional index to assert compatibility")
	return cmd
}
