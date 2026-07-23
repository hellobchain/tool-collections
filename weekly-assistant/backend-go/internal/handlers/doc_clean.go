package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func DocClean(c *gin.Context) {
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

	removeComments := c.DefaultPostForm("remove_comments", "true") == "true"
	acceptChanges := c.DefaultPostForm("accept_changes", "true") == "true"

	cleaner := &services.DocCleaner{}
	result, err := cleaner.CleanFile(header.Filename, data, services.CleanOptions{
		RemoveComments: removeComments,
		AcceptChanges:  acceptChanges,
	})
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "清理失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"content": string(result),
	})
}
