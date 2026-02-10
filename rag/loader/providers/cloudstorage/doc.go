// Package cloudstorage provides a DocumentLoader that loads files from cloud
// storage services (S3, GCS, Azure Blob). It detects the provider by URL prefix
// (s3://, gs://, az://) and uses direct HTTP calls with pre-signed URLs or
// service-specific APIs.
//
// # Registration
//
// The provider registers as "cloudstorage" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage"
//
//	l, err := loader.New("cloudstorage", config.ProviderConfig{
//	    APIKey: "your-access-key",
//	    Options: map[string]any{
//	        "secret_key": "your-secret-key",
//	        "region":     "us-east-1",
//	    },
//	})
//	docs, err := l.Load(ctx, "s3://bucket/path/to/file.txt")
//
// # Supported Providers
//
//   - S3 — URLs starting with "s3://"
//   - GCS — URLs starting with "gs://"
//   - Azure Blob — URLs starting with "az://"
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — access key (required)
//   - Options["secret_key"] — secret key
//   - Options["region"] — cloud region
package cloudstorage
