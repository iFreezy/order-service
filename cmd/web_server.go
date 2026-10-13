package cmd

import (
	"github.com/iFreezy/order-service/internal/app/builder"
	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	return &cli.App{Name: "order-service", Version: "1.0.0", Usage: "Order management service", Flags: []cli.Flag{&cli.BoolFlag{Name: "no-json", Usage: "Use human-readable logs"}}, Commands: []*cli.Command{WebServer()}}
}

func WebServer() *cli.Command {
	return &cli.Command{Name: "web-server", Aliases: []string{"ws"}, Usage: "Start HTTP server with all routes", Description: "Initializes configuration, database and handlers, then starts HTTP. Graceful shutdown on SIGINT/SIGTERM.", Action: cmdWebServer, HideHelpCommand: true}
}
func cmdWebServer(cCtx *cli.Context) error {
	b := builder.NewBuilder(cCtx)
	b.BuildConfig()
	b.BuildRepoConnPostgres()
	b.BuildProcHttp()
	return b.Run()
}
