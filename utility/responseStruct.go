package utility

import (
	"encoding/json"
	"io"
	"log"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func DecodeResponseBody(body io.Reader, responseStruct any) any {
	decoder := json.NewDecoder(body)
	err := decoder.Decode(responseStruct)
	if err != nil {
		log.Printf("[RESPONSE_DECODER]: Error decoding response body %v", err)
	}

	return responseStruct
}
