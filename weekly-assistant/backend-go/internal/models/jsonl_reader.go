package models

type JsonlReaderRequest struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
	Schema   string `json:"schema"`
}
