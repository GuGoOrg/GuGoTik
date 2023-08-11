package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func TestListVideos(t *testing.T) {
	//url := "http://127.0.0.1:37000/douyin/feed/?token=90aee89f-43c0-4e90-a440-cf4e47c9b790"
	url := "http://127.0.0.1:37000/douyin/feed/?latest_time=2006-01-02T15:04:05.999Z&token=90aee89f-43c0-4e90-a440-cf4e47c9b790"

	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	assert.Empty(t, err)

	res, err := client.Do(req)
	assert.Empty(t, err)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		assert.Empty(t, err)
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	assert.Empty(t, err)
	feed := &models.ListVideosRes{}
	err = json.Unmarshal(body, &feed)
	assert.Empty(t, err)
	assert.Equal(t, 0, feed.StatusCode)
}
