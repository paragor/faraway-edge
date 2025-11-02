/*
Copyright © 2025 Egor Novikov aka paragor <novikov46en@gmail.com>
*/
package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/paragor/faraway-edge/pkg/diags"
	"github.com/paragor/faraway-edge/pkg/envoy"
	"github.com/paragor/faraway-edge/pkg/k8s"
	"github.com/paragor/faraway-edge/pkg/log"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Envoy xDS control plane server",
	Long: `Start the Envoy xDS control plane server that provides dynamic configuration
to Envoy proxies via the xDS protocol.

The server reads a LogicalCluster configuration from a JSON file and translates it
into Envoy listener, cluster, and route resources. It serves these configurations
via gRPC on the specified xDS port.

Example:
  faraway-edge run --static-path config.json
  faraway-edge run --static-path config.json --xds-port 19000`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		logger := log.FromContext(ctx)

		xdsPort, _ := cmd.Flags().GetInt("xds-port")
		staticPath, _ := cmd.Flags().GetString("static-path")
		token, _ := cmd.Flags().GetString("token")

		providers := []envoy.LogicalClusterProvider{}
		if staticPath != "" {
			data, err := os.ReadFile(staticPath)
			if err != nil {
				logger.Error("Error reading file", slog.String("path", staticPath), log.Error(err))
				os.Exit(1)
			}

			cluster := &envoy.LogicalCluster{}
			if err := json.Unmarshal(data, cluster); err != nil {
				logger.Error("Error parsing JSON", slog.String("path", staticPath), log.Error(err))
				os.Exit(1)
			}

			if err := cluster.Validate(); err != nil {
				logger.Error("Configuration validation failed", log.Error(err))
				os.Exit(1)
			}
			providers = append(providers, envoy.NewStaticLogicalClusterProvider(cluster))
		}

		// Set up signal handling
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Handle signals in background
		go func() {
			sig := <-sigChan
			logger.Info("Received signal, initiating graceful shutdown", slog.String("signal", sig.String()))
			cancel()
		}()

		k8sProviderErrChan := make(chan error, 1)
		k8sEnabled, _ := cmd.Flags().GetBool("k8s-enabled")
		if k8sEnabled {
			k8sClusterName, _ := cmd.Flags().GetString("k8s-cluster-name")
			k8sIngressClasses, _ := cmd.Flags().GetString("k8s-ingress-classes")

			ics := []string{}
			if len(k8sIngressClasses) > 0 {
				for _, ic := range strings.Split(k8sIngressClasses, ",") {
					ic = strings.TrimSpace(ic)
					if ic != "" {
						ics = append(ics, ic)
					}
				}
			}

			clientset, err := k8s.NewClientset()
			if err != nil {
				logger.Error("Cant init k8s clientset", log.Error(err))
				os.Exit(1)
			}
			k8sProvider, err := k8s.NewIngressProvider(k8sClusterName, ics, clientset, time.Hour*24)
			if err != nil {
				logger.Error("Cant init k8s provider", log.Error(err))
				os.Exit(1)
			}

			go func() {
				k8sProviderErrChan <- k8sProvider.Run(ctx)
			}()

			providers = append(providers, k8sProvider)
		}

		// Create HTTP server
		httpServer := diags.NewHTTPServer(8080)

		// Start HTTP server in background
		httpErrChan := make(chan error, 1)
		go func() {
			if err := httpServer.Run(ctx); err != nil {
				httpErrChan <- err
			}
		}()

		// Create and initialize XDS
		xds := envoy.NewXDS(
			xdsPort,
			providers,
			token,
		)

		// Start XDS server in background
		xdsErrChan := make(chan error, 1)
		go func() {
			xdsErrChan <- xds.RunServer(ctx, time.Second*30)
		}()

		// Wait a bit for XDS to start, then mark HTTP server as ready
		time.Sleep(100 * time.Millisecond)
		httpServer.SetReady(true)

		// Wait for either server to error or context cancellation
		select {
		case err := <-k8sProviderErrChan:
			if err != nil {
				logger.Error("Error running xDS server", log.Error(err))
				os.Exit(1)
			}
		case err := <-xdsErrChan:
			if err != nil {
				logger.Error("Error running xDS server", log.Error(err))
				os.Exit(1)
			}
		case err := <-httpErrChan:
			if err != nil {
				logger.Error("Error running HTTP server", log.Error(err))
				os.Exit(1)
			}
		case <-ctx.Done():
			logger.Info("Shutdown complete")
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().Int("xds-port", 18000, "Port for XDS server")
	runCmd.Flags().String("static-path", "", "Path to JSON file containing LogicalCluster configuration (optional)")
	runCmd.Flags().String("token", "", "Authentication token for gRPC xDS server (optional)")
	runCmd.Flags().Bool("k8s-enabled", true, "Enable local k8s")
	runCmd.Flags().String("k8s-cluster-name", "k8s-local", "K8s cluster name")
	runCmd.Flags().String("k8s-ingress-classes", "", "k8s ingress class classes split by ,")
}
