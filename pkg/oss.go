package pkg

import (
	"bytes"
	"context"
	"fmt"
	"one-api/common"
	"one-api/logger"
	"os"
	"strings"

	sdk "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type aliyunOss struct {
	Client    *sdk.Client
	Bucket    *sdk.Bucket
	Path      string
	CdnDomain string
}

var AliyunOssClient *aliyunOss

func InitAliyunOssClient() (err error) {
	config := os.Getenv("ALIYUN_OSS_CONN_STRING")
	configParams := strings.Split(config, "|")
	if config == "" || len(configParams) != 6 {
		common.SysLog("ALIYUN_OSS_CONN_STRING not set or incorrect, Aliyun oss is not enabled")
		return nil
	}
	endpoint := configParams[0]
	accessKeyID := configParams[1]
	accessKeySecret := configParams[2]
	bucketName := configParams[3]
	cdnDomain := configParams[4]
	dirName := configParams[5]

	client, err := sdk.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		common.SysLog("NewAliyunOssClient oss.New err,err=" + err.Error())
		return
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		common.SysLog("NewAliyunOssClient client.Bucket err,err=" + err.Error())
		return
	}
	AliyunOssClient = &aliyunOss{
		Client:    client,
		Bucket:    bucket,
		Path:      dirName,
		CdnDomain: cdnDomain,
	}
	common.SysLog("InitAliyunOssClient success")
	return
}

func (oss aliyunOss) UploadFileWithBytes(bytesData []byte, dirName string, fileName string) (dstUrl string, err error) {
	objectName := fmt.Sprintf("%s/%s/%s", oss.Path, dirName, fileName)
	err = oss.Bucket.PutObject(objectName, bytes.NewReader(bytesData))
	if err != nil {
		logger.LogError(context.Background(), "UploadFileWithBytes oss.Bucket.PutObject err,err="+err.Error())
		return
	}
	dstUrl = fmt.Sprintf("%s/%s", oss.CdnDomain, objectName)
	return
}
