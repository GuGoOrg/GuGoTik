package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/rpc/recommend"
	"context"
)

type RecommendServiceImpl struct {
	recommend.RecommendServiceServer
}

func (a RecommendServiceImpl) New() {

}

func (a RecommendServiceImpl) GetRecommendInformation(ctx context.Context, request *recommend.RecommendRequest) (resp *recommend.RecommendResponse, err error) {
	resp = &recommend.RecommendResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		VideoList:  nil,
	}
	return
}
