package about

import (
	"GuGoTik/src/web/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func Handle(c *gin.Context) {
	var req models.AboutReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_msg": "GuGoTik Gateway can not response to your request, please try again later",
		})
	}
	res := models.AboutRes{
		Echo:      req.Echo,
		TimeStamp: time.Now().Unix(),
	}
	c.JSON(http.StatusOK, res)
}
