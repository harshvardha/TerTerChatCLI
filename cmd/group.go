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

// custom type only for group command
type member struct {
	ID       uuid.UUID
	Username string
}

// function to get groups map from groups json file
func getGroupFromJsonFile(groupIndex int) *utility.GroupConversation {
	groupsMap := make(map[int]utility.GroupConversation)
	groupsJsonData, err := os.ReadFile("groups.json")
	if err != nil {
		log.Printf("error reading from groups json file: %v", err)
		return nil
	}
	if err = json.Unmarshal(groupsJsonData, &groupsMap); err != nil {
		log.Printf("error unmarshalling groups json data: %v", err)
		return nil
	}

	group := groupsMap[groupIndex]
	return &group
}

// function to get members map from members json file
func getGroupMemberFromJsonFile(groupID string, memberIndex int) *member {
	membersMap := make(map[int]member)
	membersJsonData, err := os.ReadFile(fmt.Sprintf("%s_members.json", groupID))
	if err != nil {
		log.Printf("error reading from members json file: %v", err)
		return nil
	}
	if err = json.Unmarshal(membersJsonData, &membersMap); err != nil {
		log.Printf("error unmarshalling members from json data: %v", err)
		return nil
	}
	member := membersMap[memberIndex]
	return &member
}

// groupCmd represents the group command
var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "command to perform actions related to state of group",
	Long: `This command can be used to modify the state of group such as adding a user,
	removing a user, making a user admin, etc...`,
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

			switch strings.ToLower(flag) {
			case "list":
				// reading from groups json file
				groupsMap := getGroupsMapFromJsonFile()

				// printing group names
				for index, value := range groupsMap {
					fmt.Printf("%d - %s", index, value.GroupName)
				}
			case "create":
				groupName := f.Value.String()

				// creating request body
				requestBody, err := json.Marshal(struct {
					Name string `json:"name"`
				}{
					Name: groupName,
				})
				if err != nil {
					log.Printf("error creating request body: %v", err)
					return
				}

				// creating request
				request, err := CreateRequest("POST", "http://localhost:8080/api/v1/group/create", requestBody)
				if err != nil {
					log.Printf("error creating request: %v", err)
					return
				}

				// sending request
				request.Header.Add("authorization", fmt.Sprintf("bearer %s", authToken))
				response, err := httpClient.Do(request)
				if err != nil {
					log.Printf("error sending create group request: %v", err)
					return
				}

				// parsing response
				switch response.StatusCode {
				case http.StatusCreated:
					fmt.Printf("Group Created")
					// appending new group to groups json file
					groupsMap := getGroupsMapFromJsonFile()

					// decoding response body
					type responseBody struct {
						ID          uuid.UUID `json:"id"`
						Name        string    `json:"name"`
						AccessToken string    `json:"access_token"`
					}
					newGroup := utility.DecodeResponseBody(response.Body, &responseBody{}).(*responseBody)
					if newGroup != nil {
						groupsMap[len(groupsMap)] = utility.GroupConversation{
							GroupID: uuid.NullUUID{
								UUID:  newGroup.ID,
								Valid: true,
							},
							GroupName: newGroup.Name,
						}

						// writing this new map to groups json file
						groups, err := json.MarshalIndent(groupsMap, "", " ")
						if err != nil {
							log.Printf("error marshalling groups map to json: %v", err)
							return
						}
						if err = os.WriteFile("groups.json", groups, 0770); err != nil {
							log.Printf("error writing to groups json file")
							return
						}

						// updating auth token
						if len(newGroup.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(newGroup.AccessToken), 0770); err != nil {
								log.Printf("error writing to auth file: %v", err)
							}
						}
					}
				case http.StatusInternalServerError:
					fmt.Print("server error")
				case http.StatusBadRequest:
					fallthrough
				case http.StatusNotAcceptable:
					errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if errorResponse != nil {
						log.Print(errorResponse.Error)
					}
				}
			case "update_name":
				groupIndexString := f.Value.String()
				groupIndex, err := strconv.Atoi(groupIndexString)
				if err != nil {
					log.Printf("error converting groupIndexString to Integer: %v", err)
					return
				}
				if groupIndex <= 0 {
					log.Printf("invalid group index")
					return
				}
				groupName := args[0]
				if len(groupName) == 0 {
					log.Printf("please give new group name")
					return
				}

				// fetching the group id of the given group index
				groupsMap := getGroupsMapFromJsonFile()
				group, ok := groupsMap[groupIndex]
				if !ok {
					log.Printf("invalid group index")
					return
				}

				// creating request body
				requestBody, err := json.Marshal(struct {
					GroupID uuid.UUID `json:"group_id"`
					Name    string    `json:"name"`
				}{
					GroupID: group.GroupID.UUID,
					Name:    groupName,
				})
				if err != nil {
					log.Printf("error creating request body: %s", err)
					return
				}

				// creating request
				request, err := CreateRequest("PUT", "http://localhost:8080/api/v1/group/update", requestBody)
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

				// parsing response body
				switch response.StatusCode {
				case http.StatusOK:
					fmt.Print("Group Name Updated!")

					// defining response struct
					type responseBody struct {
						Name        string `json:"name"`
						AccessToken string `json:"access_token"`
					}

					// decoding response body and updating group name in groups json file
					group := utility.DecodeResponseBody(response.Body, &responseBody{}).(*responseBody)
					if group != nil {
						existingGroup := groupsMap[groupIndex-1]
						existingGroup.GroupName = group.Name
						groupsMap[groupIndex-1] = existingGroup

						// writing this updated group to groups json file
						updatedGroups, err := json.MarshalIndent(existingGroup, "", " ")
						if err != nil {
							log.Printf("error marshalling updated group to json: %v", err)
							return
						}
						if err = os.WriteFile("groups.json", updatedGroups, 0770); err != nil {
							log.Printf("error writing to groups json file: %v", err)
							return
						}

						// updating auth token
						if len(group.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(group.AccessToken), 0770); err != nil {
								log.Printf("error writing to auth file: %v", err)
								return
							}
						}
					}
				case http.StatusInternalServerError:
					fmt.Print("server error")
				case http.StatusUnauthorized:
					fallthrough
				case http.StatusNotAcceptable:
					errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if errorResponse != nil {
						fmt.Print(errorResponse.Error)
					}
				}
			case "members":
				groupIndexString := f.Value.String()
				groupIndex, err := strconv.Atoi(groupIndexString)
				if err != nil {
					log.Printf("error casting groupIndexString to Integer: %v", err)
					return
				}
				groupsMap := getGroupsMapFromJsonFile()

				// creating request body
				requestBody, err := json.Marshal(struct {
					GroupID uuid.UUID `json:"group_id"`
				}{
					GroupID: groupsMap[groupIndex-1].GroupID.UUID,
				})
				if err != nil {
					log.Printf("error creating request body: %v", err)
					return
				}

				// creating request
				request, err := CreateRequest("GET", "http://localhost:8080/api/v1/group/members", requestBody)
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
				case http.StatusOK:

					type responseBody struct {
						Members     []member `json:"members"`
						AccessToken string   `json:"access_token"`
					}
					groupMembers := utility.DecodeResponseBody(response.Body, &responseBody{}).(*responseBody)
					if groupMembers != nil {
						// printing group members and saving them into a json file with naming pattern "<group_id>_members.json"
						membersMap := make(map[int]member)
						for index, value := range groupMembers.Members {
							fmt.Printf("%d - %s", index+1, value.Username)
							membersMap[index] = value

							// marshalling membersMap into json
							membersMapJson, err := json.Marshal(membersMap)
							if err != nil {
								log.Printf("error marshalling members map: %v", err)
								return
							}

							// writing json to a file
							if err = os.WriteFile(fmt.Sprintf("%s_members.json", groupsMap[groupIndex-1].GroupID.UUID.String()), membersMapJson, 0770); err != nil {
								log.Printf("error writing to group members json file: %v", err)
								return
							}
						}

						// updating auth token
						if len(groupMembers.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(groupMembers.AccessToken), 0770); err != nil {
								log.Printf("error updating auth file: %v", err)
							}
						}
					}
				case http.StatusUnauthorized:
					fallthrough
				case http.StatusNotAcceptable:
					errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if errorResponse != nil {
						fmt.Print(errorResponse.Error)
					}
				}
			case "remove":
				groupIndexString := f.Value.String()
				groupIndex, err := strconv.Atoi(groupIndexString)
				if err != nil {
					log.Printf("error converting group index string to integer: %v", err)
					return
				}
				if groupIndex <= 0 {
					fmt.Print("invalid group index")
					return
				}

				groupMemberIndexString := args[0]
				groupMemberIndex, err := strconv.Atoi(groupMemberIndexString)
				if err != nil {
					log.Printf("error converting group member index string to integer: %v", err)
					return
				}
				if groupMemberIndex <= 0 {
					fmt.Print("invalid group member index")
					return
				}

				// fetching group id and member id for the give group and member index
				groupsID := getGroupIDFromJsonFile()

			}
		})
	},
}

func init() {
	rootCmd.AddCommand(groupCmd)

	rootCmd.Flags().Bool("list", true, "this flag lists all the groups you are part of")
	rootCmd.Flags().String("create", "", "this flag helps in creating new group. input: <group_name>")
	rootCmd.Flags().Int("update_name", -1, "this flag helps in updating group name. input: <group_index> <new_name>")
	rootCmd.Flags().Int("members", -1, "this flag lists all the members of the group. input: <group_index>")
	rootCmd.Flags().Int("remove", -1, "this flag removes the member from the group. input: <group_index> <member_index>")
	rootCmd.Flags().Int("leave", -1, "this flag helps you to leave the group. input: <group_index>")
	rootCmd.Flags().Int("add", -1, "this flag adds a user to group. input: <group_index> <new_member_phonenumber")
	rootCmd.Flags().Int("make_admin", -1, "this flag makes a group member admin. input: <group_index> <member_index>")
	rootCmd.Flags().Int("remove_admin", -1, "this flag removes a group member from admin. input: <group_index> <member_index")
	rootCmd.Flags().Int("admins", -1, "this flag lists all the admins of the group. input: <group_index>")
}
