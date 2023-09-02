package web

import (
	"GuGoTik/src/web/models"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestListVideo(t *testing.T) {
	url := "http://127.0.0.1:37000/douyin/publish/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", token)
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
	ListPublishRes := &models.ListPublishRes{}
	err = json.Unmarshal(body, &ListPublishRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, ListPublishRes.StatusCode)
}

func TestPublishVideo(t *testing.T) {
	url := "http://127.0.0.1:37000/douyin/publish/action"
	method := "POST"
	filePath := "E:\\Administrator\\Videos\\1.mp4"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	file, err := os.Open(filePath)
	assert.Empty(t, err)
	defer func(file *os.File) {
		err := file.Close()
		assert.Empty(t, err)
	}(file)

	fileWriter, err := writer.CreateFormFile("data", file.Name())
	assert.Empty(t, err)

	_, err = io.Copy(fileWriter, file)
	assert.Empty(t, err)

	_ = writer.WriteField("token", token)
	_ = writer.WriteField("title", "10个报错，但是我代码只有9行啊？？？")

	err = writer.Close()
	assert.Empty(t, err)

	req, err := http.NewRequest(method, url, payload)
	assert.Empty(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	assert.Empty(t, err)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		assert.Empty(t, err)
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	assert.Empty(t, err)
	actionPublishRes := &models.ActionPublishRes{}
	err = json.Unmarshal(body, &actionPublishRes)
	assert.Empty(t, err)
	assert.Equal(t, 0, actionPublishRes.StatusCode)
}
