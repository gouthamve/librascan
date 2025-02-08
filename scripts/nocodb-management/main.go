package main

import (
	"log/slog"
	"os"

	"github.com/gouthamve/librascan/nococlient"
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
		bases, err := client.ListBases()
		if err != nil {
			slog.Error("Error listing bases", "error", err)
			return
		}

		slog.Info("Bases loaded", "count", len(bases.List))
		for _, base := range bases.List {
			slog.Info("Base", "name", base.Title, "id", base.ID)
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

var tableCmd = &cobra.Command{
	Use:   "table",
	Short: "Manage tables within a base",
}

var tableListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tables in the base",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		if err != nil {
			slog.Error("Error creating client", "error", err)
			return
		}

		tables, err := client.ListTables(baseName)
		if err != nil {
			slog.Error("Error listing tables", "error", err)
			return
		}

		slog.Info("Tables loaded", "count", len(tables.List))
		for _, table := range tables.List {
			slog.Info("Table", "name", table.TableName, "id", table.ID)
		}
	},
}

var tableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new table",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		// if err != nil {
		// 	slog.Error("Error creating client", "error", err)
		// 	return
		// }
		// // TODO: call client.CreateTable(args[0]) when implemented
		// slog.Info("Creating table (not implemented)", "table", args[0])
	},
}

var tableDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing table",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// client, err := nococlient.NewNocoClient(nocoURL, apiKey)
		// if err != nil {
		// 	slog.Error("Error creating client", "error", err)
		// 	return
		// }
		// // TODO: call client.DeleteTable(args[0]) when implemented
		// slog.Info("Deleting table (not implemented)", "table", args[0])
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

	// Add table commands
	tableCmd.AddCommand(tableListCmd, tableCreateCmd, tableDeleteCmd)
	rootCmd.AddCommand(tableCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Error executing command", "error", err)
		os.Exit(1)
	}
}
