package web

import (
	"GuGoTik/src/web/models"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func TestRelationAction(t *testing.T) {

	//url := "http://127.0.0.1:37000/douyin/relation/reg?username=" + uuid.New().String() + "&password=epicmo"
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/action"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "e65153ea-15b6-4959-9462-f9fb5c5d59ce")
	q.Add("user_id", "1")
	q.Add("actor_id", "4")
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
	relation := &models.RelationActionRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)
}

func TestUnFollow(t *testing.T) {

	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/unfollow"
	method := "POST"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
	q.Add("user_id", "2")
	q.Add("actor_id", "0")
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
	relation := &models.RelationActionRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)

}

func TestGetFollowList(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/follow/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
	q.Add("user_id", "1")
	q.Add("actor_id", "1")
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
	relation := &models.FollowListRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)

}

func TestGetFollowerList(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/follower/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
	q.Add("user_id", "1")
	q.Add("actor_id", "1")
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
	relation := &models.FollowerListRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)

}

func TestCountFollowList(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/follow/count"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
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
	relation := &models.CountFollowListRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)

}

func TestCountFollowerList(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/follower/count"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
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
	relation := &models.CountFollowerListRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)

}

func TestGetFriendList(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/friend/list"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
	q.Add("actor_id", "3")
	q.Add("user_id", "3")
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
	relation := &models.FriendListRes{}
	fmt.Println(relation)
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, 0, relation.StatusCode)
}

func TestIsFollow(t *testing.T) {
	client := &http.Client{}
	url := "http://127.0.0.1:37000/douyin/relation/isFollow"
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	q := req.URL.Query()
	q.Add("token", "93b9e0bf-ebd3-4d35-801d-ac9076a1d6e5")
	q.Add("user_id", "2")
	q.Add("actor_id", "1")
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
	relation := &models.IsFollowRes{}
	err = json.Unmarshal(body, &relation)
	assert.Empty(t, err)
	assert.Equal(t, false, relation.Result)

}
