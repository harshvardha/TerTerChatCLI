/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/harshvardha/TerTerChatCLI/internal"
	"github.com/spf13/cobra"
)

// runDeamonCmd represents the runDeamon command
var runDeamonCmd = &cobra.Command{
	Use: "runDeamon",
	Run: func(cmd *cobra.Command, args []string) {
		if err := internal.StartDeamon(args[0]); err != nil {
			log.Printf("Error starting deamon process: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runDeamonCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runDeamonCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runDeamonCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
