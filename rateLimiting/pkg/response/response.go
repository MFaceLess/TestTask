package response

import (
	"encoding/json"
	"io"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ResponseJSON(w io.Writer, code int, message string) {
	errResponse := &Response{
		Code:    code,
		Message: message,
	}

	resp, err := json.Marshal(errResponse)
	if err != nil {
		return
	}

	if _, err = w.Write(resp); err != nil {
		return
	}
}
