package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

func GetChildSpanFromGinContext(c *gin.Context, name string) opentracing.Span {
	pSp := opentracing.SpanFromContext(c.Request.Context())
	return pSp.Tracer().StartSpan(
		name,
		opentracing.ChildOf(pSp.Context()),
	)
}
