package utility

import (
	"encoding/json"
	"io"
	"log"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type UpdateUsernameResponse struct {
	Username    string `json:"username"`
	AccessToken string `json:"access_token"`
}

type SearchUserResponse struct {
	Username    string `json:"username"`
	CreatedAt   string `json:"created_at"`
	AccessToken string `json:"access_token"`
}

type oneToOneMessages struct {
	Sender           string
	Messages         string
	TotalNewMessages int64
}

type groupMessages struct {
	GroupName        string
	Messages         string
	TotalNewMessages int64
}

type LatestMessages struct {
	OneToOneMessages []oneToOneMessages `json:"oneToOneMessages"`
	GroupMessages    []groupMessages    `json:"groupMessages"`
	AccessToken      string             `json:"access_token"`
}

func DecodeResponseBody(body io.Reader, responseStruct any) any {
	decoder := json.NewDecoder(body)
	err := decoder.Decode(responseStruct)
	if err != nil {
		log.Printf("[RESPONSE_DECODER]: Error decoding response body %v", err)
	}

	return responseStruct
}
