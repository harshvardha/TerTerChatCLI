/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var conversationIndex int

// conversationCmd represents the conversation command
var conversationCmd = &cobra.Command{
	Use:   "conversation",
	Short: "Manage specific conversation by index",
	Long: `The 'conversation' command allows you to interact with a specific
			chat conversation using its unique numerical index.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Visit(func(f *pflag.Flag) {
			// process the flags on conversation command
		})
	},
}

func init() {
	conversationCmd.AddCommand(messageCmd)
	rootCmd.AddCommand(conversationCmd)

	// adding local flags for conversation command
	conversationCmd.Flags().Bool("list", false, "provides list of all the conversation you are part of")
	conversationCmd.Flags().Int("open", -1, "input: <conversation_index>. provides all the messages of a conversation")
	conversationCmd.Flags().Int("delete", -1, "input: <conversation_index>. deletes the entire conversation")
	conversationCmd.Flags().IntVar(&conversationIndex, "index", -1, "input: <conversation_index>. this will be used along with message command and its flags")
}
