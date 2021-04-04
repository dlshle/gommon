package oss

import "github.com/aliyun/aliyun-oss-go-sdk/oss"

type OSSBucketInfo = oss.BucketInfo

type OSSBucket struct {
	bucket *oss.Bucket
	info   OSSBucketInfo
}

type OSSClient struct {
	client  *oss.Client
	buckets []*OSSBucket
}

type IOSSClient interface {
	findBucket(name string) *OSSBucket
	forEachBucket(bucketFunc func(bucket *OSSBucket))
	mapBuckets(mapFunc func(bucket *OSSBucket) interface{}) []interface{}
	Buckets() []OSSBucketInfo
	HasBucket(bucketName string) bool
}

func NewClient(endpoint, keyId, keySecret string) (*OSSClient, error) {
	rawClient, err := oss.New(endpoint, keyId, keySecret)
	if err != nil {
		return nil, err
	}
	bucketListResult, err := rawClient.ListBuckets()
	if err != nil {
		return nil, err
	}
	bucketProperties := bucketListResult.Buckets
	ossBuckets := make([]*OSSBucket, len(bucketProperties))
	for i, bp := range bucketProperties {
		bucketName := bp.Name
		infoResult, err := rawClient.GetBucketInfo(bucketName)
		bucket, err := rawClient.Bucket(bucketName)
		if err != nil {
			return nil, err
		}
		ossBuckets[i] = &OSSBucket{bucket, infoResult.BucketInfo}
	}
	return &OSSClient{rawClient, ossBuckets}, nil
}

func (c *OSSClient) findBucket(name string) *OSSBucket {
	for _, bucket := range c.buckets {
		if bucket.bucket.BucketName == name {
			return bucket
		}
	}
	return nil
}

// PLEASE DO NOT MODIFY BUCKET
func (c *OSSClient) forEachBucket(bucketFunc func(bucket *OSSBucket)) {
	for _, bucket := range c.buckets {
		bucketFunc(bucket)
	}
}

// PLEASE DO NOT MODIFY BUCKET
func (c *OSSClient) mapBuckets(mapFunc func(bucket *OSSBucket) interface{}) []interface{} {
	mapResult := make([]interface{}, len(c.buckets))
	for i, bucket := range c.buckets {
		mapResult[i] = mapFunc(bucket)
	}
	return mapResult
}

func (c *OSSClient) Buckets() []OSSBucketInfo {
	bucketInfos := make([]OSSBucketInfo, len(c.buckets))
	for i, bucket := range c.buckets {
		bucketInfos[i] = bucket.info
	}
	return bucketInfos
}

func (c *OSSClient) HasBucket(bucketName string) bool {
	return c.findBucket(bucketName) != nil
}

// TODO get file infos, upload string, byte stream, local files, get string, byte stream, files
