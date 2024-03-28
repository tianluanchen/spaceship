package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/tianluanchen/spaceship/ship"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",

	Run: func(cmd *cobra.Command, args []string) {
		addr, _ := cmd.Flags().GetString("addr")
		logger.Infof("Listen address: %+v", addr)
		noProxyRoute, _ := cmd.Flags().GetBool("no-proxy-route")
		if noProxyRoute {
			logger.Infof("Disabled the route that provides proxy")
		}

		root, _ := cmd.Flags().GetString("root")
		auth, _ := cmd.Flags().GetString("auth")
		prefix, _ := cmd.Flags().GetString("prefix")
		srv := http.Server{
			Addr: addr,
			Handler: ship.NewService(ship.ServiceOption{
				URLPathPrefix: prefix,
				Root:          root,
				Auth:          auth,
			}),
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalln(err)
			}
		}()
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt)
		<-ch
		logger.Info("Shutting down server...")
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Fatalln(err)
		} else {
			logger.Warnln("Server stopped")
		}
	},
}

func init() {
	serveCmd.Flags().String("root", "./", "root directory")
	serveCmd.Flags().String("auth", "", "auth key")
	serveCmd.Flags().String("prefix", "/", "url prefix")
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "listen address")
	rootCmd.AddCommand(serveCmd)
}
