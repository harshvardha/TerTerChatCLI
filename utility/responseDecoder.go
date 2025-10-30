package utility

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/google/uuid"
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

type OneToOneConversation struct {
	ReceiverID uuid.UUID
	Username   string
}

type GroupConversation struct {
	GroupID   uuid.NullUUID
	GroupName string
}

type message struct {
	ID          uuid.UUID
	Description string
	SenderID    uuid.UUID
	RecieverID  uuid.NullUUID
	GroupID     uuid.NullUUID
	Sent        bool
	Recieved    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Read        bool
}

// response body decoder struct for user --login command
type LatestMessages struct {
	OneToOneMessages []oneToOneMessages `json:"oneToOneMessages"`
	GroupMessages    []groupMessages    `json:"groupMessages"`
	AccessToken      string             `json:"access_token"`
}

// response body decoder struct for conversation --list command
type Conversations struct {
	OneToOneConversations []OneToOneConversation `json:"one_to_one_conversations"`
	GroupConversations    []GroupConversation    `json:"group_conversations"`
	AccessToken           string                 `json:"access_token"`
}

// response body decoder struct for conversation --open command
type ConversationMessages struct {
	Messages    []message `json:"messages"`
	AccessToken string    `json:"access_token"`
}

func DecodeResponseBody(body io.Reader, responseStruct any) any {
	decoder := json.NewDecoder(body)
	err := decoder.Decode(responseStruct)
	if err != nil {
		log.Printf("[RESPONSE_DECODER]: Error decoding response body %v", err)
	}

	return responseStruct
}
