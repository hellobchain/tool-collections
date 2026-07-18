package services

import (
	"context"
	"log"
	"sync"

	"github.com/hellobchain/oss-go-sdk/common/models"
	"github.com/hellobchain/oss-go-sdk/ossclient"
	"github.com/hellobchain/oss-go-sdk/ossclient/impl"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/constants"
)

var (
	ossClient ossclient.OssClient
	ossOnce   sync.Once
)

func GetOssClient() ossclient.OssClient {
	ossOnce.Do(func() {
		cfg := config.AppConfig
		clientConfig := &models.Config{
			Endpoint:        cfg.MinioEndpoint,
			AccessKeyID:     cfg.MinioAccessKey,
			SecretAccessKey: cfg.MinioSecretKey,
			BucketName:      cfg.MinioBucket,
			Region:          cfg.MinioRegion,
			IsS3:            cfg.MinioIsS3,
		}
		var err error
		log.Println("Using OSS storage,", cfg.SaveType)
		switch cfg.SaveType {
		case constants.LOCAL_SAVE_TYPE:
			clientConfig = &models.Config{
				Dir: cfg.LocalSavePath,
			}
			ossClient, err = impl.NewLocalClient(clientConfig)
		case constants.OSS_MINIO_SAVE_TYPE, constants.OSS_S3_SAVE_TYPE:
			ossClient, err = impl.NewS3Client(clientConfig)
		case constants.OSS_ALIYUN_SAVE_TYPE:
			ossClient, err = impl.NewAliClient(clientConfig)
		default:
			log.Fatalf("Invalid save type: %s", cfg.SaveType)
		}
		if err != nil {
			log.Fatalf("Failed to init OSS client: %v", err)
		}
		log.Printf("OSS client initialized: endpoint=%s bucket=%s", cfg.MinioEndpoint, cfg.MinioBucket)
	})
	return ossClient
}

func GetOssBucket() string {
	switch config.AppConfig.SaveType {
	case constants.LOCAL_SAVE_TYPE:
		return config.AppConfig.LocalSavePath
	default:
		return config.AppConfig.MinioBucket
	}
}

func UploadContractFile(ctx context.Context, fileSavePath string, data []byte) error {
	client := GetOssClient()
	return client.Upload(ctx, GetOssBucket(), fileSavePath, data)
}

func DownloadContractFile(ctx context.Context, fileSavePath string) ([]byte, error) {
	client := GetOssClient()
	return client.Download(ctx, GetOssBucket(), fileSavePath)
}

func DeleteContractFile(ctx context.Context, fileSavePath string) error {
	client := GetOssClient()
	return client.DeleteObject(ctx, GetOssBucket(), fileSavePath)
}
