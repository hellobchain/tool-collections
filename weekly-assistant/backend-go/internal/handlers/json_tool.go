package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

type langConvertRequest struct {
	JsonData string `json:"json_data"`
	Language string `json:"language"`
}

func JsonLangConvert(c *gin.Context) {
	var req langConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if req.JsonData == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "json_data 不能为空")
		return
	}
	if req.Language == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "language 不能为空")
		return
	}

	code, err := services.JsonToLang(req.JsonData, req.Language)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	utils.Success(c, gin.H{"code": code})
}
