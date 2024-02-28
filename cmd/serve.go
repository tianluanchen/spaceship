package cmd

import (
	"net/http"
	"strings"

	"github.com/tianluanchen/spaceship/ship"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",

	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		logger.Infof("Listen address: %+v", addr)
		noProxyRoute, _ := cmd.Flags().GetBool("no-proxy-route")
		if noProxyRoute {
			logger.Infof("Disabled the route that provides proxy")
		}

		root, _ := cmd.Flags().GetString("root")
		auth, _ := cmd.Flags().GetString("auth")
		prefix, _ := cmd.Flags().GetString("prefix")
		srv := ship.NewService(ship.ServiceOption{
			URLPathPrefix: prefix,
			Root:          root,
			Auth:          auth,
		})
		return http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				srv.ServeHTTP(w, r)
			}
		}))
	},
}

func init() {
	serveCmd.Flags().String("root", "./", "root directory")
	serveCmd.Flags().String("auth", "", "auth key")
	serveCmd.Flags().String("prefix", "/", "url prefix")
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "listen address")
	rootCmd.AddCommand(serveCmd)
}
