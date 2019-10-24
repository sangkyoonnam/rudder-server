package fileuploader

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

// Upload passed in file to s3
func (uploader *S3Uploader) Upload(file *os.File, prefixes ...string) error {
	if uploader.Bucket == "" {
		return errors.New("no S3 bucket configured to uploader")
	}
	getRegionSession := session.Must(session.NewSession())
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), getRegionSession, uploader.Bucket, "us-east-1")
	uploadSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		// Credentials: credentials.NewStaticCredentials(config.GetEnv("IAM_S3_COPY_ACCESS_KEY_ID", ""), config.GetEnv("IAM_S3_COPY_SECRET_ACCESS_KEY", ""), ""),
	}))
	manager := s3manager.NewUploader(uploadSession)
	splitFileName := strings.Split(file.Name(), "/")
	fileName := ""
	if len(prefixes) > 0 {
		fileName = strings.Join(prefixes[:], "/") + "/"
	}
	fileName += splitFileName[len(splitFileName)-1]
	_, err = manager.Upload(&s3manager.UploadInput{
		ACL:    aws.String("bucket-owner-full-control"),
		Bucket: aws.String(uploader.Bucket),
		Key:    aws.String(fileName),
		Body:   file,
	})
	// do not panic if upload has failed for customer s3 bucket
	// misc.AssertError(err)
	return err
}

func (uploader *S3Uploader) Download(output *os.File, key string) error {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	downloader := s3manager.NewDownloader(sess)
	_, err := downloader.Download(output,
		&s3.GetObjectInput{
			Bucket: aws.String(uploader.Bucket),
			Key:    aws.String(key),
		})
	misc.AssertError(err)
	return err
}

type S3Object struct {
	Key              string
	LastModifiedTime time.Time
}

func (uploader *S3Uploader) ListFilesWithPrefix(prefix string) ([]*S3Object, error) {
	s3Objects := make([]*S3Object, 0)

	getRegionSession := session.Must(session.NewSession())
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), getRegionSession, uploader.Bucket, "us-east-1")
	uploadSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	// Create S3 service client
	svc := s3.New(uploadSession)

	// Get the list of items
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(uploader.Bucket),
		Prefix: aws.String(prefix),
		// Delimiter: aws.String("/"),
	})
	if err != nil {
		return s3Objects, err
	}

	for _, item := range resp.Contents {
		s3Objects = append(s3Objects, &S3Object{*item.Key, *item.LastModified})
	}

	return s3Objects, nil
}

type S3Uploader struct {
	Bucket string
}
