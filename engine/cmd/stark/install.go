package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GetEvinced/stark-marketplace/engine/internal/indexio"
	"github.com/GetEvinced/stark-marketplace/engine/internal/install"
	"github.com/GetEvinced/stark-marketplace/engine/internal/installplan"
	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
	"github.com/spf13/cobra"
)

func newInstallCmd(adapterFactory func(catalogDir string) installplan.Adapter) *cobra.Command {
	var rt, dest, indexPath, bundlesDir, catalogDir, removeManifest string
	var plan, force, repair, jsonOut, yes bool
	cmd := &cobra.Command{
		Use:   "install <bundle[/artifact]>",
		Short: "Install a bundle/artifact for a runtime (safe-merge, consent, atomic)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if repair {
				if err := install.Repair(dest); err != nil {
					fmt.Fprintln(os.Stderr, "repair:", err)
					os.Exit(ExitValidation)
				}
				fmt.Println("repair complete")
				return nil
			}
			if removeManifest != "" {
				if err := install.Remove(dest, removeManifest); err != nil {
					fmt.Fprintln(os.Stderr, "remove:", err)
					os.Exit(ExitValidation)
				}
				fmt.Println("removed")
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("install requires <bundle[/artifact]>")
			}
			r, err := model.ParseRuntime(rt)
			if err != nil {
				return err
			}
			idx, err := indexio.LoadIndex(indexPath)
			if err != nil {
				os.Exit(indexLoadExit(err))
			}
			bundle, artifact := splitRef(args[0])
			typ := rootType(idx, bundle, artifact)
			p, err := installplan.Compute(idx, bundlesDir, adapterFactory(catalogDir), bundle, artifact, typ, r)
			if err != nil {
				fmt.Fprintln(os.Stderr, "plan:", err)
				os.Exit(ExitValidation)
			}
			printPlan(p)
			if plan {
				return nil // --plan: show, don't write
			}
			if p.Consent.Required && !yes {
				if !confirm() {
					fmt.Println("declined")
					os.Exit(ExitConsentDeclined)
				}
			}
			res, err := install.Install(dest, p, install.Options{Force: force})
			if err != nil {
				if _, ok := err.(*install.ConflictError); ok {
					fmt.Fprintln(os.Stderr, "conflict:", err)
					os.Exit(ExitConflict)
				}
				if strings.Contains(err.Error(), "digest mismatch") {
					os.Exit(ExitDigest)
				}
				fmt.Fprintln(os.Stderr, "install:", err)
				os.Exit(ExitValidation)
			}
			if jsonOut {
				emitJSON(os.Stdout, "install", ExitOK, map[string]any{
					"runtime": r, "written": res.Written, "merged": res.Merged,
					"manifest": res.ManifestPath, "skipped": p.Skipped})
			} else {
				fmt.Printf("installed: %d written, %d merged → %s\n", res.Written, res.Merged, res.ManifestPath)
				if len(p.Skipped) > 0 {
					fmt.Printf("skipped (do not target %s): %s\n", r, strings.Join(p.Skipped, ", "))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rt, "runtime", "", "target runtime (claude|codex|gemini)")
	cmd.Flags().StringVar(&dest, "dest", ".", "destination root (test/install dir)")
	cmd.Flags().StringVar(&indexPath, "index", "index.json", "path to index.json")
	cmd.Flags().StringVar(&bundlesDir, "bundles", "bundles", "path to bundles/ dir")
	cmd.Flags().StringVar(&catalogDir, "catalog", "catalog", "path to source catalog/ dir (rendered on install)")
	cmd.Flags().BoolVar(&plan, "plan", false, "show plan + consent without writing")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite unmanaged collisions")
	cmd.Flags().BoolVar(&yes, "yes", false, "assume yes to consent (non-interactive)")
	cmd.Flags().StringVar(&removeManifest, "remove", "", "remove a prior install by manifest path")
	cmd.Flags().BoolVar(&repair, "repair", false, "recover a crashed/partial install under --dest")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable output")
	return cmd
}

// rootType resolves the artifact type from the index (bundle install uses any type).
func rootType(idx *indexio.Index, bundle, artifact string) model.ArtifactType {
	if artifact == "" {
		return model.TypeCommand // unused for bundle installs (Compute enumerates all)
	}
	for _, t := range model.AllArtifactTypes() {
		if idx.Find(bundle, artifact, t) != nil {
			return t
		}
	}
	return model.TypeCommand
}

func printPlan(p *installplan.Plan) {
	fmt.Printf("plan for runtime %s:\n", p.Runtime)
	for _, s := range p.Steps {
		for _, f := range s.Files {
			tag := ""
			if f.Emulated {
				tag = " (emulated)"
			}
			fmt.Printf("  %s/%s → %s [%s]%s\n", s.Bundle, s.Name, f.Path, f.Kind, tag)
		}
	}
	if len(p.Skipped) > 0 {
		fmt.Printf("  skipped: %s\n", strings.Join(p.Skipped, ", "))
	}
	if p.Consent.Required {
		fmt.Println("CONSENT REQUIRED — code-executing artifacts:")
		for _, c := range p.Consent.MCPCommands {
			fmt.Printf("  mcp  %s\n", c)
		}
		for _, g := range p.Consent.AgentToolGrants {
			fmt.Printf("  agent %s\n", g)
		}
		fmt.Printf("  full closure: %s\n", strings.Join(p.Consent.ClosureRefs, ", "))
	}
}

func confirm() bool {
	fmt.Print("Proceed? [y/N] ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	a := strings.TrimSpace(strings.ToLower(sc.Text()))
	return a == "y" || a == "yes"
}

var _ = filepath.Join // path anchor
