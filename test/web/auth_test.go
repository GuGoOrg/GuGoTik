package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func TestRegister(t *testing.T) {

	url := "http://127.0.0.1:37000/douyin/user/register?username=" + uuid.New().String() + "&password=epicmo"
	method := "POST"
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
	user := &models.LoginRes{}
	err = json.Unmarshal(body, &user)
	assert.Empty(t, err)
	assert.Equal(t, 0, user.StatusCode)
}

func TestDisplayRegister(t *testing.T) {

	url := "http://127.0.0.1:37000/douyin/user/register?username=epicmo&password=epicmo"
	method := "POST"
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
	user := &models.LoginRes{}
	err = json.Unmarshal(body, &user)
	assert.Empty(t, err)
	assert.Equal(t, 0, user.StatusCode)
}

func TestLogin(t *testing.T) {

	url := "http://127.0.0.1:37000/douyin/user/login?username=epicmo&password=epicmo"
	method := "POST"
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
	user := &models.LoginRes{}
	err = json.Unmarshal(body, &user)
	assert.Empty(t, err)
	assert.Equal(t, 0, user.StatusCode)
}
