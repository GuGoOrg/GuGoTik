package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	redisLib "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"strconv"
	strings2 "strings"
	"sync"
	"time"
)

type AuthServiceImpl struct {
	auth.AuthServiceServer
}

func (a AuthServiceImpl) Authenticate(ctx context.Context, request *auth.AuthenticateRequest) (resp *auth.AuthenticateResponse, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "AuthenticateService")
	defer span.Finish()

	has, userId, err := hasToken(ctx, request.Token)
	if err != nil {
		resp = &auth.AuthenticateResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	if !has {
		resp = &auth.AuthenticateResponse{
			StatusCode: strings.AuthUserNotExistedCode,
			StatusMsg:  strings.AuthUserNotExisted,
		}
		return
	}

	id, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		resp = &auth.AuthenticateResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	resp = &auth.AuthenticateResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserId:     uint32(id),
	}

	return
}

func (a AuthServiceImpl) Register(ctx context.Context, request *auth.RegisterRequest) (resp *auth.RegisterResponse, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RegisterService")
	defer span.Finish()

	resp = &auth.RegisterResponse{}
	var user models.User
	result := database.Client.WithContext(ctx).Limit(1).Where("user_name = ?", request.Username).Find(&user)
	if result.RowsAffected != 0 {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthUserExistedCode,
			StatusMsg:  strings.AuthUserExisted,
		}
		return
	}

	var hashedPassword string
	if hashedPassword, err = hashPassword(ctx, request.Password); err != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	//Get Sign
	go func() {
		defer wg.Done()
		resp, err := http.Get("https://v1.hitokoto.cn/?c=b&encode=text")
		logger := logging.GetSpanLogger(span, "Auth.FetchSignature")
		if err != nil {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not reach hitokoto")
			return
		}

		if resp.StatusCode != http.StatusOK {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Warnf("Hitokoto service may be error")
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not decode the response body of hitokoto")
			return
		}

		user.Signature = string(body)
	}()

	go func() {
		defer wg.Done()
		if user.IsNameEmail() {
			user.Avatar = getAvatarByEmail(ctx, user.UserName)
		}
	}()

	wg.Wait()

	user.UserName = request.Username
	user.Password = hashedPassword

	result = database.Client.WithContext(ctx).Create(&user)
	if result.Error != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	if resp.Token, err = getToken(ctx, user.ID); err != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return resp, nil
	}

	resp.UserId = uint32(user.ID)
	resp.StatusCode = strings.ServiceOKCode
	resp.StatusMsg = strings.ServiceOK
	return
}

func (a AuthServiceImpl) Login(ctx context.Context, request *auth.LoginRequest) (resp *auth.LoginResponse, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "LoginService")
	defer span.Finish()
	logger := logging.GetSpanLogger(span, "AuthService.Login")
	logger.WithFields(logrus.Fields{
		"username": request.Username,
	}).Infof("User try to log in.")

	resp = &auth.LoginResponse{}
	user := models.User{
		UserName: request.Username,
	}

	if !isUserVerifiedInRedis(ctx, request.Username, request.Password) {
		result := database.Client.Where("user_name = ?", request.Username).Find(&user)
		if result.Error != nil {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			return
		}

		if result.RowsAffected == 0 {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthUserNotExistedCode,
				StatusMsg:  strings.AuthUserNotExisted,
			}
			return
		}

		if !checkPasswordHash(ctx, request.Password, user.Password) {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthUserLoginFailedCode,
				StatusMsg:  strings.AuthUserLoginFailed,
			}
			return
		}

		hashed, errs := hashPassword(ctx, request.Password)
		if errs != nil {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			return
		}
		setUserInfoToRedis(ctx, user.UserName, hashed)
	}

	token, err := getToken(ctx, user.ID)
	if err != nil {
		resp = &auth.LoginResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return resp, nil
	}

	resp = &auth.LoginResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserId:     uint32(user.ID),
		Token:      token,
	}
	return
}

func hashPassword(ctx context.Context, password string) (string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Auth-PasswordHash")
	defer span.Finish()
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func checkPasswordHash(ctx context.Context, password, hash string) bool {
	span, _ := opentracing.StartSpanFromContext(ctx, "Auth-PasswordHashChecked")
	defer span.Finish()
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getToken(ctx context.Context, userId uint) (token string, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Redis-GetToken")
	defer span.Finish()
	token, err = redis.Client.Get(ctx, "U2T"+strconv.Itoa(int(userId))).Result()
	span.LogFields(log.String("token", token))
	switch {
	case err == redisLib.Nil: // User do not log in
		token = uuid.New().String()
		redis.Client.Set(ctx, "U2T"+strconv.Itoa(int(userId)), token, 240*time.Hour)
		redis.Client.Set(ctx, "T2U"+token, userId, 240*time.Hour)
		return token, nil
	default:
		return
	}
}

func hasToken(ctx context.Context, token string) (bool, string, error) {
	userId, err := redis.Client.Get(ctx, "T2U"+token).Result()
	switch {
	case err == redisLib.Nil: // User do not log in
		return false, "", nil
	case err != nil:
		return false, "", err
	default:
		return true, userId, nil
	}
}

func isUserVerifiedInRedis(ctx context.Context, username string, password string) bool {
	span, _ := opentracing.StartSpanFromContext(ctx, "Redis-VerifiedLogUserInfo")
	defer span.Finish()
	saved, err := redis.Client.Get(ctx, "UserLog"+username).Result()
	switch {
	case err == redisLib.Nil: // User do not log in
		return false
	case err != nil:
		return false
	default:
		if checkPasswordHash(ctx, password, saved) {
			return true
		}
		return false
	}
}

func setUserInfoToRedis(ctx context.Context, username string, password string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Redis-SetUserLog")
	defer span.Finish()
	saved, err := redis.Client.Get(ctx, "UserLog"+username).Result()
	switch {
	case err == redisLib.Nil:
		redis.Client.Set(ctx, "UserLog"+username, password, 240*time.Hour)
	case err != nil:
	default:
		redis.Client.Del(ctx, "UserLog"+saved)
		redis.Client.Set(ctx, "UserLog"+username, password, 240*time.Hour)
	}
}

func getAvatarByEmail(ctx context.Context, email string) string {
	span, _ := opentracing.StartSpanFromContext(ctx, "Auth-GetAvatar")
	defer span.Finish()
	return fmt.Sprintf("https://cravatar.cn/avatar/%s?d=identicon", getEmailMD5(ctx, email))
}

func getEmailMD5(ctx context.Context, email string) (md5String string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Auth-EmailMD5")
	defer span.Finish()
	lowerEmail := strings2.ToLower(email)
	hashed := md5.New()
	hashed.Write([]byte(lowerEmail))
	md5Bytes := hashed.Sum(nil)
	md5String = hex.EncodeToString(md5Bytes)
	return
}
