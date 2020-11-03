package util

import (
	"crypto/md5"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"strings"
)

//
func MergeStringMap(base, toMerge map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range toMerge {
		result[k] = v
	}
	return result
}

//
func ContainString(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

// 忽略没有匹配资源的错误
func IgnoreNoMatchError(err error) error {
	if err != nil && !apierrors.IsNotFound(err) && !strings.HasPrefix(err.Error(), "no matches for kind") {
		return err
	}
	return nil
}

//
func GetDeleteCheckSum(name string) string {
	salt := os.Getenv("MD5_SALT")
	if salt == "" {
		salt = "0e80b3a3-ad6b-4bc5-a41e-57ea49266417"
	}
	checksum := md5.Sum([]byte(name + salt))
	return fmt.Sprintf("%x", checksum)
}

func GetSubsetName(service, plane string) string {
	serviceName := strings.Split(service, ".")[0]
	return serviceName + "-" + plane
}
