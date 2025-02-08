package main

import (
	"log/slog"
	"os"

	"github.com/gouthamve/librascan/scripts/nocodb-management/nococlient"
	"github.com/spf13/cobra"
)

var (
	apiKey  string
	nocoURL string

	baseName = "LibraScan"
)

var rootCmd = &cobra.Command{
	Use:   "nocodb-management",
	Short: "A CLI tool for managing nocodb",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bases",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		if err != nil {
			slog.Error("Error creating client", "error", err)
			return
		}
		if err := client.ListBases(); err != nil {
			slog.Error("Error listing bases", "error", err)
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new base",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		if err != nil {
			slog.Error("Error creating client", "error", err)
			return
		}
		slog.Info("Creating database", "name", baseName)
		if err := client.CreateBaseIfNotExists(baseName); err != nil {
			slog.Error("Error creating base", "error", err)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing base",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		if err != nil {
			slog.Error("Error creating client", "error", err)
			return
		}
		slog.Info("Deleting base", "name", baseName)
		if err := client.DeleteBase(baseName); err != nil {
			slog.Error("Error deleting base", "error", err)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiKey, "apikey", "", "API key")
	rootCmd.PersistentFlags().StringVar(&nocoURL, "noco-url", "", "Noco base URL")
	rootCmd.MarkPersistentFlagRequired("apikey")
	rootCmd.MarkPersistentFlagRequired("noco-url")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Error executing command", "error", err)
		os.Exit(1)
	}
}
