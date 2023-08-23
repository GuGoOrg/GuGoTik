package utils

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"net/http"
)

type CustomJSON struct {
	Data    proto.Message
	Context *gin.Context
}

var m = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseProtoNames:   true,
}

func (r CustomJSON) Render(w http.ResponseWriter) (err error) {
	r.WriteContentType(w)
	res, _ := m.Marshal(r.Data)
	_, err = w.Write(res)
	return
}

func (r CustomJSON) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}
