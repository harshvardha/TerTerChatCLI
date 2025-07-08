/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/harshvardha/TerTerChatCLI/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// userCmd represents the user command
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "A brief description of user command",
	Long: `This command is used to execute user related actions such as
	connect, disconnect, register, update.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("user command called")

		// checking which flags were set by the user
		cmd.Flags().Visit(func(f *pflag.Flag) {
			name := f.Name
			switch strings.ToLower(name) {
			case "connect":
				fmt.Println("connecting to server")
			case "disconnect":
				fmt.Println("disconnecting")
			case "register":
				// asking for username, phonenumber and password for registering the user
				reader := bufio.NewReader(os.Stdin)

				fmt.Print("Enter Username: ")
				username, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("invalid username")
					return
				}

				fmt.Print("Enter Phonenumber: ")
				phonenumber, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("invalid phonenumber")
					return
				}

				fmt.Print("Enter Password: ")
				password, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("invalid password")
					return
				}

				// requesting the server to send OTP to given phonenumber
				otpRequest, err := json.Marshal(struct {
					Phonenumber string `json:"phonenumber"`
				}{
					Phonenumber: phonenumber,
				})
				if err != nil {
					fmt.Printf("Error sending request: %v", err)
					return
				}

				// creating http post request
				request, err := http.NewRequest("POST", "http://localhost:8080/api/v1/auth/otp/send", bytes.NewBuffer(otpRequest))
				if err != nil {
					fmt.Printf("Error creating request: %v", err)
					return
				}

				// setting request header
				request.Header.Set("Content-Type", "application/json")

				// creating http client
				httpClient := &http.Client{}

				// sending the request
				response, err := httpClient.Do(request)
				if err != nil {
					fmt.Printf("Error sending the request: %v", err)
					return
				}
				response.Body.Close()

				// checking response status
				// if response is 200 then asking user to enter the otp recieved and sending the registration request
				if response.StatusCode == http.StatusOK {
					fmt.Print("Enter OTP send to your phonenumber: ")
					otp, err := reader.ReadString('\n')
					if err != nil {
						fmt.Printf("Error reading otp input: %v", err)
						return
					}

					// marshalling the registration information and sending registration request to server
					registrationInformation, err := json.Marshal(struct {
						Username    string `json:"username"`
						Phonenumber string `json:"phonenumber"`
						Password    string `json:"password"`
						OTP         string `json:"otp"`
					}{
						Username:    username,
						Phonenumber: phonenumber,
						Password:    password,
						OTP:         otp,
					})
					if err != nil {
						fmt.Printf("Error marshalling registration information: %v", err)
						return
					}

					// sending registration request to server
					request.URL.Path = "/api/v1/user/register"
					request.Body = io.NopCloser(bytes.NewBuffer(registrationInformation))

					response, err = httpClient.Do(request)
					if err != nil {
						fmt.Printf("Error sending registration request: %v", err)
						return
					}
					response.Body.Close()

					// checking response status
					if response.StatusCode == http.StatusCreated {
						fmt.Println("Registration Successful")
						utility.ClearConsole()
					}
				}
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(userCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// userCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// userCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	userCmd.Flags().StringP("connect", "c", "", "This command will connect you to the server")
	userCmd.Flags().StringP("disconnect", "d", "", "This command will diconnect you from the sever")
	userCmd.Flags().StringP("register", "r", "", "This command will help you register for service")
	userCmd.Flags().StringP("search", "s", "", "This command will help you search for a user")
	userCmd.Flags().Bool("remove", false, "This command will help you delete your account")
	userCmd.Flags().StringP("update", "u", "", "This command helps you to update your credentials but you have to provide what you want to update like username, password, phonenumber")
	userCmd.Flags().String("username", "", "This command helps you update the username")
	userCmd.Flags().String("phonenumber", "", "This command helps you update the phonenumber")
	userCmd.Flags().String("password", "", "This command helps you update the password")
}
