// Package cloudstorage provides a DocumentLoader that loads files from cloud
// storage services (S3, GCS, Azure Blob). It detects the provider by URL prefix
// (s3://, gs://, az://) and fetches the object with a direct HTTPS GET.
//
// # Registration
//
// The provider registers as "cloudstorage" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/loader/providers/cloudstorage"
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
//   - S3 — "s3://<bucket>/<key>", fetched from <bucket>.s3.<region>.amazonaws.com
//   - GCS — "gs://<bucket>/<key>", fetched from storage.googleapis.com
//   - Azure Blob — "az://<account>/<container>/<blob>", fetched from
//     <account>.blob.core.windows.net
//
// # Security
//
// Bucket and account names are validated against each provider's naming
// rules before they become part of the request host, the object key is
// path-escaped, and requests are only sent to (and redirected within) the
// providers' own hosts, so a crafted source cannot redirect the request or
// its credentials to another host. Downloads are capped at
// Options["max_bytes"] (default 32 MiB).
//
// # Authentication
//
//   - S3 — SigV4: APIKey is the access key ID, Options["secret_key"] the
//     secret access key, Options["session_token"] an optional session token.
//   - GCS — APIKey is sent as an OAuth 2.0 bearer token.
//   - Azure — APIKey is sent as a Microsoft Entra ID bearer token (with
//     x-ms-version).
//
// Without an APIKey requests are anonymous (public objects).
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — access key ID (S3) or bearer token (GCS, Azure); optional
//   - Options["secret_key"] — S3 secret access key (required with APIKey for S3)
//   - Options["session_token"] — S3 session token (optional)
//   - Options["region"] — AWS region (default "us-east-1")
//   - Options["max_bytes"] — maximum object size in bytes (default 32 MiB)
package cloudstorage
