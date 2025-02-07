package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

struct nocodbClient {
	APIKey string
}

var rootCmd = &cobra.Command{
	Use:   "nocodb-management",
	Short: "A CLI tool for managing undb",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases",
	Run: func(cmd *cobra.Command, args []string) {
		
		// Add your list implementation here
	},
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Creating database: %s\n", args[0])
		// Add your create implementation here
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
