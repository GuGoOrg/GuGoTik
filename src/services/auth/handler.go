package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/rpc/recommend"
	"GuGoTik/src/rpc/relation"
	user2 "GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/cached"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/willf/bloom"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	stringsLib "strings"
	"sync"
)

var relationClient relation.RelationServiceClient
var userClient user2.UserServiceClient
var recommendClient recommend.RecommendServiceClient

var BloomFilter *bloom.BloomFilter

type AuthServiceImpl struct {
	auth.AuthServiceServer
}

func (a AuthServiceImpl) New() {
	relationConn := grpc2.Connect(config.RelationRpcServerName)
	relationClient = relation.NewRelationServiceClient(relationConn)
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	userClient = user2.NewUserServiceClient(userRpcConn)
	recommendRpcConn := grpc2.Connect(config.RecommendRpcServiceName)
	recommendClient = recommend.NewRecommendServiceClient(recommendRpcConn)
}

func (a AuthServiceImpl) Authenticate(ctx context.Context, request *auth.AuthenticateRequest) (resp *auth.AuthenticateResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "AuthenticateService")
	defer span.End()
	logging.SetSpanWithHostname(span)
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
	logging.SetSpanWithHostname(span)
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
		user.UserName = request.Username
		if user.IsNameEmail() {
			logger.WithFields(logrus.Fields{
				"mail": request.Username,
			}).Infof("Trying to get the user avatar")
			user.Avatar = getAvatarByEmail(ctx, request.Username)
		} else {
			logger.WithFields(logrus.Fields{
				"mail": request.Username,
			}).Infof("Username is not the email, using default logic to fetch avatar")
			user.Avatar = fmt.Sprintf("https://api.multiavatar.com/%s.png", url.QueryEscape(request.Username))
		}
	}()

	wg.Wait()

	user.BackgroundImage = "https://i.mij.rip/2023/08/26/0caa1681f9ae3de38f7d8abcc3b849fc.jpeg"
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

	recommendResp, err := recommendClient.RegisterRecommendUser(ctx, &recommend.RecommendRegisterRequest{UserId: user.ID, Username: request.Username})
	if err != nil || recommendResp.StatusCode != strings.ServiceOKCode {
		resp = &auth.RegisterResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	resp.UserId = user.ID
	resp.StatusCode = strings.ServiceOKCode
	resp.StatusMsg = strings.ServiceOK

	// Publish the username to redis
	BloomFilter.AddString(user.UserName)
	logger.WithFields(logrus.Fields{
		"username": user.UserName,
	}).Infof("Publishing user name to redis channel")
	err = redis.Client.Publish(ctx, config.BloomRedisChannel, user.UserName).Err()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":      err,
			"username": user.UserName,
		}).Errorf("Publishing user name to redis channel happens error")
		logging.SetSpanError(span, err)
	}

	addMagicUserFriend(ctx, &span, user.ID)

	return
}

func (a AuthServiceImpl) Login(ctx context.Context, request *auth.LoginRequest) (resp *auth.LoginResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "LoginService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("AuthService.Login").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"username": request.Username,
	}).Debugf("User try to log in.")

	// Check if a username might be in the filter
	if !BloomFilter.TestString(request.Username) {
		resp = &auth.LoginResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}

		logger.WithFields(logrus.Fields{
			"username": request.Username,
		}).Infof("The user is blocked by Bloom Filter")
		return
	}

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
		logging.SetSpanError(span, err)
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
			logging.SetSpanError(span, err)
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
			logging.SetSpanError(span, err)
			return
		}

		if err = setUserInfoToRedis(ctx, user.UserName, hashed); err != nil {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			logging.SetSpanError(span, err)
			return
		}

		cached.Write(ctx, fmt.Sprintf("UserId%s", request.Username), strconv.Itoa(int(user.ID)), true)
	} else {
		id, _, err := cached.Get(ctx, fmt.Sprintf("UserId%s", request.Username))
		if err != nil {
			resp = &auth.LoginResponse{
				StatusCode: strings.AuthServiceInnerErrorCode,
				StatusMsg:  strings.AuthServiceInnerError,
			}
			logging.SetSpanError(span, err)
			return nil, err
		}
		uintId, _ := strconv.ParseUint(id, 10, 32)
		user.ID = uint32(uintId)
	}

	token, err := getToken(ctx, user.ID)

	if err != nil {
		resp = &auth.LoginResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	logger.WithFields(logrus.Fields{
		"token":  token,
		"userId": user.ID,
	}).Debugf("User log in sucess !")
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
	logging.SetSpanWithHostname(span)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func checkPasswordHash(ctx context.Context, password, hash string) bool {
	_, span := tracing.Tracer.Start(ctx, "PasswordHashChecked")
	defer span.End()
	logging.SetSpanWithHostname(span)
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getToken(ctx context.Context, userId uint32) (string, error) {
	span := trace.SpanFromContext(ctx)
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("AuthService.Login").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"userId": userId,
	}).Debugf("Select for user token")
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
	logging.SetSpanWithHostname(span)
	lowerEmail := stringsLib.ToLower(email)
	hashed := md5.New()
	hashed.Write([]byte(lowerEmail))
	md5Bytes := hashed.Sum(nil)
	md5String = hex.EncodeToString(md5Bytes)
	return
}

func addMagicUserFriend(ctx context.Context, span *trace.Span, userId uint32) {
	logger := logging.LogService("AuthService.Register.AddMagicUserFriend").WithContext(ctx)

	isMagicUserExist, err := userClient.GetUserExistInformation(ctx, &user2.UserExistRequest{
		UserId: config.EnvCfg.MagicUserId,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"UserId": userId,
			"Err":    err,
		}).Errorf("Failed to check if the magic user exists")
		logging.SetSpanError(*span, err)
		return
	}

	if !isMagicUserExist.Existed {
		logger.WithFields(logrus.Fields{
			"UserId": userId,
		}).Errorf("Magic user does not exist")
		logging.SetSpanError(*span, errors.New("magic user does not exist"))
		return
	}

	// User follow magic user
	_, err = relationClient.Follow(ctx, &relation.RelationActionRequest{
		ActorId: userId,
		UserId:  config.EnvCfg.MagicUserId,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"UserId": userId,
			"Err":    err,
		}).Errorf("Failed to follow magic user")
		logging.SetSpanError(*span, err)
		return
	}

	// Magic user follow user
	_, err = relationClient.Follow(ctx, &relation.RelationActionRequest{
		ActorId: config.EnvCfg.MagicUserId,
		UserId:  userId,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"UserId": userId,
			"Err":    err,
		}).Errorf("Magic user failed to follow user")
		logging.SetSpanError(*span, err)
		return
	}
}
