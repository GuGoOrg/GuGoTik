package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func TestListVideo(t *testing.T) {
	url := "http://127.0.0.1:37000/douyin/publish/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
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
	ListPublishRes := &models.ListPublishRes{}
	err = json.Unmarshal(body, &ListPublishRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, ListPublishRes.StatusCode)
}
