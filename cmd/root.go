// Copyright 2025 riteshnoronha
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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sbommv",
	Short: "sbommv is a CLI tool for transferring SBOMs between systems",
	Long:  `sbommv helps in transferring SBOMs from GitHub repositories to Interlynk or other systems.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to sbommv!")
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
