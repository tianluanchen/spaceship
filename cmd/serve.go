package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"spaceship/ship"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Start the server",
	Long:    "Start the server, if specified cert and key, listen on https",
	Example: "serve <root directory?>",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}
		addr, _ := cmd.Flags().GetString("addr")
		auth, _ := cmd.Flags().GetString("auth")
		prefix, _ := cmd.Flags().GetString("prefix")
		keyfile, _ := cmd.Flags().GetString("keyfile")
		certfile, _ := cmd.Flags().GetString("certfile")
		if !(certfile == "" && keyfile == "") && !(certfile != "" && keyfile != "") {
			logger.Fatalln("specify either both certfile and keyfile or none")
		}
		svc := ship.NewService(ship.ServiceOption{
			URLPathPrefix: prefix,
			Root:          root,
			Auth:          auth,
		})
		srv := http.Server{
			Addr:    addr,
			Handler: svc,
		}
		logger.Infof("Root directory: %s", svc.GetRoot())
		logger.Infof("URL Path Prefix: %s", prefix)
		if auth != "" {
			logger.Infof("Use Auth: %s", logger.Green("true"))
		} else {
			logger.Infof("Use Auth: %s", logger.Yellow("false"))
		}
		logger.Infof("Use TLS certificate: %v", certfile != "")
		logger.Infof("Listen address: %s", addr)
		go func() {
			var err error
			if certfile != "" {
				err = srv.ListenAndServeTLS(certfile, keyfile)
			} else {
				err = srv.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
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
	serveCmd.Flags().String("auth", "", "auth key")
	serveCmd.Flags().String("prefix", "/", "url prefix")
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "listen address")
	serveCmd.Flags().String("keyfile", "", "specify private key file")
	serveCmd.Flags().String("certfile", "", "specify certificate file")
	rootCmd.AddCommand(serveCmd)
}
