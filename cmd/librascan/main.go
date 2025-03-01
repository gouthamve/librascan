package main

import (
	"log"

	"github.com/spf13/cobra"

	_ "github.com/gouthamve/librascan/migrations"
	_ "modernc.org/sqlite"

	"github.com/gouthamve/librascan/pkg/readIsbn"
	"github.com/gouthamve/librascan/pkg/tui"
)

var database = "./.db/librascan.db"

func main() {
	rootCmd := &cobra.Command{
		Use:   "librascan",
		Short: "A book lookup server and ISBN CLI tool",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			apiKey, err := cmd.Flags().GetString("perplexity-key")
			if err != nil {
				log.Fatalln("cannot get perplexity-key flag:", err)
			}
			serve(apiKey)
		},
	}
	serveCmd.Flags().String("perplexity-key", "", "The perplexity API key.")

	// Add a flag option for server URL in the read-isbn command.
	waitCmd := &cobra.Command{
		Use:   "read-isbn",
		Short: "Start ISBN input loop",
		Run: func(cmd *cobra.Command, args []string) {
			serverURL, err := cmd.Flags().GetString("server-url")
			if err != nil {
				log.Fatalln("cannot get server URL:", err)
			}
			inputDevicePath, err := cmd.Flags().GetString("input-device-path")
			if err != nil {
				log.Fatalln("cannot get inputPath flag:", err)
			}
			readIsbn.StartCLI(serverURL, inputDevicePath)
		},
	}
	waitCmd.Flags().String("server-url", "http://localhost:8080", "Server URL for posting ISBNs.")
	waitCmd.Flags().String("input-device-path", "", "Path to the scanners udev device.")

	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the TUI interface",
		Run: func(cmd *cobra.Command, args []string) {
			serverURL, err := cmd.Flags().GetString("server-url")
			if err != nil {
				log.Fatalln("cannot get server URL:", err)
			}

			tui.Start(serverURL)
		},
	}
	tuiCmd.Flags().String("server-url", "http://localhost:8080", "Server URL for posting ISBNs.")

	rootCmd.AddCommand(tuiCmd)

	rootCmd.AddCommand(serveCmd, waitCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
