// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/interlynk-io/sbommv/pkg/engine"
	ifolder "github.com/interlynk-io/sbommv/pkg/source/folder"
	is3 "github.com/interlynk-io/sbommv/pkg/source/s3"
	"github.com/interlynk-io/sbommv/pkg/target/dependencytrack"
	ofolder "github.com/interlynk-io/sbommv/pkg/target/folder"
	os3 "github.com/interlynk-io/sbommv/pkg/target/s3"

	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/interlynk-io/sbommv/pkg/types"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FlagData holds information about a flag for template rendering
type FlagData struct {
	Name      string
	Shorthand string
	Usage     string
	ValueType string
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long:  `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).`,
	Args:  cobra.NoArgs,
	RunE:  transferSBOM,
}

func init() {
	rootCmd.AddCommand(transferCmd)

	// General Flags
	transferCmd.Flags().BoolP("daemon", "d", false, "Enable daemon mode")
	transferCmd.Flags().BoolP("debug", "D", false, "Enable debug logging")
	transferCmd.Flags().Bool("dry-run", false, "Simulate transfer without executing")
	transferCmd.Flags().String("processing-mode", "sequential", "Processing strategy (sequential, parallel)")
	transferCmd.Flags().Bool("overwrite", false, "Overwrite existing SBOMs at destination")
	transferCmd.Flags().Bool("guide", false, "Show beginner-friendly guide")

	// Input and Output Adapter Flags(both required)
	transferCmd.Flags().String("input-adapter", "", "Input adapter type (github, folder, s3)")
	transferCmd.Flags().String("output-adapter", "", "Output adapter type (folder, s3, dtrack, interlynk)")

	registerAdapterFlags(transferCmd)

	// Define custom template functions
	funcMap := template.FuncMap{
		"prefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"eq": func(a, b string) bool {
			return a == b
		},
	}

	// define the help template as a string
	const helpTemplate = `
{{.Command.Short}}

Usage:
  {{.Command.UseLine}}

Examples:
  # GitHub (release) to Folder
  sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" --in-github-method=release \
                  --output-adapter=folder --out-folder-path="temp"

  # Folder to S3
  sbommv transfer --input-adapter=folder --in-folder-path="temp" --in-folder-recursive \
                  --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="sboms" --out-s3-region="us-east-1"

  # S3 to Dependency Track
  sbommv transfer --input-adapter=s3 --in-s3-bucket-name="source-test-sbom" --in-s3-prefix="dropwizard" --in-s3-region="us-east-1" \
                  --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" --out-dtrack-project-name="my-project"

  # GitHub (api) to Interlynk
  sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                  --output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi" --out-interlynk-project-name="sbomqs"

General Flags:
{{- range .Flags}}
{{- if and (not (or (prefix .Name "in-") (prefix .Name "out-"))) (not (eq .Name "input-adapter")) (not (eq .Name "output-adapter"))}}
  {{if .Shorthand}}-{{.Shorthand}}, {{end}}--{{.Name}}{{if eq .ValueType "string"}} string{{end}}  {{.Usage}}
{{- end}}
{{- end}}

Input Adapter Flags(required):
  --input-adapter string  Input adapter type (github, folder, s3)

  GitHub Input Adapter:
{{- range .Flags}}
{{- if prefix .Name "in-github-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

  Folder Input Adapter(required):
{{- range .Flags}}
{{- if prefix .Name "in-folder-"}}
    --{{.Name}} {{if eq .ValueType "bool"}}{{else}}{{.ValueType}}{{end}}  {{.Usage}}
{{- end}}
{{- end}}

  S3 Input Adapter:
{{- range .Flags}}
{{- if prefix .Name "in-s3-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

Output Adapter Flags(required):
  --output-adapter string  Output adapter type (folder, s3, dtrack, interlynk)

  Folder Output Adapter:
{{- range .Flags}}
{{- if prefix .Name "out-folder-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

  S3 Output Adapter:
{{- range .Flags}}
{{- if prefix .Name "out-s3-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

  Dependency Track Output Adapter:
{{- range .Flags}}
{{- if prefix .Name "out-dtrack-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

  Interlynk Output Adapter:
{{- range .Flags}}
{{- if prefix .Name "out-interlynk-"}}
    --{{.Name}} {{.ValueType}}  {{.Usage}}
{{- end}}
{{- end}}

Run 'sbommv transfer --guide' for a beginner-friendly guide or visit https://github.com/interlynk-io/sbommv/tree/main/examples for more examples.
`

	// Set custom help function to render template with funcMap
	transferCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Collect all flags into a slice
		var flags []FlagData
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, FlagData{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Usage:     f.Usage,
				ValueType: f.Value.Type(),
			})
		})

		// Data for template
		data := struct {
			Command *cobra.Command
			Flags   []FlagData
		}{
			Command: cmd,
			Flags:   flags,
		}

		// Parse and render template
		tmpl, err := template.New("help").Funcs(funcMap).Parse(helpTemplate)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error parsing help template: %v\n", err)
			return
		}

		// Execute template with data
		if err := tmpl.Execute(cmd.OutOrStdout(), data); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error rendering help template: %v\n", err)
		}
	})
}

// registerAdapterFlags dynamically adds flags for the selected adapters after flag parsing
func registerAdapterFlags(cmd *cobra.Command) {
	// Register GitHub Adapter Flags
	githubAdapter := &github.GitHubAdapter{}
	githubAdapter.AddCommandParams(cmd)

	// Register Input Folder Adapter Flags
	folderInputAdapter := &ifolder.FolderAdapter{}
	folderInputAdapter.AddCommandParams(cmd)

	// Register Input S3 Adapter Flags
	s3InputAdapter := &is3.S3Adapter{}
	s3InputAdapter.AddCommandParams(cmd)

	// Register Output Interlynk Adapter Flags
	interlynkAdapter := &interlynk.InterlynkAdapter{}
	interlynkAdapter.AddCommandParams(cmd)

	// Register Output Folder Adapter Flags
	folderOutputAdapter := &ofolder.FolderAdapter{}
	folderOutputAdapter.AddCommandParams(cmd)

	dtrackAdapter := &dependencytrack.DependencyTrackAdapter{}
	dtrackAdapter.AddCommandParams(cmd)
	// similarly for all other Adapters

	s3OutputAdapter := &os3.S3Adapter{}
	s3OutputAdapter.AddCommandParams(cmd)
}

func transferSBOM(cmd *cobra.Command, args []string) error {
	// Check for guide flag
	guide, _ := cmd.Flags().GetBool("guide")
	if guide {
		fmt.Println(`Welcome to sbommv! The ` + "`transfer`" + ` command moves Software Bill of Materials (SBOMs) from one place to another.

Get started in 3 steps:
1. Choose an input source (where SBOMs come from):
   - GitHub: Fetch from repositories (e.g., a project’s code).
   - Folder: Use SBOM files from a local directory.
   - S3: Pull SBOMs from an AWS S3 bucket.
2. Choose an output destination (where SBOMs go):
   - Folder: Save to a local directory.
   - S3: Upload to an AWS S3 bucket.
   - Dependency Track: Send to a Dependency Track server.
   - Interlynk: Upload to the Interlynk platform.
3. Run a command like:
   sbommv transfer --input-adapter=folder --in-folder-path="sboms" --output-adapter=s3 --out-s3-bucket-name="my-bucket" --out-s3-prefix="sboms"
   sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"

For more details and options, run ` + "`sbommv transfer --help`" + `.
Explore examples at https://github.com/interlynk-io/sbommv/tree/main/examples.`)
		return nil
	}

	// Suppress automatic usage message for non-flag errors
	cmd.SilenceUsage = true

	// Initialize logger based on debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	logger.InitLogger(debug, false)
	defer logger.DeinitLogger()
	defer logger.Sync()

	ctx := logger.WithLogger(context.Background())
	viper.AutomaticEnv()
	logger.LogDebug(ctx, "Starting transferSBOM")

	// Parse config
	config, err := parseConfig(cmd)
	if err != nil {
		return err
	}

	logger.LogDebug(ctx, "configuration", "value", config)

	if err := engine.TransferRun(ctx, cmd, config); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func parseConfig(cmd *cobra.Command) (types.Config, error) {
	inputType, _ := cmd.Flags().GetString("input-adapter")
	outputType, _ := cmd.Flags().GetString("output-adapter")
	dr, _ := cmd.Flags().GetBool("dry-run")
	processingMode, _ := cmd.Flags().GetString("processing-mode")
	daemon, _ := cmd.Flags().GetBool("daemon")
	overwrite, _ := cmd.Flags().GetBool("overwrite")

	validInputAdapter := map[string]bool{"github": true, "folder": true, "s3": true}
	validOutputAdapter := map[string]bool{"interlynk": true, "folder": true, "dtrack": true, "s3": true}

	// Custom validation for required flags
	missingFlags := []string{}
	invalidFlags := []string{}

	if inputType == "" {
		missingFlags = append(missingFlags, "--input-adapter")
	}

	if outputType == "" {
		missingFlags = append(missingFlags, "--output-adapter")
	}

	validModes := map[string]bool{"sequential": true, "parallel": true}
	if !validModes[processingMode] {
		invalidFlags = append(invalidFlags, fmt.Sprintf("%s=%s (must be one of: sequential, parallel)", "--processing-mode", processingMode))
	}

	// Show error message if required flags are missing
	if len(invalidFlags) > 0 {
		return types.Config{}, fmt.Errorf("missing required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", invalidFlags)
	}

	// Show error message if required flags are missing
	if len(missingFlags) > 0 {
		return types.Config{}, fmt.Errorf("missing required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}

	if !validInputAdapter[inputType] {
		return types.Config{}, fmt.Errorf("input adapter must be one of type: github, folder")
	}

	if !validOutputAdapter[outputType] {
		return types.Config{}, fmt.Errorf("output adapter must be one of type: dtrack, interlynk, folder")
	}
	config := types.Config{
		SourceAdapter:      inputType,
		DestinationAdapter: outputType,
		DryRun:             dr,
		ProcessingStrategy: processingMode,
		Daemon:             daemon,
		Overwrite:          overwrite,
	}

	return config, nil
}
