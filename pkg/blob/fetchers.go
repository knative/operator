/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package blob

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gocloud.dev/blob"
	// Include S3 URL-scheme
	_ "gocloud.dev/blob/s3blob"
	// Include GCS URL-scheme
	"gocloud.dev/blob/gcsblob"
	// Include Azure URL-scheme
	_ "gocloud.dev/blob/azureblob"
	// Include file URL-scheme
	_ "gocloud.dev/blob/fileblob"
	"gocloud.dev/gcp"
	"knative.dev/operator/pkg/packages"
)

func init() {
	if os.Getenv("GCS_NOAUTH") != "" {
		opener := gcsblob.URLOpener{
			Client: gcp.NewAnonymousHTTPClient(gcp.DefaultTransport()),
		}
		blob.DefaultURLMux().RegisterBucket(gcsblob.Scheme+"-noauth", &opener)
	}
}

// GetReleases collects and returns releases from the specified S3 configuration.
func GetReleases(ctx context.Context, client *http.Client, cfg packages.S3Source) ([]packages.Release, error) {
	bucketUrl, err := url.Parse(cfg.Bucket)
	if err != nil {
		return nil, err
	}
	bucketName := bucketUrl.Host
	b, err := blob.OpenBucket(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	li := blob.ListOptions{
		Prefix: cfg.Prefix,
	}
	iter := b.List(&li)
	type releaseKey struct {
		repo string
		tag  string
	}
	commonRepo := ""

	found := make(map[releaseKey][]packages.Asset)
	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		fields := strings.Split(obj.Key, "/")
		if commonRepo == "" {
			commonRepo = fields[0]
		}
		if fields[0] != commonRepo {
			return nil, fmt.Errorf("Multiple repos listed in a single release source (%q and %q) in %s", fields[0], commonRepo, cfg)
		}
		rk := releaseKey{
			repo: fields[0],
			tag:  fields[len(fields)-2],
		}
		found[rk] = append(found[rk], packages.Asset{
			Name: fields[len(fields)-1],
			// GCS has two formats:
			// https://storage.googleapis.com/storage/v1/b/knative-releases/o/serving%2Fprevious%2Fv0.20.0%2Fserving-core.yaml?alt=media
			// https://knative-releases.storage.googleapis.com/serving/previous/v0.20.0/serving-core.yaml
			// We use the second (shorter) one
			URL: "https://" + bucketName + ".storage.googleapis.com/" + obj.Key,
		})
	}
	ret := make([]packages.Release, 0, len(found))
	for k, v := range found {
		ret = append(ret, packages.Release{
			Org:     "",
			Repo:    commonRepo,
			TagName: k.tag,
			Assets:  v,
		})
	}
	return ret, nil
}
