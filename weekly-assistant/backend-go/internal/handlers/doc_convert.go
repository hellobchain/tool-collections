package handlers

import (
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func DocConvert(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "请上传文件")
		return
	}
	defer file.Close()

	toFormats := c.DefaultPostForm("to_formats", "md")
	// do_ocr is accepted but ignored at this level; the downstream service handles it

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "读取文件失败")
		return
	}

	req := &services.ConvertRequest{
		File:     strings.NewReader(string(fileBytes)),
		FileName: header.Filename,
		ToFormat: toFormats,
	}

	result, err := services.DocConvertFile(req)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, "转换失败: "+err.Error())
		return
	}
	// Return the converted content as text based on requested format
	var content string
	switch toFormats {
	case "text":
		content = result.Data.Document.Text
	case "html":
		content = result.Data.Document.HTML
	case "json":
		content = result.Data.Document.JSON
	default:
		content = result.Data.Document.MD
	}

	if content == "" {
		// Fallback: return whatever is available
		if result.Data.Document.MD != "" {
			content = result.Data.Document.MD
		} else if result.Data.Document.Text != "" {
			content = result.Data.Document.Text
		} else if result.Data.Document.HTML != "" {
			content = result.Data.Document.HTML
		} else {
			content = result.Data.Document.JSON
		}
	}

	extMap := map[string]string{"md": ".md", "text": ".txt", "html": ".html", "json": ".json"}
	ext := extMap[toFormats]
	if ext == "" {
		ext = ".txt"
	}

	baseName := strings.TrimSuffix(header.Filename, ".docx")
	baseName = strings.TrimSuffix(baseName, ".doc")
	baseName = strings.TrimSuffix(baseName, ".pdf")
	trueFileName := baseName + ext

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+utils.PercentEncode(trueFileName))
	c.Data(200, "text/plain; charset=utf-8", []byte(content))
}
