package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func DetectDocType(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请上传文件")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "读取文件失败")
		return
	}

	docType := services.DetectDocType(fileBytes)
	utils.Success(c, gin.H{"type": docType})
}
