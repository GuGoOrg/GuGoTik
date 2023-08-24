package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

var client = &http.Client{}
var commentBaseUrl = "http://127.0.0.1:37000/douyin/comment"

func TestActionComment_Add(t *testing.T) {
	url := commentBaseUrl + "/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", token)
	q.Add("actor_id", "1")
	q.Add("video_id", "0")
	q.Add("action_type", "1")
	q.Add("comment_text", "test comment")
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
	comment := &models.ActionCommentRes{}
	err = json.Unmarshal(body, &comment)
	assert.Empty(t, err)
	assert.Equal(t, 0, comment.StatusCode)
}

func TestActionComment_Delete(t *testing.T) {
	url := commentBaseUrl + "/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", token)
	q.Add("actor_id", "1")
	q.Add("video_id", "0")
	q.Add("action_type", "2")
	q.Add("comment_id", "2")
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
	actionCommentRes := &models.ActionCommentRes{}
	err = json.Unmarshal(body, &actionCommentRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, actionCommentRes.StatusCode)
}

func TestListComment(t *testing.T) {
	url := commentBaseUrl + "/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", token)
	q.Add("actor_id", "1")
	q.Add("video_id", "0")
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
	listCommentRes := &models.ListCommentRes{}
	err = json.Unmarshal(body, &listCommentRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, listCommentRes.StatusCode)
}

func TestCountComment(t *testing.T) {
	url := commentBaseUrl + "/count"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", token)
	q.Add("actor_id", "1")
	q.Add("video_id", "0")
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
	countCommentRes := &models.CountCommentRes{}
	err = json.Unmarshal(body, &countCommentRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, countCommentRes.StatusCode)
}
