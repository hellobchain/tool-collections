package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func GitLabCommits(c *gin.Context) {
	var req models.GitLabCommitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}

	projectName := req.ProjectName
	if projectName == "" {
		projectName = req.ProjectID
	}

	commits, err := services.FetchGitLabCommits(
		req.BaseURL, req.Token,
		req.ProjectID, req.Branch, req.Email, req.StartDate, req.EndDate,
	)
	if err != nil {
		utils.ErrorWithMsg(c, utils.CodeServerError, err.Error())
		return
	}

	for i := range commits {
		commits[i]["project_name"] = projectName
	}

	utils.Success(c, models.GitlabCommitResponse{
		ProjectID:   req.ProjectID,
		ProjectName: projectName,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Commits:     commits,
	})
}
