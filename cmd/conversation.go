/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/harshvardha/TerTerChatCLI/utility"
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
			flag := f.Name

			// creating a http client to send request to server
			httpClient := http.Client{}

			// loading authentication token
			authToken, err := os.ReadFile("token.auth")
			if err != nil {
				log.Printf("error reading auth token: %v", err)
				return
			}
			switch flag {
			case "list":
				// list all the  conversations user is involved in
				request, err := CreateRequest("GET", "http://localhost:8080/api/v1/message/conversations", nil)
				if err != nil {
					log.Printf("error creating request for fetching all conversations: %v", err)
					return
				}
				request.Header.Add("authorization", fmt.Sprintf("bearer %s", string(authToken)))

				// sending request to server
				response, err := httpClient.Do(request)
				if err != nil {
					log.Printf("error sending request to server for fetching all coversations: %v", err)
					return
				}

				if response.StatusCode == http.StatusOK {
					conversations := utility.DecodeResponseBody(response.Body, &utility.Conversations{}).(*utility.Conversations)
					if conversations != nil {
						if len(conversations.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(conversations.AccessToken), 0770); err != nil {
								log.Printf("error updating auth token: %v", err)
							}

							var offset uint // offset will track the converstaion number which can be used as index by user to do other operations

							// checking of one to one conversations file which stores receivers id exist or not
							if _, err = os.Stat("one_to_one.conv"); err != nil {
								if _, err = os.Create("one_to_one.conv"); err != nil {
									log.Printf("error creating one to one conversation file")
									return
								}
							}

							// checking if group conversation file exist which stores the group id
							if _, err = os.Stat("group.conv"); err != nil {
								if _, err = os.Create("group.conv"); err != nil {
									log.Printf("error creating group conversation file")
									return
								}
							}

							for _, value := range conversations.OneToOneConversations {
								// printing the name of the receiver with index
								// index is the position of the receiver id in one_to_one conversation file
								fmt.Printf("%d - %s", offset+1, value.Username)

								// writing the receiver id to one to one conversation file
								if err = os.WriteFile("one_to_one.conv", []byte(value.ReceiverID.String()+"\n"), 0770); err != nil {
									log.Printf("error writing to one to one conversation file: %v", err)
									return
								}

								offset++
							}

							// printing the name of group with index
							// and storing the group id in group conversation file
							// index is the position of the group id in group conversation file
							for _, value := range conversations.GroupConversations {
								fmt.Printf("%d - %s", offset+1, value.GroupName)

								// writing the group id to group conversation file
								if err = os.WriteFile("group.conv", []byte(value.GroupID.UUID.String()+"\n"), 0770); err != nil {
									log.Printf("error writing to group conversation file: %v", err)
									return
								}

								offset++
							}
						}
					}
				}
				response.Body.Close()
			case "open":

			case "delete":
				// delete a specific conversation
			}
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
