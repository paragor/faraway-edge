/*
Copyright © 2025 Egor Novikov aka paragor <novikov46en@gmail.com>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "faraway-edge",
	Short: "Envoy xDS control plane for dynamic proxy configuration",
	Long: `faraway-edge is an Envoy xDS control plane that dynamically configures
Envoy proxies to route HTTP/HTTPS traffic based on domain names to backend clusters.

It provides a control plane that translates high-level logical cluster configurations
into low-level Envoy proxy configurations, serving them via the xDS gRPC protocol.

The control plane supports:
  - Domain-based HTTP routing
  - SNI-based HTTPS routing with TLS inspection
  - Static backend cluster configurations
  - Dynamic configuration updates`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.faraway-edge.yaml)")
}


