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

	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChatCLI/utility"
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
			flag := f.Name

			// creating http client to send requests to server
			httpClient := http.Client{}

			// loading authentication token
			authToken, err := os.ReadFile("token.auth")
			if err != nil {
				log.Printf("error reading auth token: %v", err)
				return
			}

			// checking if the conversation index is valid or not
			if conversationIndex <= 0 {
				log.Print("invalid conversation index")
				return
			}

			switch strings.ToLower(flag) {
			case "new":
				// checking if the conversation index is in one to one conversation json file
				oneToOneConversationsMap := make(map[int]utility.OneToOneConversation)
				jsonData, err := os.ReadFile("one_to_one.json")
				if err != nil {
					log.Printf("error reading from one to one conversation json file")
					return
				}

				if err = json.Unmarshal(jsonData, &oneToOneConversationsMap); err != nil {
					log.Printf("error unmarshalling one to one conversation json data: %v", err)
					return
				}

				if conversationIndex-1 < len(oneToOneConversationsMap) {
					// creating new message request
					message := f.Value.String()
					requestBody, err := json.Marshal(struct {
						Description string `json:"description"`
						ReceiverID  string `json:"receiver_id"`
						GroupID     string `json:"group_id"`
					}{
						Description: message,
						ReceiverID:  oneToOneConversationsMap[conversationIndex-1].ReceiverID.String(),
						GroupID:     "",
					})
					if err != nil {
						log.Printf("error creating request body for new message request: %v", err)
						return
					}

					// creating request
					newMessageRequest, err := CreateRequest("POST", "http://localhost:8080/api/v1/message/create", requestBody)
					if err != nil {
						log.Printf("error creating new message request please try again: %v", err)
						return
					}

					// sending request
					newMessageRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
					response, err := httpClient.Do(newMessageRequest)
					if err != nil {
						log.Printf("error sending request please try again: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusCreated:
						log.Printf("message sent!")
						emptyResponse := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResponse != nil {
							if len(emptyResponse.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
									log.Printf("error writing to auth token file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Printf("server error")
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotAcceptable:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					}
				}

				// checking if conversation index is in group conversations json file
				groupConversationsMap := make(map[int]utility.GroupConversation)
				groupConversationJsonData, err := os.ReadFile("group.json")
				if err != nil {
					log.Printf("error reading group json data: %v", err)
					return
				}
				if err = json.Unmarshal(groupConversationJsonData, &groupConversationsMap); err != nil {
					log.Printf("error unmarshalling group conversation data: %v", err)
					return
				}

				if conversationIndex-1 < len(groupConversationsMap) {
					// creating new group message
					message := f.Value.String()

					// creating request body
					requestBody, err := json.Marshal(struct {
						Description string `json:"description"`
						ReceiverID  string `json:"receiver_id"`
						GroupID     string `json:"group_id"`
					}{
						Description: message,
						ReceiverID:  "",
						GroupID:     groupConversationsMap[conversationIndex-1].GroupID.UUID.String(),
					})
					if err != nil {
						log.Printf("error creating new group message request body: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("POST", "http://localhost:8080/api/v1/message/create", requestBody)
					if err != nil {
						log.Printf("error creating request: %v", err)
						return
					}

					// sending request
					request.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending request: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusCreated:
						log.Print("message sent!")
						emptyResponse := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResponse != nil {
							if len(emptyResponse.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
									log.Printf("error writing to auth token file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Printf("server error")
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotAcceptable:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					default:
						log.Print("invalid response")
					}
				}
			case "edit":
				// getting message index to edit
				messageIndexString := f.Value.String()
				messageIndex, err := strconv.Atoi(messageIndexString)
				if err != nil {
					log.Printf("error converting message index from string to integer: %v", err)
					return
				}

				// getting the new edited message from args
				editedMessage := args[0]

				if conversationIndex <= 0 {
					log.Printf("invalid conversation index")
					return
				}

				// checking if the conversation index is a valid one_to_one conversation index or group conversation index
				oneToOneConversationMap := make(map[int]utility.OneToOneConversation)
				oneToOneConversationJsonData, err := os.ReadFile("one_to_one.json")
				if err != nil {
					log.Printf("error reading one to one conversation json data: %v", err)
					return
				}
				if err = json.Unmarshal(oneToOneConversationJsonData, &oneToOneConversationMap); err != nil {
					log.Printf("error unmarshalling one to one json data: %v", err)
					return
				}

				if conversationIndex-1 < len(oneToOneConversationMap) {
					// checking if the provided message index is valid or not
					// if its valid then sending the request for editing the message
					messageJsonData, err := os.ReadFile(fmt.Sprintf("%s.json", oneToOneConversationMap[conversationIndex-1].ReceiverID.String()))
					if err != nil {
						log.Printf("error reading from messages json file: %v", err)
						return
					}
					messagesMap := make(map[int]utility.Message)
					if err = json.Unmarshal(messageJsonData, &messagesMap); err != nil {
						log.Printf("error unmarshalling messages json data: %v", err)
						return
					}
					if messageIndex <= 0 || messageIndex-1 > len(messagesMap) {
						log.Print("invalid message index")
						return
					}

					// creating request body
					requestBody, err := json.Marshal(struct {
						ID          uuid.UUID `json:"id"`
						Description string    `json:"description"`
						ReceiverID  uuid.UUID `json:"receiver_id"`
						GroupID     uuid.UUID `json:"group_id"`
					}{
						ID:          messagesMap[messageIndex-1].ID,
						Description: editedMessage,
						ReceiverID:  messagesMap[messageIndex-1].RecieverID.UUID,
						GroupID:     uuid.Nil,
					})
					if err != nil {
						log.Printf("error creating request body: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("PUT", "http://localhost:8080/api/v1/message/update", requestBody)
					if err != nil {
						log.Printf("error creating update message request: %v", err)
						return
					}

					// sending request
					request.Header.Add("authorization", string(authToken))
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending update message request: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusOK:
						log.Print("message updated")
						emptyResponse := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResponse != nil {
							if len(emptyResponse.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
									log.Printf("error writing auth token to auth file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Printf("server error")
					case http.StatusNotAcceptable:
						fallthrough
					case http.StatusBadRequest:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					}
				}

				groupConversationsMap := make(map[int]utility.GroupConversation)
				groupConversationJsonData, err := os.ReadFile("group.json")
				if err != nil {
					log.Printf("error reading group json data: %v", err)
					return
				}
				if err = json.Unmarshal(groupConversationJsonData, &groupConversationsMap); err != nil {
					log.Printf("error unmarshalling group conversation data: %v", err)
					return
				}
				if conversationIndex-1 < len(groupConversationsMap) {
					// checking if the message index is a valid group message index
					groupMessagesMap := make(map[int]utility.Message)
					groupMessageJsonData, err := os.ReadFile(fmt.Sprintf("%s.json", groupConversationsMap[conversationIndex-1].GroupID.UUID.String()))
					if err != nil {
						log.Printf("error reading from group chat json file: %v", err)
						return
					}
					if err = json.Unmarshal(groupMessageJsonData, &groupMessagesMap); err != nil {
						log.Printf("error unmarshalling group messages: %v", err)
						return
					}
					if messageIndex <= 0 || messageIndex-1 > len(groupMessagesMap) {
						log.Print("invalid message index")
						return
					}

					// creating request body
					requestBody, err := json.Marshal(struct {
						ID          uuid.UUID `json:"id"`
						Description string    `json:"description"`
						ReceiverID  uuid.UUID `json:"receiver_id"`
						GroupID     uuid.UUID `json:"group_id"`
					}{
						ID:          groupMessagesMap[messageIndex-1].ID,
						Description: editedMessage,
						ReceiverID:  uuid.Nil,
						GroupID:     groupMessagesMap[messageIndex-1].GroupID.UUID,
					})
					if err != nil {
						log.Printf("error creating group message update request body: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("PUT", "http://localhost:8080/api/v1/message/update", requestBody)
					if err != nil {
						log.Printf("error creating update message request: %v", err)
						return
					}

					// sending request
					request.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending update message request: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusOK:
						log.Print("message updated!")
						emptyResposne := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResposne != nil {
							if len(emptyResposne.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResposne.AccessToken), 0770); err != nil {
									log.Printf("error writing to auth token file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Print("server error")
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotAcceptable:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					default:
						log.Print("invalid response")
					}
				}

				log.Print("invalid conversation index")
			case "delete":
				messageIndexString := f.Value.String()
				messageIndex, err := strconv.Atoi(messageIndexString)
				if err != nil {
					log.Printf("error parsing message Index into integer: %v", err)
					return
				}
				if messageIndex <= 0 || conversationIndex <= 0 {
					log.Printf("invalid message index or conversation index")
					return
				}

				// checking if message belongs to one to one conversation
				oneToOneConversationsMap := make(map[int]utility.OneToOneConversation)
				oneToOneConversationsJsonData, err := os.ReadFile("one_to_one.json")
				if err != nil {
					log.Printf("error reading one to one coversations json file: %v", err)
					return
				}
				if err = json.Unmarshal(oneToOneConversationsJsonData, &oneToOneConversationsMap); err != nil {
					log.Printf("error unmarshalling one to one conversation json data: %v", err)
					return
				}
				if messageIndex-1 < len(oneToOneConversationsMap) {
					// reading from messages json file
					messagesMap := make(map[int]utility.Message)
					messagesJsonData, err := os.ReadFile(fmt.Sprintf("%s.json", oneToOneConversationsMap[messageIndex-1].ReceiverID.String()))
					if err != nil {
						log.Printf("error reading from messages json file: %v", err)
						return
					}
					if err = json.Unmarshal(messagesJsonData, &messagesMap); err != nil {
						log.Printf("error unmarshalling messages json data: %v", err)
						return
					}
					if messageIndex-1 > len(messagesMap) {
						log.Printf("invalid message index")
						return
					}

					// creating request body
					requestBody, err := json.Marshal(struct {
						ID      uuid.UUID `json:"id"`
						GroupID uuid.UUID `json:"group_id"`
					}{
						ID:      messagesMap[messageIndex-1].ID,
						GroupID: uuid.Nil,
					})
					if err != nil {
						log.Printf("error creating delete message request body: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("DELETE", "http://localhost:8080/api/v1/message/delete", requestBody)
					if err != nil {
						log.Printf("error creating message delete request: %v", err)
						return
					}

					// sending request
					request.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending request: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusOK:
						log.Print("message deleted!")
						emptyResponse := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResponse != nil {
							if len(emptyResponse.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
									log.Printf("error writing to auth token file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Print("server error")
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotAcceptable:
						fallthrough
					case http.StatusNotFound:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					default:
						log.Print("invalid response")
					}
				}

				// checking if message belongs to group conversations
				groupConversationsMap := make(map[int]utility.GroupConversation)
				groupConversationsJsonData, err := os.ReadFile("group.json")
				if err != nil {
					log.Printf("error reading from group coverstaions json file: %v", err)
					return
				}
				if err = json.Unmarshal(groupConversationsJsonData, &groupConversationsMap); err != nil {
					log.Printf("error unmarshalling group conversations json data: %v", err)
					return
				}
				if conversationIndex-1 < len(groupConversationsMap) {
					// checking if the messageIndex is valid or not
					messagesMap := make(map[int]utility.Message)
					messagesJsonData, err := os.ReadFile(fmt.Sprintf("%s.json", groupConversationsMap[conversationIndex-1].GroupID.UUID.String()))
					if err != nil {
						log.Printf("error reading from group chats json file: %v", err)
						return
					}
					if err = json.Unmarshal(messagesJsonData, &messagesMap); err != nil {
						log.Printf("error unmarshaling json data: %v", err)
						return
					}
					if messageIndex-1 > len(messagesMap) {
						log.Print("invalid message index")
						return
					}

					// creating request body
					requestBody, err := json.Marshal(struct {
						ID      uuid.UUID `json:"id"`
						GroupID uuid.UUID `json:"group_id"`
					}{
						ID:      messagesMap[messageIndex-1].ID,
						GroupID: messagesMap[messageIndex-1].GroupID.UUID,
					})
					if err != nil {
						log.Printf("error creating request body: %v", err)
						return
					}

					// creating request
					request, err := CreateRequest("DELETE", "http://localhost:8080/api/v1/message/delete", requestBody)
					if err != nil {
						log.Printf("error creating delete message request: %v", err)
						return
					}

					// sending request
					request.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
					response, err := httpClient.Do(request)
					if err != nil {
						log.Printf("error sending request: %v", err)
						return
					}

					// parsing response
					switch response.StatusCode {
					case http.StatusOK:
						log.Print("message deleted!")
						emptyResponse := utility.DecodeResponseBody(response.Body, &utility.EmptyResponse{}).(*utility.EmptyResponse)
						if emptyResponse != nil {
							if len(emptyResponse.AccessToken) > 0 {
								if err = os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
									log.Printf("error writing to auth token file: %v", err)
								}
							}
						}
						return
					case http.StatusInternalServerError:
						log.Print("server error")
					case http.StatusBadRequest:
						fallthrough
					case http.StatusNotAcceptable:
						fallthrough
					case http.StatusNotFound:
						errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						if errorResponse != nil {
							log.Print(errorResponse.Error)
						}
					default:
						log.Print("invalid response")
					}
				}
			}
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
