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
	"os"
	"path/filepath"
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
	"github.com/manifoldco/promptui"
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
	transferCmd.Flags().Bool("interactive", false, "Run interactive setup for transfer")

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

	// Check for interactive flag
	interactive, _ := cmd.Flags().GetBool("interactive")
	if interactive {
		// return runInteractiveMode(cmd)
		runInteractiveMode(cmd)
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

func getGithubURL() promptui.Prompt {
	return promptui.Prompt{
		Label:    "Step 2: Enter GitHub repository or organization URL (e.g., https://github.com/interlynk-io/sbomqs)",
		Validate: validateGithubURLPrompt,
		Default:  "https://github.com/interlynk-io/sbomqs",
	}
}

func selectGithubMethod() promptui.Select {
	return promptui.Select{
		Label: "Select GitHub method",
		Items: []string{
			"API (Dependency Graph)",
			"Release (from release page)",
			"Tool (generate with Syft)",
		},
	}
}

func getFolderPath() promptui.Prompt {
	return promptui.Prompt{
		Label: "Step 2: Enter folder path containing SBOMs (e.g., ./sboms)",
		Validate: func(input string) error {
			if info, err := os.Stat(input); err != nil || !info.IsDir() {
				return fmt.Errorf("path must be a valid directory")
			}
			return nil
		},
	}
}

func selectFolderRecurssiveScan() promptui.Select {
	return promptui.Select{
		Label: "Scan sub-folders recursively?",
		Items: []string{"Yes", "No"},
	}
}

func getBucketName() promptui.Prompt {
	return promptui.Prompt{
		Label: "Step 2: Enter S3 bucket name",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("bucket name cannot be empty")
			}
			return nil
		},
	}
}

func getS3Prefix() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter S3 prefix (optional, press Enter to skip)",
	}
}

func getS3Region() promptui.Prompt {
	return promptui.Prompt{
		Label:   "Enter S3 region (e.g., us-east-1)",
		Default: "us-east-1",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("region cannot be empty")
			}
			return nil
		},
	}
}

func getS3AccessKey() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter S3 access key",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("access key cannot be empty")
			}
			return nil
		},
	}
}

func getS3SecretKey() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter S3 secret key",
		Mask:  '*',
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("secret key cannot be empty")
			}
			return nil
		},
	}
}

func selectOutputAdapter() promptui.Select {
	return promptui.Select{
		Label: "Step 3: Choose an output destination (where SBOMs go)",
		Items: []string{
			"Folder (save to a local directory)",
			"S3 (upload to an AWS S3 bucket)",
			"Dependency Track (send to a server)",
			"Interlynk (upload to Interlynk platform)",
		},
	}
}

func getTargetFolderPath() promptui.Prompt {
	return promptui.Prompt{
		Label:   "Step 4: Enter folder path to save SBOMs (e.g., ./temp)",
		Default: "temp",
	}
}

func getTargetBucketName() promptui.Prompt {
	return promptui.Prompt{
		Label: "Step 4: Enter S3 bucket name",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("bucket name cannot be empty")
			}
			return nil
		},
	}
}

func getTargetPrefix() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter S3 prefix (optional, press Enter to skip)",
	}
}

func getTargetRegion() promptui.Prompt {
	return promptui.Prompt{
		Label:   "Enter S3 region (e.g., us-east-1)",
		Default: "us-east-1",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("region cannot be empty")
			}
			return nil
		},
	}
}

func getDtrackURL() promptui.Prompt {
	return promptui.Prompt{
		Label: "Step 4: Enter Dependency Track API URL (e.g., http://localhost:8081)",
		Validate: func(input string) error {
			if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
				return fmt.Errorf("URL must start with http:// or https://")
			}
			return nil
		},
		Default: "http://localhost:8081",
	}
}

func getDTrackAPIKey() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter Dependency Track API key",
		Mask:  '*',
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("API key cannot be empty")
			}
			return nil
		},
	}
}

func getInterlynkURL() promptui.Prompt {
	return promptui.Prompt{
		Label:   "Step 4: Enter Interlynk API URL (e.g., https://api.interlynk.io/lynkapi)",
		Default: "https://api.interlynk.io/lynkapi",
		Validate: func(input string) error {
			if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
				return fmt.Errorf("URL must start with http:// or https://")
			}
			return nil
		},
	}
}

func getInterlynkProjectName() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter project name(optional)",
	}
}

func getInterlynkSecurityToken() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter Interlynk security token",
		Mask:  '*',
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("security token cannot be empty")
			}
			return nil
		},
	}
}

func selectInputAdapter() promptui.Select {
	return promptui.Select{
		Label: "Step 1: Choose an input source/adapter (where SBOMs come from)",
		Items: []string{
			"GitHub (fetch from  API, Release, Tool)",
			"Folder (use local SBOM files)",
			"S3 (pull from an AWS S3 bucket)",
		},
	}
}

func runInteractiveMode(cmd *cobra.Command) error {
	fmt.Println("Welcome to the sbommv transfer interactive setup!")

	// Step 1: Choose input adapter
	inputPrompt := selectInputAdapter()
	inputIndex, _, err := inputPrompt.Run()
	if err != nil {
		return fmt.Errorf("input source selection failed: %w", err)
	}

	// fmt.Println("Step 1: Input source/adapter (where SBOMs come from): ", inputResult)

	inputAdapters := []string{"github", "folder", "s3"}
	inputAdapter := inputAdapters[inputIndex]

	// Step 2: Collect input adapter parameters
	var inputFlags []string

	switch inputAdapter {
	case "github":

		// github URL prompt
		urlPrompt := getGithubURL()
		url, err := urlPrompt.Run()
		if err != nil {
			return fmt.Errorf("GitHub URL input failed: %w", err)
		}
		inputFlags = append(inputFlags, fmt.Sprintf("--in-github-url=%s", url))

		// method selection
		methodPrompt := selectGithubMethod()
		methodIndex, _, err := methodPrompt.Run()
		if err != nil {
			return fmt.Errorf("GitHub method selection failed: %w", err)
		}
		methods := []string{"api", "release", "tool"}
		inputFlags = append(inputFlags, fmt.Sprintf("--in-github-method=%s", methods[methodIndex]))

	case "folder":

		// folder path prompt
		pathPrompt := getFolderPath()
		path, err := pathPrompt.Run()
		if err != nil {
			return fmt.Errorf("folder path input failed: %w", err)
		}
		inputFlags = append(inputFlags, fmt.Sprintf("--in-folder-path=%s", path))

		// recursive scan selection
		recursivePrompt := selectFolderRecurssiveScan()
		recursiveIndex, _, err := recursivePrompt.Run()
		if err != nil {
			return fmt.Errorf("recursive selection failed: %w", err)
		}
		if recursiveIndex == 0 {
			inputFlags = append(inputFlags, "--in-folder-recursive")
		}

	case "s3":
		// S3 bucket name prompt
		bucketPrompt := getBucketName()
		bucket, err := bucketPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 bucket input failed: %w", err)
		}
		inputFlags = append(inputFlags, fmt.Sprintf("--in-s3-bucket-name=%s", bucket))

		// S3 prefix and region prompts
		prefixPrompt := getS3Prefix()
		prefix, err := prefixPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 prefix input failed: %w", err)
		}
		if prefix != "" {
			inputFlags = append(inputFlags, fmt.Sprintf("--in-s3-prefix=%s", prefix))
		}

		// S3 region prompt
		regionPrompt := getS3Region()
		region, err := regionPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 region input failed: %w", err)
		}
		inputFlags = append(inputFlags, fmt.Sprintf("--in-s3-region=%s", region))

		// check for default AWS credentials
		if !hasDefaultAWSCredentials() {

			// prompt for AWS Access Key
			accessKeyPrompt := getS3AccessKey()
			accessKey, err := accessKeyPrompt.Run()
			if err != nil {
				return fmt.Errorf("S3 access key input failed: %w", err)
			}
			inputFlags = append(inputFlags, fmt.Sprintf("--in-s3-access-key=%s", accessKey))

			// prompt for AWS Secret Key
			secretKeyPrompt := getS3SecretKey()
			secretKey, err := secretKeyPrompt.Run()
			if err != nil {
				return fmt.Errorf("S3 secret key input failed: %w", err)
			}
			inputFlags = append(inputFlags, fmt.Sprintf("--in-s3-secret-key=%s", secretKey))
		}

	default:
		return fmt.Errorf("unsupported input adapter: %s", inputAdapter)

	}

	// Step 3: Choose output adapter
	outputPrompt := selectOutputAdapter()
	outputIndex, _, err := outputPrompt.Run()
	if err != nil {
		return fmt.Errorf("output destination selection failed: %w", err)
	}
	outputAdapters := []string{"folder", "s3", "dtrack", "interlynk"}
	outputAdapter := outputAdapters[outputIndex]

	// Step 4: Collect output adapter parameters
	var outputFlags []string

	switch outputAdapter {
	case "folder":
		// folder path prompt
		pathPrompt := getTargetFolderPath()
		path, err := pathPrompt.Run()
		if err != nil {
			return fmt.Errorf("folder path input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-folder-path=%s", path))

	case "s3":
		// S3 bucket name prompt
		bucketPrompt := getTargetBucketName()
		bucket, err := bucketPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 bucket input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-s3-bucket-name=%s", bucket))

		// S3 prefix prompts
		prefixPrompt := getTargetPrefix()
		prefix, err := prefixPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 prefix input failed: %w", err)
		}
		if prefix != "" {
			outputFlags = append(outputFlags, fmt.Sprintf("--out-s3-prefix=%s", prefix))
		}

		// S3 region prompt
		regionPrompt := getTargetRegion()
		region, err := regionPrompt.Run()
		if err != nil {
			return fmt.Errorf("S3 region input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-s3-region=%s", region))

		// check for default AWS credentials
		if !hasDefaultAWSCredentials() {
			accessKeyPrompt := getS3AccessKey()
			accessKey, err := accessKeyPrompt.Run()
			if err != nil {
				return fmt.Errorf("S3 access key input failed: %w", err)
			}
			outputFlags = append(outputFlags, fmt.Sprintf("--out-s3-access-key=%s", accessKey))

			secretKeyPrompt := getS3SecretKey()
			secretKey, err := secretKeyPrompt.Run()
			if err != nil {
				return fmt.Errorf("S3 secret key input failed: %w", err)
			}
			outputFlags = append(outputFlags, fmt.Sprintf("--out-s3-secret-key=%s", secretKey))
		}

	case "dtrack":
		// Dependency Track URL prompt
		urlPrompt := getDtrackURL()
		url, err := urlPrompt.Run()
		if err != nil {
			return fmt.Errorf("Dependency Track URL input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-dtrack-url=%s", url))

		// project name prompt
		projectPrompt := promptui.Prompt{
			Label: "Enter project name(optional)",
		}
		project, err := projectPrompt.Run()
		if err != nil {
			return fmt.Errorf("project name input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-dtrack-project-name=%s", project))

		// dtrack API key
		if !isDTrackAPIKeyExported() {
			apiKeyPrompt := getDTrackAPIKey()
			apiKey, err := apiKeyPrompt.Run()
			if err != nil {
				return fmt.Errorf("Dependency Track API key input failed: %w", err)
			}
			outputFlags = append(outputFlags, fmt.Sprintf("--out-dtrack-api-key=%s", apiKey))
		}

	case "interlynk":

		// Interlynk URL prompt
		urlPrompt := getInterlynkURL()
		url, err := urlPrompt.Run()
		if err != nil {
			return fmt.Errorf("Interlynk URL input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-interlynk-url=%s", url))

		// Interlynk project name prompt
		projectPrompt := getInterlynkProjectName()
		project, err := projectPrompt.Run()
		if err != nil {
			return fmt.Errorf("project name input failed: %w", err)
		}
		outputFlags = append(outputFlags, fmt.Sprintf("--out-interlynk-project-name=%s", project))

		// Interlynk security token prompt
		if !isInterlynkSecurityKeyKeyExported() {
			tokenPrompt := getInterlynkSecurityToken()
			token, err := tokenPrompt.Run()
			if err != nil {
				return fmt.Errorf("Interlynk security token input failed: %w", err)
			}
			outputFlags = append(outputFlags, fmt.Sprintf("--out-interlynk-security-token=%s", token))
		}

	default:
		return fmt.Errorf("unsupported output adapter: %s", outputAdapter)
	}

	// Step 5: Collect global flags
	var globalFlags []string

	fmt.Println("Step 5: Global flags")
	fmt.Println("  Enable debug logging for detailed output")

	// debug mode prompt
	debugPrompt := selectDebugMode()
	debugIndex, _, err := debugPrompt.Run()
	if err != nil {
		return fmt.Errorf("debug mode selection failed: %w", err)
	}
	fmt.Println("debugIndex: ", debugIndex)
	if debugIndex != 0 {
		globalFlags = append(globalFlags, "--debug")
	}

	// dry-run prompt
	fmt.Println("  Run in dry-run mode (simulate without executing)?")
	dryRunPrompt := selectDryRun()
	dryRunIndex, _, err := dryRunPrompt.Run()
	if err != nil {
		return fmt.Errorf("dry-run selection failed: %w", err)
	}
	if dryRunIndex != 0 {
		globalFlags = append(globalFlags, "--dry-run")
	}

	fmt.Println("  Choose processing mode")

	// processing mode prompt
	processingModePrompt := selectProcessingMode()
	processingModeIndex, _, err := processingModePrompt.Run()
	if err != nil {
		return fmt.Errorf("processing mode selection failed: %w", err)
	}
	processingModes := []string{"sequential", "parallel"}
	globalFlags = append(globalFlags, fmt.Sprintf("--processing-mode=%s", processingModes[processingModeIndex]))

	fmt.Println("  Overwrite existing SBOMs at destination?")

	// overwrite prompt
	overwritePrompt := selectOverwrite()
	overwriteIndex, _, err := overwritePrompt.Run()
	if err != nil {
		return fmt.Errorf("overwrite selection failed: %w", err)
	}
	if overwriteIndex != 0 {
		globalFlags = append(globalFlags, "--overwrite")
	}

	// construct the command
	commandFlags := append(append([]string{
		fmt.Sprintf("--input-adapter=%s", inputAdapter),
		fmt.Sprintf("--output-adapter=%s", outputAdapter),
	}, globalFlags...), append(inputFlags, outputFlags...)...)

	commandStr := fmt.Sprintf("sbommv transfer %s", strings.Join(commandFlags, " "))

	// confirm execution
	fmt.Printf("\nReady to run this command:\n%s\n", commandStr)
	confirmPrompt := promptui.Select{
		Label: "Run now?",
		Items: []string{"Yes", "No"},
	}

	confirmIndex, _, err := confirmPrompt.Run()
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if confirmIndex != 0 {
		fmt.Println("Transfer cancelled.")
		os.Exit(0)
	}

	// Set flags on cmd for engine.TransferRun
	cmd.Flags().Set("input-adapter", inputAdapter)
	cmd.Flags().Set("output-adapter", outputAdapter)
	for _, flag := range append(append(globalFlags, inputFlags...), outputFlags...) {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) == 2 {
			cmd.Flags().Set(strings.TrimPrefix(parts[0], "--"), parts[1])
		} else {
			cmd.Flags().Set(strings.TrimPrefix(parts[0], "--"), "true")
		}
	}

	return nil
}

func selectDryRun() promptui.Select {
	return promptui.Select{
		Label: "Step 5: Run in dry-run mode (simulate without executing)?",
		Items: []string{"No", "Yes"},
	}
}

func selectDebugMode() promptui.Select {
	return promptui.Select{
		Label: "Step 5: Enable debug logging for detailed output?",
		Items: []string{"No", "Yes"},
	}
}

func selectProcessingMode() promptui.Select {
	return promptui.Select{
		Label: "Step 5: Choose processing mode",
		Items: []string{
			"Sequential (process one SBOM at a time)",
			"Parallel (process multiple SBOMs simultaneously)",
		},
	}
}

func selectOverwrite() promptui.Select {
	return promptui.Select{
		Label: "Step 5: Overwrite existing SBOMs at destination?",
		Items: []string{"No", "Yes"},
	}
}

func isDTrackAPIKeyExported() bool {
	if os.Getenv("DTRACK_API_KEY") != "" {
		return true
	}
	return false
}

func isInterlynkSecurityKeyKeyExported() bool {
	if os.Getenv("DTRACK_API_KEY") != "" {
		return true
	}
	return false
}

// hasDefaultAWSCredentials checks if default AWS credentials are available
func hasDefaultAWSCredentials() bool {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	// Check ~/.aws/credentials file
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	credFile := filepath.Join(home, ".aws", "credentials")
	if _, err := os.Stat(credFile); err == nil {
		return true
	}

	return false
}

func validateGithubURLPrompt(input string) error {
	if !strings.HasPrefix(input, "https://github.com/") {
		return fmt.Errorf("URL must start with https://github.com/")
	}
	return nil
}
