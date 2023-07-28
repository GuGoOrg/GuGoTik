package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/google/uuid"
	redisLib "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"sync"
	"time"
)

type AuthServiceImpl struct {
	auth.AuthServiceServer
}

func (a AuthServiceImpl) Authenticate(ctx context.Context, request *auth.AuthenticateRequest) (resp *auth.AuthenticateResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (a AuthServiceImpl) Register(ctx context.Context, request *auth.RegisterRequest) (resp *auth.RegisterResponse, err error) {
	resp = &auth.RegisterResponse{}
	var user models.User
	result := database.DB.WithContext(ctx).Limit(1).Find(&user)
	if result.RowsAffected != 0 {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthUserExistedCode,
			StatusMsg:  strings.AuthUserExisted,
			UserId:     0,
			Token:      "",
		}
		return
	}

	var hashedPassword string
	if hashedPassword, err = hashPassword(request.Password); err != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
			UserId:     0,
			Token:      "",
		}
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	//Get Sign
	go func() {
		defer wg.Done()
		resp, err := http.Get("https://v1.hitokoto.cn/?c=b&encode=text")
		log := logging.LogMethod("Auth.FetchSignature")
		if err != nil {
			user.Signature = user.UserName
			log.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not reach hitokoto")
			return
		}

		if resp.StatusCode != http.StatusOK {
			user.Signature = user.UserName
			log.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Warnf("Hitokoto service may be error")
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			user.Signature = user.UserName
			log.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not decode the response body of hitokoto")
			return
		}

		user.Signature = string(body)
	}()

	wg.Wait()

	user.UserName = request.Username
	user.Password = hashedPassword

	// We should ensure it can get token, then we can write new user data into database
	if resp.Token, err = getToken(ctx, user.UserName); err != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
			UserId:     0,
			Token:      "",
		}
		return
	}
	result = database.DB.WithContext(ctx).Create(user)
	if result.Error != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
			UserId:     0,
			Token:      "",
		}
		return
	}
	resp.UserId = uint32(user.ID)
	resp.StatusCode = strings.ServiceOKCode
	resp.StatusMsg = strings.ServiceOK
	return
}

func (a AuthServiceImpl) Login(ctx context.Context, request *auth.LoginRequest) (resp *auth.LoginResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getToken(ctx context.Context, username string) (token string, err error) {
	token, err = redis.Client.Get(ctx, username).Result()
	switch {
	case err == redisLib.Nil: // User do not log in
		token = uuid.New().String()
		redis.Client.Set(ctx, username, token, 240*time.Hour)
		return
	default:
		return
	}
}
