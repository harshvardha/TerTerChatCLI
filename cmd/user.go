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
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/harshvardha/TerTerChatCLI/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sys/windows"
)

const (
	socketType     = "unix"
	socketFileName = "cli.sock"
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

// function to return the socket file path
// this also allows for future updations where we can
// provide user the fexibility to configure path for the
// socket file and then in other parts of codebase we can use this
// function to get the path for the socket file
func getSocketAddress() string {
	return filepath.Join(os.TempDir(), socketFileName)
}

// this function is important for checking the existence of the socket file
// if the socket file exist then deamon process is running
// otherwise we have to start a new deamon process
func isDeamonRunning() bool {
	conn, err := net.DialTimeout(socketType, getSocketAddress(), 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
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
				phonenumber := "+91" + f.Value.String()

				// asking user for password
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter password: ")
				password, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Error reading input")
					return
				}
				password = strings.TrimSuffix(password, "\r\n")

				// creating login request
				loginRequestData, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
					Password    string `json:"password"`
				}{
					Phonenumber: phonenumber,
					Password:    password,
				})
				if err != nil {
					fmt.Printf("Error creating login request: %v", err)
					return
				}
				loginRequest, err := createRequest("POST", "http://localhost:8080/api/v1/auth/login", loginRequestData)
				if err != nil {
					fmt.Println("Error creating login request")
					return
				}

				// sending request to server
				response, err := httpClient.Do(loginRequest)
				if err != nil {
					fmt.Println("Error sending request")
					return
				}
				switch response.StatusCode {
				case http.StatusNotAcceptable:
					fmt.Println("Phonenumber or password does not follow the requirements")
					return
				case http.StatusNotFound:
					fmt.Println("User not found")
					return
				case http.StatusBadRequest:
					fmt.Println("Phonenumber or password is incorrect")
					return
				case http.StatusInternalServerError:
					fmt.Println("Server error")
					return
				case http.StatusOK:
					responseData := utility.DecodeResponseBody(response.Body, &utility.LatestMessages{}).(*utility.LatestMessages)
					if responseData != nil {
						// fmt.Println(len(responseData.OneToOneMessages))
						// fmt.Println(len(responseData.GroupMessages))
						// fmt.Println(len(responseData.AccessToken))

						if len(responseData.AccessToken) > 0 {
							if err = os.WriteFile("token.auth", []byte(responseData.AccessToken), 0700); err != nil {
								fmt.Printf("Error storing authentication token: %v", err)
								return
							}
						}
					}
				default:
					responseError := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if responseError != nil {
						fmt.Println(responseError.Error)
						return
					}
				}
				response.Body.Close()

				// checking if deamon process is already running
				if isDeamonRunning() {
					fmt.Println("Already connected")
					return
				}

				// // creating a log file for deamon process to log any error
				deamonLogFile, err := os.OpenFile("deamon_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					fmt.Printf("Error opening log file: %v\n", err)
					return
				}
				defer deamonLogFile.Close()

				// // starting the deamon process
				cmd := exec.Command("E:/golang_projects/TerTerChatCLI/TerTerChatCLI.exe", "runDeamon", phonenumber)
				cmd.Stdin = deamonLogFile
				cmd.Stdout = deamonLogFile
				cmd.Stderr = deamonLogFile
				cmd.SysProcAttr = &syscall.SysProcAttr{
					HideWindow:    true,
					CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
				}
				if err = cmd.Start(); err != nil {
					fmt.Printf("Error starting deamon process: %v", err)
					return
				}
				fmt.Printf("Deamon service started with PID %d.\n", cmd.Process.Pid)
			case "disconnect":
				// initiate a socket connection to unix socket deamon process
				// and send this command to it to execute required code
				conn, err := net.Dial(socketType, getSocketAddress())
				if err != nil {
					fmt.Printf("Error establishing connection with deamon: %v", err)
					return
				}

				fmt.Println("sending command to deamon process")
				if _, err = conn.Write([]byte(f.Name + "\n")); err != nil {
					log.Printf("Error writing to deamon: %v", err)
					return
				}

				// reading the repsonse from deamon process
				fmt.Println("reading response from deamon process")
				reader := bufio.NewReader(conn)
				response, err := reader.ReadBytes('\n')
				if err != nil {
					log.Printf("Error reading response from deamon: %v", err)
					return
				}

				fmt.Println(string(response))
				conn.Close()
			case "status":
				// requesting deamon process to return the status of connection
				conn, err := net.Dial(socketType, getSocketAddress())
				if err != nil {
					log.Println(err)
					return
				}
				if _, err = conn.Write([]byte(f.Name + "\n")); err != nil {
					log.Printf("Error fetching status of connection: %v", err)
					return
				}

				// reading status of connection
				reader := bufio.NewReader(conn)
				status, err := reader.ReadBytes('\n')
				if err != nil {
					log.Printf("Error fetching connection status: %v", err)
					return
				}
				fmt.Println(string(status))
				conn.Close()
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
			case "search":
				// creating search request with the phonenumber provided
				jwtToken, err := os.ReadFile("token.auth")
				if err != nil {
					fmt.Printf("Error creating search request: %v", err)
					return
				}
				searchQuery := "+91" + f.Value.String()
				searchRequestBody, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
				}{
					Phonenumber: searchQuery,
				})
				if err != nil {
					fmt.Printf("Error creating search request: %v", err)
					return
				}
				searchRequest, err := createRequest("GET", "http://localhost:8080/api/v1/users/info", searchRequestBody)
				if err != nil {
					fmt.Printf("Error creating search request: %v", err)
					return
				}
				searchRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", jwtToken))

				// sending request to server
				response, err := httpClient.Do(searchRequest)
				if err != nil {
					fmt.Printf("Error sending request to server: %v", err)
					return
				}
				switch response.StatusCode {
				case http.StatusNotFound:
					fmt.Printf("No user found with phonenumber: %s", searchQuery)
				case http.StatusOK:
					responseData := utility.DecodeResponseBody(response.Body, &utility.SearchUserResponse{}).(*utility.SearchUserResponse)
					if responseData != nil {
						fmt.Printf("Username: %s, Joined On: %s", responseData.Username, responseData.CreatedAt)
						if len(responseData.AccessToken) > 0 {
							err = os.WriteFile("token.auth", []byte(responseData.AccessToken), 0770)
							if err != nil {
								fmt.Printf("Error updating authentication information: %v", err)
							}
						}
					}
				default:
					responseError := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if responseError != nil {
						fmt.Println(responseError.Error)
					}
				}
				response.Body.Close()
			case "remove":
				// creating request to remove current user's account
				jwtToken, err := os.ReadFile("token.auth")
				if err != nil {
					fmt.Printf("Error creating account removal request: %v", err)
					return
				}
				removeAccountRequest, err := createRequest("DELETE", "http://localhost:8080/api/v1/users/remove", nil)
				if err != nil {
					fmt.Printf("Error creating account removal request: %v", err)
					return
				}
				removeAccountRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", jwtToken))

				// sending account removal request to server
				response, err := httpClient.Do(removeAccountRequest)
				if err != nil {
					fmt.Printf("Error sending account removal request to server: %v", err)
					return
				}

				switch response.StatusCode {
				case http.StatusOK:
					fmt.Println("Account removed successfully!")
				case http.StatusNotFound:
					fmt.Println("User not found")
				}

				response.Body.Close()
			}
		})
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
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
				newUsername := f.Value.String()
				for _, value := range args {
					newUsername += " " + value
				}

				// creating http post request for updating username
				requestBodyData, err := json.Marshal(struct {
					Username string `json:"username"`
				}{
					Username: newUsername,
				})
				if err != nil {
					fmt.Printf("Error sending request: %v", err)
					return
				}

				updateUsernameRequest, err := createRequest("PUT", "http://localhost:8080/api/v1/users/update/username", requestBodyData)
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
				fmt.Println(newPassword)

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
				newPhonenumber := "+91" + f.Value.String()

				// creating otp request for new phonenumber
				otpRequestData, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
				}{
					Phonenumber: newPhonenumber,
				})
				if err != nil {
					fmt.Printf("Error creating otp request for new phonenumber: %v", err)
					return
				}
				otpRequest, err := createRequest("POST", "http://localhost:8080/api/v1/auth/otp/send", otpRequestData)
				if err != nil {
					fmt.Printf("Error creating otp request for new phonenumber: %v", err)
					return
				}

				// semding request for otp on new phonenumber
				response, err := httpClient.Do(otpRequest)
				if err != nil {
					fmt.Printf("Error sending the otp request for new phonenumber: %v", err)
					return
				}

				switch response.StatusCode {
				case http.StatusBadRequest:
					fmt.Print("Enter the otp you have already received on registered phonenumber: ")
				case http.StatusOK:
					fmt.Print("Enter the otp sent to your registered phonenumber: ")
				default:
					responseError := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if responseError != nil {
						fmt.Println(responseError.Error)
					}
				}
				response.Body.Close()

				// creating update phonenumber request
				reader := bufio.NewReader(os.Stdin)
				otp, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("Error reading input: %v", err)
					return
				}
				otp = strings.TrimSuffix(otp, "\r\n")
				updatePhonenumberRequestData, err := json.Marshal(struct {
					Phonenumber string `json:"Phonenumber"`
					OTP         string `json:"otp"`
				}{
					Phonenumber: newPhonenumber,
					OTP:         otp,
				})
				if err != nil {
					fmt.Printf("Error creating update phonenumber request: %v", err)
					return
				}
				updatePhonenumberRequest, err := createRequest("PUT", "http://localhost:8080/api/v1/users/update/phonenumber", updatePhonenumberRequestData)
				if err != nil {
					fmt.Printf("Error creating update phonenumber request: %v", err)
					return
				}
				updatePhonenumberRequest.Header.Add("authorization", fmt.Sprintf("bearer %s", jwtToken))

				// sending update phonenumber request
				response, err = httpClient.Do(updatePhonenumberRequest)
				if err != nil {
					fmt.Printf("Error sending update phonenumber request: %v", err)
					return
				}
				if response.StatusCode != http.StatusOK {
					responseError := utility.DecodeResponseBody(response.Body, &utility.ErrorResponse{}).(*utility.ErrorResponse)
					if responseError != nil {
						fmt.Println(responseError.Error)
					}
				} else {
					fmt.Println("Phonenumber updated. Please login again!")
				}

				response.Body.Close()
			}
		})
	},
}

func init() {
	userCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(userCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// userCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// userCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	userCmd.Flags().StringP("connect", "c", "", "This command will connect you to the server")
	userCmd.Flags().Bool("disconnect", false, "This command will diconnect you from the sever")
	userCmd.Flags().Bool("status", false, "This command will tell you the status of connection to server")
	userCmd.Flags().StringP("register", "r", "", "This command will help you register for service.\nIt takes username, phonenumber and password as input(space separated)")
	userCmd.Flags().StringP("search", "s", "", "This command will help you search for a user")
	userCmd.Flags().Bool("remove", false, "This command will help you delete your account")
	updateCmd.Flags().String("username", "", "This command helps you update the username")
	updateCmd.Flags().String("phonenumber", "", "This command helps you update the phonenumber")
	updateCmd.Flags().String("password", "", "This command helps you update the password")
}
