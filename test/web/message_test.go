package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionMessage_Add(t *testing.T) {

	var client = &http.Client{}
	var baseUrl = "http://127.0.0.1:37000/douyin/message"
	url := baseUrl + "/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "1ae83f2a-7b82-4901-9e66-50d49dba00d5")
	q.Add("actor_id", "1")
	q.Add("user_id", "1")
	q.Add("action_type", "1")
	q.Add("content", "test comment")
	req.URL.RawQuery = q.Encode()

	assert.Empty(t, err)

	res, err := client.Do(req)
	assert.Empty(t, err)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		assert.Empty(t, err)
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	assert.Empty(t, err)
	message := &models.ListMessageRes{}
	err = json.Unmarshal(body, &message)
	assert.Empty(t, err)
	assert.Equal(t, 0, message.StatusCode)
}

func TestChat(t *testing.T) {
	var client = &http.Client{}
	var baseUrl = "http://127.0.0.1:37000/douyin/message"
	url := baseUrl + "/chat"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)

	q := req.URL.Query()
	q.Add("token", "1ae83f2a-7b82-4901-9e66-50d49dba00d5")
	q.Add("actor_id", "1")
	q.Add("user_id", "1")
	q.Add("perMsgTime", "0")
	req.URL.RawQuery = q.Encode()
	assert.Empty(t, err)

	res, err := client.Do(req)
	assert.Empty(t, err)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		assert.Empty(t, err)
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	assert.Empty(t, err)
	listMessage := &models.ListMessageRes{}
	fmt.Println(listMessage)
	err = json.Unmarshal(body, &listMessage)
	assert.Empty(t, err)
	assert.Equal(t, 0, listMessage.StatusCode)
}
