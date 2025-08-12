/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// messageCmd represents the message command
var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Manage specific message by index",
	Long: `The 'message' command allows you to interact with a specific
			message using its unique numerical index.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Visit(func(f *pflag.Flag) {
			// process the flags for message command
		})
	},
}

func init() {
	rootCmd.AddCommand(messageCmd)

	// adding local flags to message command
	messageCmd.Flags().String("new", "", "input: <new_message>")
	messageCmd.Flags().Int("edit", -1, "input: <message_index> <edited_message>")
	messageCmd.Flags().Int("delete", -1, "input: <message_index>")
}
