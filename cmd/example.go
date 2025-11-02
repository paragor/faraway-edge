/*
Copyright © 2025 Egor Novikov aka paragor <novikov46en@gmail.com>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/paragor/faraway-edge/pkg/encodinghelper"
	"github.com/paragor/faraway-edge/pkg/envoy"
	"github.com/spf13/cobra"
)

// exampleCmd represents the example command
var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Generate an example LogicalCluster configuration",
	Long: `Generate an example LogicalCluster configuration in JSON format that can be
used as a template for creating your own configuration file.

The example includes:
  - A logical cluster with HTTP and HTTPS upstreams
  - Multiple domain frontends
  - Properly formatted duration values
  - Static IP address configurations

Example:
  faraway-edge example > config.json
  faraway-edge example | jq .`,
	Run: func(cmd *cobra.Command, args []string) {
		cluster := &envoy.LogicalCluster{
			Name: "static",
			Ingresses: []*envoy.LogicalClusterIngress{
				{
					Name: "example",
					HttpUpstream: &envoy.EnvoyUpstreamStaticAddresses{
						Port:            80,
						StaticAddresses: []string{"10.10.10.10"},
						ConnectTimeout:  encodinghelper.NewDuration(time.Second),
					},
					HttpsUpstream: &envoy.EnvoyUpstreamStaticAddresses{
						Port:            443,
						StaticAddresses: []string{"12.12.12.12"},
						ConnectTimeout:  encodinghelper.NewDuration(time.Second),
					},
					Frontends: []*envoy.IngressConfig{
						{
							Domain: "first.example.com",
						},
						{
							Domain: "second.example.com",
						},
					},
				},
			},
		}
		data, err := json.MarshalIndent(cluster, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)

}
