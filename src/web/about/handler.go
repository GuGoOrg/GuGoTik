package about

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/utils/trace"
	"GuGoTik/src/web/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func Handle(c *gin.Context) {
	span := trace.GetChildSpanFromGinContext(c, "gateway-about")
	defer span.Finish()
	var req models.AboutReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": strings.GateWayErrorCode,
			"status_msg":  strings.GateWayError,
		})
	}
	res := models.AboutRes{
		Echo:      req.Echo,
		TimeStamp: time.Now().Unix(),
	}
	c.JSON(http.StatusOK, res)
}
