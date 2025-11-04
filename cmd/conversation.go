/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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

// this function provides oneToOneConversations Map
func getOneToOneConversationMap() map[int]utility.OneToOneConversation {
	oneToOneConversationsMap := make(map[int]utility.OneToOneConversation)
	jsonData, err := os.ReadFile("one_to_one.json")
	if err != nil {
		log.Printf("error reading from one to one conversation json file")
		return nil
	}

	if err = json.Unmarshal(jsonData, &oneToOneConversationsMap); err != nil {
		log.Printf("error unmarshalling one to one conversation json data: %v", err)
		return nil
	}

	return oneToOneConversationsMap
}

// this function will update the auth file when recieved empty response
func updateAuthFileForEmptyResponse(responseBody io.Reader) {
	emptyResponse := utility.DecodeResponseBody(responseBody, &utility.EmptyResponse{}).(*utility.EmptyResponse)
	if emptyResponse != nil {
		if len(emptyResponse.AccessToken) > 0 {
			if err := os.WriteFile("token.auth", []byte(emptyResponse.AccessToken), 0770); err != nil {
				log.Printf("error updating auth file: %v", err)
			}
		}
	}
}

// this function provided messages map from json file
func getMessagesMap(fileName string) map[int]utility.Message {
	messagesMap := make(map[int]utility.Message)
	messagesJsonData, err := os.ReadFile(fmt.Sprintf("%s.json", fileName))
	if err != nil {
		log.Printf("error reading from messages file: %v", err)
		return nil
	}
	if err = json.Unmarshal(messagesJsonData, &messagesMap); err != nil {
		log.Printf("error unmarshalling messages json data: %v", err)
		return nil
	}

	return messagesMap
}

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
							if _, err = os.Stat("one_to_one.json"); err != nil {
								if _, err = os.Create("one_to_one.json"); err != nil {
									log.Printf("error creating one to one conversation file")
									return
								}
							}

							// checking if group conversation file exist which stores the group id
							if _, err = os.Stat("group.json"); err != nil {
								if _, err = os.Create("group.json"); err != nil {
									log.Printf("error creating group conversation file")
									return
								}
							}

							// creating a one_to_one conversations map which we will marshal to json and write it to one_to_one conversation json file
							oneToOneConversations := make(map[uint]utility.OneToOneConversation)
							for _, value := range conversations.OneToOneConversations {
								// printing the name of the receiver with index
								// index is the key of the receiver id in one_to_one conversation json file
								fmt.Printf("%d - %s", offset+1, value.Username)

								// writing the conversation to the map
								oneToOneConversations[offset] = value
								offset++
							}

							// writing the one_to_one conversations to its json file
							jsonData, err := json.MarshalIndent(oneToOneConversations, "", " \n")
							if err != nil {
								log.Printf("error marshalling the one to one convesation map to json data")
								return
							}

							if err = os.WriteFile("one_to_one.json", jsonData, 0770); err != nil {
								log.Printf("error writing to one to one conversation json file: %v", err)
								return
							}

							// creating a group conversation map which we will marshal to json and write it to group conversation json file
							groupConversationMap := make(map[uint]utility.GroupConversation)
							for _, value := range conversations.GroupConversations {
								// printing the name of group with index
								// index is the position of the group id in group conversation json file
								fmt.Printf("%d - %s", offset+1, value.GroupName)

								// writing the group conversation to the map
								groupConversationMap[offset] = value
								offset++
							}

							// writing group conversations to its json file
							jsonData, err = json.MarshalIndent(groupConversationMap, "", " \n")
							if err != nil {
								log.Printf("error marshalling group conversation map to json file: %v", err)
								return
							}

							if err = os.WriteFile("group.json", jsonData, 0770); err != nil {
								log.Printf("error writing to group conversations json file: %v", err)
								return
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
				oneToOneConversationsMap := getOneToOneConversationMap()

				// counting the number of group ids in group.conv
				groupConversationsMap := getGroupsMapFromJsonFile()

				// checking if receiver id exist in one_to_one or group conversation file
				if index-1 < len(oneToOneConversationsMap) {
					receiverId := uuid.MustParse(oneToOneConversationsMap[index-1].ReceiverID.String())
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
							// checking if the messages file exist or not
							// if not then we will create the messages file with naming pattern as '<receiverID>.json'
							if _, err = os.Stat(fmt.Sprintf("%s.json", receiverId.String())); err != nil {
								if _, err = os.Create(fmt.Sprintf("%s.json", receiverId.String())); err != nil {
									log.Printf("error creating messages file: %v", err)
									return
								}
							}

							messagesMap := make(map[int]utility.Message)
							for index, message := range messages.Messages {
								messagesMap[index] = message
								if message.SenderID == receiverId {
									fmt.Printf("%s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
								} else if message.RecieverID.UUID == receiverId {
									fmt.Printf("You: %s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
								}
							}

							// writing messages map to messages json file
							jsonData, err := json.MarshalIndent(messagesMap, "", " ")
							if err != nil {
								log.Printf("error marshalling messages json data: %v", err)
								return
							}

							if err = os.WriteFile(fmt.Sprintf("%s.json", receiverId.String()), jsonData, 0770); err != nil {
								log.Printf("error writing messages to json file: %v", err)
								return
							}
						}
						if len(messages.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(messages.AccessToken), 0770); err != nil {
								log.Printf("error writing auth token to file: %v", err)
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
				} else if index-1 < len(groupConversationsMap) {
					requestBody, err := json.Marshal(struct {
						GroupID uuid.UUID `json:"group_id"`
						Before  time.Time `json:"before"`
					}{
						GroupID: uuid.MustParse(groupConversationsMap[index-1].GroupID.UUID.String()),
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
							// storing messages for the group chats with this naming pattern: <groupID>.json
							// first checking if the group chats json file exist or not
							// if not then create the file
							if _, err := os.Stat(fmt.Sprintf("%s.json", groupConversationsMap[index-1].GroupID.UUID.String())); err != nil {
								if _, err := os.Create(fmt.Sprintf("%s.json", groupConversationsMap[index-1].GroupID.UUID.String())); err != nil {
									log.Printf("error creating group chats json file: %v", err)
									return
								}
							}

							groupChatsMap := make(map[int]utility.Message)
							for index, message := range messages.Messages {
								groupChatsMap[index] = message
								fmt.Printf("%s, %s", message.Description, message.CreatedAt.Format(time.RFC1123))
							}

							// writing the group chats map into a json file
							jsonData, err := json.MarshalIndent(groupChatsMap, "", " ")
							if err != nil {
								log.Printf("error marshalling group chats map: %v", err)
								return
							}

							if err = os.WriteFile(fmt.Sprintf("%s.json", groupConversationsMap[index-1].GroupID.UUID.String()), jsonData, 0770); err != nil {
								log.Printf("error writing to group chats json file: %v", err)
								return
							}
						}
						if len(messages.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(messages.AccessToken), 0770); err != nil {
								log.Printf("error writing auth token to file: %v", err)
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
					response.Body.Close()
				} else {
					log.Println("invalid index")
				}
			case "delete":
				// user will provide the index of the conversation they want to delete
				// then we will first check if the index of conversation exist in one_to_one conversation
				// if it exist in one_to_one conversation then we will hit the endpoint http://localhost:8080/api/v1/message/conversation/delete
				// to delete the conversation between this user and the other user involved
				stringIndex := strings.TrimSuffix(f.Value.String(), "\r\n")
				index, err := strconv.Atoi(stringIndex)
				if err != nil {
					log.Printf("error converting string index to integer in delete command of conversation: %v", err)
					return
				}

				// checking if this index exist in one_to_one conversation json file
				oneToOneConversationsMap := getOneToOneConversationMap()

				// checking if the key exist in the map
				value, ok := oneToOneConversationsMap[index-1]
				if !ok {
					log.Printf("Invalid Index")
					return
				}

				// creating request body for delete conversation request
				requestBody, err := json.Marshal(struct {
					ReceiverID uuid.NullUUID `json:"reciever_id"`
				}{
					ReceiverID: uuid.NullUUID{
						UUID:  value.ReceiverID,
						Valid: true,
					},
				})
				if err != nil {
					log.Printf("error creating request body for delete conversation request: %v", err)
					return
				}

				// creating delete conversation request
				request, err := CreateRequest("DELETE", "http://localhost:8080/api/v1/message/conversation/delete", requestBody)
				if err != nil {
					log.Printf("error creating delete conversation request: %v", err)
					return
				}

				// sending delete one_to_one conversation request
				response, err := httpClient.Do(request)
				if err != nil {
					log.Printf("error sending one_to_one conversation delete request: %v", err)
					return
				}

				// processing response based on status codes
				switch response.StatusCode {
				case http.StatusOK:
					updateAuthFileForEmptyResponse(response.Body)
				case http.StatusBadRequest:
					errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if errorResponse != nil {
						log.Print(errorResponse.Error)
					}
				case http.StatusInternalServerError:
					log.Print("server error")
				}
			}
		})
	},
}

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
				oneToOneConversationsMap := getOneToOneConversationMap()

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
						updateAuthFileForEmptyResponse(response.Body)
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
				groupConversationsMap := getGroupsMapFromJsonFile()

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
						updateAuthFileForEmptyResponse(response.Body)
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
				oneToOneConversationMap := getOneToOneConversationMap()
				if conversationIndex-1 < len(oneToOneConversationMap) {
					// checking if the provided message index is valid or not
					// if its valid then sending the request for editing the message
					messagesMap := getMessagesMap(oneToOneConversationMap[conversationIndex-1].ReceiverID.String())
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
						updateAuthFileForEmptyResponse(response.Body)
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

				groupConversationsMap := getGroupsMapFromJsonFile()
				if conversationIndex-1 < len(groupConversationsMap) {
					// checking if the message index is a valid group message index
					messagesMap := getMessagesMap(groupConversationsMap[conversationIndex-1].GroupID.UUID.String())
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
						ReceiverID:  uuid.Nil,
						GroupID:     messagesMap[messageIndex-1].GroupID.UUID,
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
						updateAuthFileForEmptyResponse(response.Body)
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
				oneToOneConversationsMap := getOneToOneConversationMap()
				if messageIndex-1 < len(oneToOneConversationsMap) {
					// reading from messages json file
					messagesMap := getMessagesMap(oneToOneConversationsMap[conversationIndex-1].ReceiverID.String())
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
						updateAuthFileForEmptyResponse(response.Body)
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
				groupConversationsMap := getGroupsMapFromJsonFile()
				if conversationIndex-1 < len(groupConversationsMap) {
					// checking if the messageIndex is valid or not
					messagesMap := getMessagesMap(groupConversationsMap[conversationIndex-1].GroupID.UUID.String())
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
						updateAuthFileForEmptyResponse(response.Body)
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
	conversationCmd.AddCommand(messageCmd)
	rootCmd.AddCommand(conversationCmd)

	// adding local flags for conversation command
	conversationCmd.Flags().Bool("list", false, "provides list of all the conversation you are part of")
	conversationCmd.Flags().Int("open", -1, "input: <conversation_index>. provides all the messages of a conversation")
	conversationCmd.Flags().Int("delete", -1, "input: <conversation_index>. deletes the entire conversation")
	conversationCmd.Flags().IntVar(&conversationIndex, "index", -1, "input: <conversation_index>. this will be used along with message command and its flags")

	// adding local flags to message command
	messageCmd.Flags().String("new", "", "input: <new_message>")
	messageCmd.Flags().Int("edit", -1, "input: <message_index> <edited_message>")
	messageCmd.Flags().Int("delete", -1, "input: <message_index>")
}
