/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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
				// user will provide the index of the conversation they want to open
				// first we will find out whether that index exist in one_to_one conversation or group conversation
				// if it exist in one_to_one conversation then use api url: http://localhost:8080/api/v1/message/conversation to get the messages
				// if it exist in group conversation then use api url: http://localhost:8080/api/v1/message/group/all to get the messages
				stringIndex := strings.TrimSuffix(f.Value.String(), "\r\n")
				index, err := strconv.Atoi(stringIndex)
				if err != nil {
					log.Printf("error reading index: %v", err)
					return
				}

				// counting the number of receiver ids in one_to_one.conv
				oneToOneFileContents, err := os.ReadFile("one_to_one.conv")
				if err != nil {
					log.Printf("error fetching conversation: %v", err)
					return
				}
				oneToOneConversationsString := string(oneToOneFileContents)
				oneToOneConversations := strings.Split(oneToOneConversationsString, "\n")

				// counting the number of group ids in group.conv
				groupFileContents, err := os.ReadFile("group.conv")
				if err != nil {
					log.Printf("error fetching conversation: %v", err)
					return
				}
				groupConversationsString := string(groupFileContents)
				groupConversations := strings.Split(groupConversationsString, "\n")

				// checking if receiver id exist in one_to_one or group conversation file
				if index-1 < len(oneToOneConversations) {
					receiverId := uuid.MustParse(oneToOneConversations[index-1])
					requestBody, err := json.Marshal(struct {
						ReceiverID uuid.NullUUID `json:"receiver_id"`
						CreatedAt  time.Time     `json:"created_at"`
					}{
						ReceiverID: uuid.NullUUID{
							UUID:  receiverId,
							Valid: true,
						},
						CreatedAt: time.Now(),
					})
					if err != nil {
						log.Printf("error creating request for fetching messages of one to one conversation: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("GET", "http://localhost:8080/api/v1/message/conversation", requestBody)
					if err != nil {
						log.Printf("error creating request for fetching messages of conversation: %v", err)
						return
					}

					// sending request to server
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending request to server for fetching messages of conversation: %v", err)
						return
					}
					switch response.StatusCode {
					case http.StatusOK:
						// print all the messages
						messages := utility.DecodeResponseBody(response.Body, &utility.ConversationMessages{}).(*utility.ConversationMessages)
						if messages != nil {
							for _, message := range messages.Messages {
								if message.SenderID == receiverId {
									fmt.Printf("%s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
								} else if message.RecieverID.UUID == receiverId {
									fmt.Printf("You: %s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
								}
							}
						}
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotFound:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							fmt.Println(errorResponse.Error)
						}
					default:
						fmt.Println("Server error")
					}
				} else if index-1 < len(groupConversations) {
					requestBody, err := json.Marshal(struct {
						GroupID uuid.UUID `json:"group_id"`
						Before  time.Time `json:"before"`
					}{
						GroupID: uuid.MustParse(groupConversations[index-1]),
						Before:  time.Now(),
					})
					if err != nil {
						log.Printf("error creating request for fetching messages of group conversation: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("GET", "http://localhost:8080/api/v1/message/group/all", requestBody)
					if err != nil {
						log.Printf("error creating request to fetch messages of conversation: %v", err)
						return
					}

					// sending request
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending request to fetch messages of conversation: %v", err)
						return
					}

					switch response.StatusCode {
					case http.StatusOK:
						// print all group messages
						messages := utility.DecodeResponseBody(response.Body, &utility.ConversationMessages{}).(*utility.ConversationMessages)
						if messages != nil {
							for _, message := range messages.Messages {
								fmt.Printf("%s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
							}
						}
					case http.StatusNotAcceptable:
						fallthrough
					case http.StatusBadRequest:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							fmt.Println(errorResponse.Error)
						}
					default:
						fmt.Println("Server error")
					}
				} else {
					log.Println("invalid index")
				}
			case "delete":

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
