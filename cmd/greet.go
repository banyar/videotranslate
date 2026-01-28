package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var name string

var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Greet a user",
	Long:  "Print a greeting message. Use --name to specify who to greet.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Hello, %s!\n", name)
	},
}

func init() {
	greetCmd.Flags().StringVarP(&name, "name", "n", "World", "name of the person to greet")
	rootCmd.AddCommand(greetCmd)
}
