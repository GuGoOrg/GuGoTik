package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/storage/cached"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/logging"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"strconv"
	stringsLib "strings"
	"sync"
)

type AuthServiceImpl struct {
	auth.AuthServiceServer
}

func (a AuthServiceImpl) Authenticate(ctx context.Context, request *auth.AuthenticateRequest) (resp *auth.AuthenticateResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "AuthenticateService")
	defer span.End()
	logger := logging.LogService("AuthService.Authenticate").WithContext(ctx)

	userId, ok, err := hasToken(ctx, request.Token)

	if err != nil {
		resp = &auth.AuthenticateResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	if !ok {
		resp = &auth.AuthenticateResponse{
			StatusCode: strings.UserNotExistedCode,
			StatusMsg:  strings.UserNotExisted,
		}
		return
	}

	id, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":   err,
			"token": request.Token,
		}).Warnf("AuthService Authenticate Action failed to response when parsering uint")
		logging.SetSpanError(span, err)

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
	ctx, span := tracing.Tracer.Start(ctx, "RegisterService")
	defer span.End()
	logger := logging.LogService("AuthService.Register").WithContext(ctx)

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
		logger.WithFields(logrus.Fields{
			"err":      result.Error,
			"username": request.Username,
		}).Warnf("AuthService Register Action failed to response when hashing password")
		logging.SetSpanError(span, err)

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
		_, span := tracing.Tracer.Start(ctx, "FetchSignature")
		defer span.End()
		logger := logging.LogService("AuthService.FetchSignature").WithContext(ctx)

		if err != nil {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not reach hitokoto")
			logging.SetSpanError(span, err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Warnf("Hitokoto service may be error")
			logging.SetSpanError(span, err)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			user.Signature = user.UserName
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Warnf("Can not decode the response body of hitokoto")
			logging.SetSpanError(span, err)
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
		logger.WithFields(logrus.Fields{
			"err":      result.Error,
			"username": request.Username,
		}).Warnf("AuthService Register Action failed to response when creating user")
		logging.SetSpanError(span, result.Error)

		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	resp.Token, err = getToken(ctx, user.ID)

	if err != nil {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	logger.WithFields(logrus.Fields{
		"username": request.Username,
	}).Infof("User register success!")

	resp.UserId = user.ID
	resp.StatusCode = strings.ServiceOKCode
	resp.StatusMsg = strings.ServiceOK
	return
}

func (a AuthServiceImpl) Login(ctx context.Context, request *auth.LoginRequest) (resp *auth.LoginResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "LoginService")
	defer span.End()
	logger := logging.LogService("AuthService.Login").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"username": request.Username,
	}).Infof("User try to log in.")

	resp = &auth.LoginResponse{}
	user := models.User{
		UserName: request.Username,
	}

	ok, err := isUserVerifiedInRedis(ctx, request.Username, request.Password)
	if err != nil {
		resp = &auth.LoginResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	if !ok {
		result := database.Client.Where("user_name = ?", request.Username).WithContext(ctx).Find(&user)
		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"err":      result.Error,
				"username": request.Username,
			}).Warnf("AuthService Login Action failed to response with inner err.")
			logging.SetSpanError(span, result.Error)

			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			return
		}

		if result.RowsAffected == 0 {
			resp = &auth.LoginResponse{
				StatusCode: strings.UserNotExistedCode,
				StatusMsg:  strings.UserNotExisted,
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
			logger.WithFields(logrus.Fields{
				"err":      errs,
				"username": request.Username,
			}).Warnf("AuthService Login Action failed to response with inner err.")
			logging.SetSpanError(span, errs)

			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			return
		}

		if err = setUserInfoToRedis(ctx, user.UserName, hashed); err != nil {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			return
		}
	}

	token, err := getToken(ctx, user.ID)

	if err != nil {
		resp = &auth.LoginResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	resp = &auth.LoginResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserId:     user.ID,
		Token:      token,
	}
	return
}

func hashPassword(ctx context.Context, password string) (string, error) {
	_, span := tracing.Tracer.Start(ctx, "PasswordHash")
	defer span.End()

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func checkPasswordHash(ctx context.Context, password, hash string) bool {
	_, span := tracing.Tracer.Start(ctx, "PasswordHashChecked")
	defer span.End()
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getToken(ctx context.Context, userId uint32) (string, error) {
	return cached.GetWithFunc(ctx, "U2T"+strconv.FormatUint(uint64(userId), 10),
		func(ctx context.Context, key string) (string, error) {
			span := trace.SpanFromContext(ctx)
			token := uuid.New().String()
			span.SetAttributes(attribute.String("token", token))
			cached.Write(ctx, "T2U"+token, strconv.FormatUint(uint64(userId), 10), true)
			return token, nil
		})
}

func hasToken(ctx context.Context, token string) (string, bool, error) {
	return cached.Get(ctx, "T2U"+token)
}

func isUserVerifiedInRedis(ctx context.Context, username string, password string) (bool, error) {
	pass, ok, err := cached.Get(ctx, "UserLog"+username)

	if err != nil {
		return false, nil
	}

	if !ok {
		return false, nil
	}

	if checkPasswordHash(ctx, password, pass) {
		return true, nil
	}

	return false, nil
}

func setUserInfoToRedis(ctx context.Context, username string, password string) error {
	_, ok, err := cached.Get(ctx, "UserLog"+username)

	if err != nil {
		return err
	}

	if ok {
		cached.TagDelete(ctx, "UserLog"+username)
	}
	cached.Write(ctx, "UserLog"+username, password, true)
	return nil
}

func getAvatarByEmail(ctx context.Context, email string) string {
	ctx, span := tracing.Tracer.Start(ctx, "Auth-GetAvatar")
	defer span.End()
	return fmt.Sprintf("https://cravatar.cn/avatar/%s?d=identicon", getEmailMD5(ctx, email))
}

func getEmailMD5(ctx context.Context, email string) (md5String string) {
	_, span := tracing.Tracer.Start(ctx, "Auth-EmailMD5")
	defer span.End()

	lowerEmail := stringsLib.ToLower(email)
	hashed := md5.New()
	hashed.Write([]byte(lowerEmail))
	md5Bytes := hashed.Sum(nil)
	md5String = hex.EncodeToString(md5Bytes)
	return
}
