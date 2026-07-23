package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func MarkitdownConvert(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请上传文件")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "读取文件失败")
		return
	}

	md := &services.Markitdown{}
	result, err := md.Convert(header.Filename, data)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "转换失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"title":        result.Title,
		"content":      result.Content,
		"text_content": result.TextContent,
	})
}
