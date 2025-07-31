/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/harshvardha/TerTerChatCLI/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// function to send http request
func createRequest(verb string, url string, body []byte) (*http.Request, error) {
	request, err := http.NewRequest(strings.ToUpper(verb), url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

// userCmd represents the user command
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "A brief description of user command",
	Long: `This command is used to execute user related actions such as
	connect, disconnect, register, update.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("user command called")

		// http client to send request to server
		httpClient := &http.Client{}

		// checking which flags were set by the user
		cmd.Flags().Visit(func(f *pflag.Flag) {
			name := f.Name
			switch strings.ToLower(name) {
			case "connect":
				fmt.Println("connecting to server")
			case "disconnect":
				fmt.Println("disconnecting")
			case "register":
				phonenumber := f.Value

				// requesting the server to send OTP to given phonenumber
				otpRequestData, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
				}{
					Phonenumber: phonenumber.String(),
				})
				if err != nil {
					fmt.Printf("Error sending request: %v", err)
					return
				}

				// creating http post request for sending otp to user phonenumber
				otpRequest, err := createRequest("POST", "http://localhost:8080/api/v1/auth/otp/send", otpRequestData)
				if err != nil {
					fmt.Printf("Error creating request: %v", err)
					return
				}

				// sending the request
				response, err := httpClient.Do(otpRequest)
				if err != nil {
					fmt.Printf("Error sending the request: %v", err)
					return
				}
				response.Body.Close()

				// checking if we can use the same otp
				if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusOK {
					// asking for username, password and OTP for registering the user
					reader := bufio.NewReader(os.Stdin)

					fmt.Print("Enter Username: ")
					username, err := reader.ReadString('\n')
					if err != nil {
						fmt.Printf("invalid username")
						return
					}
					username = strings.TrimSuffix(username, "\r\n")
					fmt.Println(username[0])
					fmt.Println(username[len(username)-1])

					fmt.Print("Enter Password: ")
					password, err := reader.ReadString('\n')
					if err != nil {
						fmt.Printf("invalid password")
						return
					}
					password = strings.TrimSuffix(password, "\r\n")

					fmt.Print("Enter OTP send to your phonenumber: ")
					otp, err := reader.ReadString('\n')
					if err != nil {
						fmt.Printf("Error reading otp input: %v", err)
						return
					}
					otp = strings.TrimSuffix(otp, "\r\n")

					// marshalling the registration information and sending registration request to server
					registrationInformation, err := json.Marshal(struct {
						Username    string `json:"username"`
						Phonenumber string `json:"phonenumber"`
						Password    string `json:"password"`
						OTP         string `json:"otp"`
					}{
						Username:    username,
						Phonenumber: phonenumber.String(),
						Password:    password,
						OTP:         otp,
					})
					if err != nil {
						fmt.Printf("Error marshalling registration information: %v", err)
						return
					}

					// creating registration request
					registrationRequest, err := createRequest("POST", "http://localhost:8080/api/v1/auth/register", registrationInformation)
					if err != nil {
						log.Printf("Error creating registration request: %v", err)
						return
					}

					response, err = httpClient.Do(registrationRequest)
					if err != nil {
						fmt.Printf("Error sending registration request: %v", err)
						return
					}

					// checking response status
					if response.StatusCode == http.StatusCreated {
						fmt.Println("Registration Successful")
					} else if response.StatusCode > 399 && response.StatusCode < 500 {
						log.Printf("Error %d", response.StatusCode)
					} else {
						fmt.Println(response.StatusCode)
						params := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
						log.Printf("Error: %s", params.Error)
						response.Body.Close()
					}
				}
			}
		})
	},
}

var updateCmd = &cobra.Command{
	Use:   "User credentials update",
	Short: "Subcommand used to update user crendentials such as phonenumber, password, username",
	Long:  "This command helps you to update your credentials but you have to provide what you want to update like username, password, phonenumber",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Visit(func(f *pflag.Flag) {
			command := f.Name
			httpClient := &http.Client{}

			// fetching the jwt token for authentication
			jwtToken, err := os.ReadFile("token.auth")
			if err != nil {
				fmt.Printf("Error creating request: %v", err)
				return
			}

			switch strings.ToLower(command) {
			case "username":
				newUsername := f.Value

				// creating http post request for updating username
				requestBodyData, err := json.Marshal(struct {
					Username string `json:"username"`
				}{
					Username: newUsername.String(),
				})
				if err != nil {
					fmt.Printf("Error sending request: %v", err)
					return
				}

				updateUsernameRequest, err := createRequest("POST", "http://localhost:8080/api/v1/user/update/username", requestBodyData)
				if err != nil {
					fmt.Printf("Error creating a update username request: %v", err)
					return
				}
				updateUsernameRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", jwtToken))

				// sending request
				response, err := httpClient.Do(updateUsernameRequest)
				if err != nil {
					fmt.Printf("Error sending update username request: %v", err)
					return
				}

				if response.StatusCode == http.StatusOK {
					responseData := utility.DecodeResponseBody(response.Body, &utility.UpdateUsernameResponse{}).(*utility.UpdateUsernameResponse)
					if responseData == nil {
						fmt.Printf("Error decoding response")
						return
					}

					// checking if new accessToken is provided
					// if provided then replace the current with the new token
					if len(responseData.AccessToken) > 0 {
						err = os.WriteFile("token.auth", []byte(responseData.AccessToken), 0700)
						if err != nil {
							fmt.Printf("Error updating auth credentials: %v", err)
							return
						}
					}
					fmt.Printf("Updated username: %v", responseData.Username)
				}
			case "password":
				// send request to update password
				newPassword := f.Value.String()

				// making a otp request on the registered phonenumber
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter the registered phonenumber: ")
				registeredPhonenumber, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("Error reading input: %v", err)
					return
				}

				registeredPhonenumber = "+91" + strings.TrimSuffix(registeredPhonenumber, "\r\n")
				otpRequestData, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
				}{
					Phonenumber: registeredPhonenumber,
				})
				if err != nil {
					fmt.Printf("Error creating otp request: %v", err)
					return
				}

				otpRequest, err := createRequest("POST", "http://localhost:8080/api/v1/auth/send/otp", otpRequestData)
				if err != nil {
					fmt.Printf("Error creating request: %v", err)
					return
				}

				// sending the request to server
				response, err := httpClient.Do(otpRequest)
				if err != nil {
					fmt.Printf("Error sending the otp request while updating password: %v", err)
					return
				}
				switch response.StatusCode {
				case http.StatusBadRequest:
					fmt.Print("Enter the otp you have already received on registered phonenumber: ")
				case http.StatusOK:
					fmt.Print("Enter the otp sent to your registered phonenumber: ")
				}
				response.Body.Close()

				// creating update password request
				otp, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("Error reading input: %v", err)
					return
				}
				otp = strings.TrimSuffix(otp, "\r\n")
				updatePasswordRequestBody, err := json.Marshal(struct {
					Password    string `json:"password"`
					Phonenumber string `json:"phonenumber"`
					OTP         string `json:"otp"`
				}{
					Password:    newPassword,
					Phonenumber: registeredPhonenumber,
					OTP:         otp,
				})
				if err != nil {
					fmt.Printf("Error creating update password request: %v", err)
					return
				}
				updatePasswordRequest, err := createRequest("PUT", "http://localhost:8080/api/v1/users/update/password", updatePasswordRequestBody)
				if err != nil {
					fmt.Printf("Error creating update password request: %v", err)
					return
				}
				updatePasswordRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", jwtToken))

				// sending the update password request
				response, err = httpClient.Do(updatePasswordRequest)
				if err != nil {
					fmt.Printf("Error sending the update password request: %v", err)
					return
				}
				if response.StatusCode != http.StatusOK {
					errorResponse := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					fmt.Println(errorResponse.Error)
				} else {
					fmt.Println("Update password successfull. Please login again!")
				}
				response.Body.Close()
			case "phonenumber":
				// send request to update phonenumber
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// userCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// userCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	userCmd.Flags().StringP("connect", "c", "", "This command will connect you to the server")
	userCmd.Flags().StringP("disconnect", "d", "", "This command will diconnect you from the sever")
	userCmd.Flags().StringP("register", "r", "", "This command will help you register for service.\nIt takes username, phonenumber and password as input(space separated)")
	userCmd.Flags().StringP("search", "s", "", "This command will help you search for a user")
	userCmd.Flags().Bool("remove", false, "This command will help you delete your account")
	updateCmd.Flags().String("username", "", "This command helps you update the username")
	updateCmd.Flags().String("phonenumber", "", "This command helps you update the phonenumber")
	updateCmd.Flags().String("password", "", "This command helps you update the password")
}
