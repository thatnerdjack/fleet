package s3

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

const (
	accessKeyID           = "minio"
	secretAccessKey       = "minio123!"
	testEndpoint          = "localhost:9000"
	mockInstallerContents = "mock"
)

func setupInstallerStore(tb testing.TB, bucket, prefix string) *InstallerStore {
	checkEnv(tb)

	store, err := NewInstallerStore(config.S3Config{
		Bucket:           bucket,
		Prefix:           prefix,
		Region:           "minio",
		EndpointURL:      testEndpoint,
		AccessKeyID:      accessKeyID,
		SecretAccessKey:  secretAccessKey,
		ForceS3PathStyle: true,
		DisableSSL:       true,
	})
	require.Nil(tb, err)

	err = store.CreateTestBucket(bucket)
	require.NoError(tb, err)

	tb.Cleanup(func() { cleanupStore(tb, store) })

	return store
}

func seedInstallerStore(tb testing.TB, store *InstallerStore, enrollSecret string) []fleet.Installer {
	checkEnv(tb)
	installers := []fleet.Installer{
		mockInstaller(enrollSecret, "pkg", true),
		mockInstaller(enrollSecret, "msi", true),
		mockInstaller(enrollSecret, "deb", true),
		mockInstaller(enrollSecret, "rpm", true),
		mockInstaller(enrollSecret, "pkg", false),
		mockInstaller(enrollSecret, "msi", false),
		mockInstaller(enrollSecret, "deb", false),
		mockInstaller(enrollSecret, "rpm", false),
	}

	for _, i := range installers {
		_, err := store.Put(context.Background(), i)
		require.NoError(tb, err)
	}

	return installers
}

func mockInstaller(secret, kind string, desktop bool) fleet.Installer {
	return fleet.Installer{
		EnrollSecret: secret,
		Kind:         kind,
		Desktop:      desktop,
		Content:      aws.ReadSeekCloser(strings.NewReader(mockInstallerContents)),
	}
}

func cleanupStore(tb testing.TB, store *InstallerStore) {
	checkEnv(tb)
	resp, err := store.s3client.ListObjects(&s3.ListObjectsInput{
		Bucket: &store.bucket,
	})
	require.NoError(tb, err)

	var objs []*s3.ObjectIdentifier
	for _, o := range resp.Contents {
		objs = append(objs, &s3.ObjectIdentifier{Key: o.Key})
	}
	_, err = store.s3client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: &store.bucket,
		Delete: &s3.Delete{
			Objects: objs,
		},
	})
	require.NoError(tb, err)

	_, err = store.s3client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: &store.bucket,
	})
	require.NoError(tb, err)
}

func checkEnv(tb testing.TB) {
	if _, ok := os.LookupEnv("MINIO_STORAGE_TEST"); !ok {
		tb.Skip("set MINIO_STORAGE_TEST environment variable to run S3-based tests")
	}
}
