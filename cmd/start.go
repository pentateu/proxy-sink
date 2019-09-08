/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the start command
var serveCmd = &cobra.Command{
	Use:   "start",
	Short: "Start proxy-sink",
	Long: `Reverse proxy that saves incoming request to a Redis
		   and return mock results or forward to a target system and return real results.`,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")

	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// sink will store data and return predefine mock result
	serveCmd.Flags().StringP("mock", "m", "", "return mock results")
	serveCmd.Flags().StringP("forward", "f", "", "forward request to target system and return result.")

	viper.BindPFlag("mock", rootCmd.PersistentFlags().Lookup("mock"))
	viper.BindPFlag("forward", rootCmd.PersistentFlags().Lookup("forward"))

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
