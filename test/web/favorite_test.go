package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

var favoriteBaseUrl = "http://127.0.0.1:37000/douyin/favorite"

func TestActionFavorite_Do(t *testing.T) {
	url := favoriteBaseUrl + "/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "") // replace token, video_id
	q.Add("video_id", "1948195853")
	q.Add("action_type", "1")
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
	actionFavoriteRes := &models.ActionFavoriteRes{}
	err = json.Unmarshal(body, &actionFavoriteRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, actionFavoriteRes.StatusCode)
}

func TestActionFavorite_Cancel(t *testing.T) {
	url := favoriteBaseUrl + "/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "") // replace token, video_id
	q.Add("video_id", "1948195853")
	q.Add("action_type", "2")
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
	actionFavoriteRes := &models.ActionFavoriteRes{}
	err = json.Unmarshal(body, &actionFavoriteRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, actionFavoriteRes.StatusCode)
}

func TestListFavorite(t *testing.T) {
	url := favoriteBaseUrl + "/list"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "") // replace token, user_id
	q.Add("user_id", "1")
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
	listFavoriteRes := &models.ListFavoriteRes{}
	err = json.Unmarshal(body, &listFavoriteRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, listFavoriteRes.StatusCode)
}
