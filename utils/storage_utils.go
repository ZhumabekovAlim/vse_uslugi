package utils

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Настройки для подключения к PSCloud (S3-совместимый сервис)
var (
	accessKey = "01IKG2JGI6AGBYW5K057"
	secretKey = "XPcuZGTQmb14gu1ptDBFfF5d2o2R5KTJzIejRo91"
	bucket    = "udg-mobile"
	region    = "us-east-1"
	endpoint  = "https://object.pscloud.io"
)

// Получаем клиента для работы с S3 (PSCloud)
func getS3Client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(
			accessKey, secretKey, "", // Статические учетные данные
		),
	}))
	return s3.New(sess)
}

// Загружаем файл на S3
func UploadFileToS3(file []byte, fileName string, folder string) (string, error) {
	// Формируем путь для файла на S3
	filePath := fmt.Sprintf("%s/%s", folder, fileName)

	// Получаем клиента S3
	s3Client := getS3Client()

	// Загружаем файл на S3
	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(filePath),
		Body:          bytes.NewReader(file),
		ContentLength: aws.Int64(int64(len(file))),
		ContentType:   aws.String("image/jpeg"),
		ACL:           aws.String("public-read"), // Публичный доступ
	})

	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3: %v", err)
	}

	// Возвращаем URL для доступа к файлу
	return fmt.Sprintf("https://%s.object.pscloud.io/%s", bucket, filePath), nil
}
