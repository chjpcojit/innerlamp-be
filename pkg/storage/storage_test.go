package storage

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty backend",
			config:  &Config{Timeout: 30 * time.Second},
			wantErr: true,
			errMsg:  "backend is required",
		},
		{
			name: "missing timeout",
			config: &Config{
				Backend: BackendGCS,
				GCS: &GCSConfig{
					ProjectID:             "test-project",
					UseDefaultCredentials: true,
				},
			},
			wantErr: true,
			errMsg:  "timeout is required",
		},
		{
			name: "unsupported backend",
			config: &Config{
				Backend: Backend("unknown"),
				Timeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "unsupported storage backend",
		},
		{
			name: "GCS without config",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "GCS configuration is required",
		},
		{
			name: "GCS without project ID",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
				GCS:     &GCSConfig{},
			},
			wantErr: true,
			errMsg:  "GCS project ID is required",
		},
		{
			name: "GCS without credentials",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
				GCS: &GCSConfig{
					ProjectID:             "test-project",
					UseDefaultCredentials: false,
				},
			},
			wantErr: true,
			errMsg:  "GCS credentials are required",
		},
		{
			name: "valid GCS with default credentials",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
				GCS: &GCSConfig{
					ProjectID:             "test-project",
					UseDefaultCredentials: true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid GCS with credentials file",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
				GCS: &GCSConfig{
					ProjectID:       "test-project",
					CredentialsFile: "/path/to/creds.json",
				},
			},
			wantErr: false,
		},
		{
			name: "valid GCS with credentials JSON",
			config: &Config{
				Backend: BackendGCS,
				Timeout: 30 * time.Second,
				GCS: &GCSConfig{
					ProjectID:       "test-project",
					CredentialsJSON: []byte(`{"type": "service_account"}`),
				},
			},
			wantErr: false,
		},
		{
			name: "S3 without config",
			config: &Config{
				Backend: BackendS3,
				Timeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "S3 configuration is required",
		},
		{
			name: "S3 without region",
			config: &Config{
				Backend: BackendS3,
				Timeout: 30 * time.Second,
				S3:      &S3Config{},
			},
			wantErr: true,
			errMsg:  "S3 region is required",
		},
		{
			name: "valid S3 config",
			config: &Config{
				Backend: BackendS3,
				Timeout: 30 * time.Second,
				S3: &S3Config{
					Region: "us-east-1",
				},
			},
			wantErr: false,
		},
		{
			name: "valid S3 with credentials",
			config: &Config{
				Backend: BackendS3,
				Timeout: 30 * time.Second,
				S3: &S3Config{
					Region:          "us-east-1",
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name       string
		objectName string
		want       string
	}{
		{"HTML file", "index.html", "text/html"},
		{"HTM file", "page.htm", "text/html"},
		{"CSS file", "styles.css", "text/css"},
		{"JavaScript file", "app.js", "application/javascript"},
		{"JSON file", "data.json", "application/json"},
		{"XML file", "config.xml", "application/xml"},
		{"Plain text", "readme.txt", "text/plain"},
		{"CSV file", "data.csv", "text/csv"},
		{"PDF file", "document.pdf", "application/pdf"},
		{"ZIP file", "archive.zip", "application/zip"},
		{"GZIP file", "data.gz", "application/gzip"},
		{"TAR file", "backup.tar", "application/x-tar"},
		{"PNG image", "image.png", "image/png"},
		{"JPEG image", "photo.jpg", "image/jpeg"},
		{"JPEG image alt", "photo.jpeg", "image/jpeg"},
		{"GIF image", "animation.gif", "image/gif"},
		{"SVG image", "icon.svg", "image/svg+xml"},
		{"WebP image", "image.webp", "image/webp"},
		{"ICO file", "favicon.ico", "image/x-icon"},
		{"MP3 audio", "song.mp3", "audio/mpeg"},
		{"WAV audio", "sound.wav", "audio/wav"},
		{"MP4 video", "video.mp4", "video/mp4"},
		{"WebM video", "video.webm", "video/webm"},
		{"WOFF font", "font.woff", "font/woff"},
		{"WOFF2 font", "font.woff2", "font/woff2"},
		{"TTF font", "font.ttf", "font/ttf"},
		{"OTF font", "font.otf", "font/otf"},
		{"EOT font", "font.eot", "application/vnd.ms-fontobject"},
		{"YAML file", "config.yaml", "application/x-yaml"},
		{"YML file", "config.yml", "application/x-yaml"},
		{"Markdown file", "README.md", "text/markdown"},
		{"Go file", "main.go", "text/x-go"},
		{"Python file", "script.py", "text/x-python"},
		{"Ruby file", "app.rb", "text/x-ruby"},
		{"Java file", "Main.java", "text/x-java"},
		{"C file", "program.c", "text/x-c"},
		{"C++ file", "program.cpp", "text/x-c++"},
		{"Header file", "header.h", "text/x-c"},
		{"Rust file", "main.rs", "text/x-rust"},
		{"TypeScript file", "app.ts", "application/typescript"},
		{"TSX file", "component.tsx", "application/typescript"},
		{"JSX file", "component.jsx", "application/javascript"},
		{"Unknown extension", "file.unknown", "application/octet-stream"},
		{"No extension", "README", "application/octet-stream"},
		{"Path with directory", "path/to/image.png", "image/png"},
		{"Deep path", "a/b/c/d/e/file.json", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.objectName)
			if got != tt.want {
				t.Errorf("DetectContentType(%q) = %v, want %v", tt.objectName, got, tt.want)
			}
		})
	}
}

func TestValidateObjectName(t *testing.T) {
	tests := []struct {
		name    string
		objName string
		wantErr error
	}{
		{
			name:    "empty name",
			objName: "",
			wantErr: ErrEmptyObjectName,
		},
		{
			name:    "valid simple name",
			objName: "file.txt",
			wantErr: nil,
		},
		{
			name:    "valid path",
			objName: "path/to/file.txt",
			wantErr: nil,
		},
		{
			name:    "consecutive slashes",
			objName: "path//to/file.txt",
			wantErr: ErrInvalidObjectName,
		},
		{
			name:    "valid with special chars",
			objName: "path/to/file-name_v1.2.3.txt",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateObjectName(tt.objName)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v but got nil", tt.wantErr)
					return
				}
				if !containsString(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name       string
		bucketName string
		wantErr    bool
	}{
		{
			name:       "empty name",
			bucketName: "",
			wantErr:    true,
		},
		{
			name:       "too short",
			bucketName: "ab",
			wantErr:    true,
		},
		{
			name:       "minimum length",
			bucketName: "abc",
			wantErr:    false,
		},
		{
			name:       "valid name",
			bucketName: "my-bucket-name",
			wantErr:    false,
		},
		{
			name:       "maximum length (63 chars)",
			bucketName: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz1",
			wantErr:    false,
		},
		{
			name:       "too long (64 chars)",
			bucketName: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz12",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBucketName(tt.bucketName)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCopyMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name:     "empty input",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name:     "with data",
			input:    map[string]string{"key1": "value1", "key2": "value2"},
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CopyMetadata(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Length = %v, want %v", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("result[%q] = %v, want %v", k, result[k], v)
				}
			}

			// Verify it's a copy, not the same reference
			if tt.input != nil && len(tt.input) > 0 {
				result["new_key"] = "new_value"
				if _, exists := tt.input["new_key"]; exists {
					t.Error("Modifying result should not affect input")
				}
			}
		})
	}
}

func TestMergeMetadata(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []map[string]string
		expected map[string]string
	}{
		{
			name:     "no inputs",
			inputs:   nil,
			expected: map[string]string{},
		},
		{
			name:     "single input",
			inputs:   []map[string]string{{"key1": "value1"}},
			expected: map[string]string{"key1": "value1"},
		},
		{
			name: "multiple inputs",
			inputs: []map[string]string{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "overlapping keys - later wins",
			inputs: []map[string]string{
				{"key1": "value1", "shared": "first"},
				{"key2": "value2", "shared": "second"},
			},
			expected: map[string]string{"key1": "value1", "key2": "value2", "shared": "second"},
		},
		{
			name: "with nil map",
			inputs: []map[string]string{
				{"key1": "value1"},
				nil,
				{"key2": "value2"},
			},
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMetadata(tt.inputs...)

			if len(result) != len(tt.expected) {
				t.Errorf("Length = %v, want %v", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("result[%q] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestObjectInfo(t *testing.T) {
	now := time.Now()
	info := &ObjectInfo{
		Name:               "test/file.txt",
		Bucket:             "my-bucket",
		Size:               1024,
		ContentType:        "text/plain",
		Created:            now,
		Updated:            now,
		ETag:               "abc123",
		StorageClass:       StorageClassStandard,
		Metadata:           map[string]string{"custom": "value"},
		CacheControl:       "max-age=3600",
		ContentEncoding:    "gzip",
		ContentDisposition: "attachment",
		ContentLanguage:    "en-US",
	}

	if info.Name != "test/file.txt" {
		t.Errorf("Name = %v, want test/file.txt", info.Name)
	}
	if info.Bucket != "my-bucket" {
		t.Errorf("Bucket = %v, want my-bucket", info.Bucket)
	}
	if info.Size != 1024 {
		t.Errorf("Size = %v, want 1024", info.Size)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("ContentType = %v, want text/plain", info.ContentType)
	}
	if info.Metadata["custom"] != "value" {
		t.Errorf("Metadata[custom] = %v, want value", info.Metadata["custom"])
	}
}

func TestBucketInfo(t *testing.T) {
	now := time.Now()
	info := &BucketInfo{
		Name:         "my-bucket",
		Location:     "us-east-1",
		StorageClass: StorageClassStandard,
		Created:      now,
		Labels:       map[string]string{"env": "prod"},
		Versioning:   true,
	}

	if info.Name != "my-bucket" {
		t.Errorf("Name = %v, want my-bucket", info.Name)
	}
	if info.Location != "us-east-1" {
		t.Errorf("Location = %v, want us-east-1", info.Location)
	}
	if !info.Versioning {
		t.Error("Versioning should be true")
	}
	if info.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %v, want prod", info.Labels["env"])
	}
}

func TestStorageConstants(t *testing.T) {
	// Test Backend constants
	if BackendGCS != "gcs" {
		t.Errorf("BackendGCS = %v, want gcs", BackendGCS)
	}
	if BackendS3 != "s3" {
		t.Errorf("BackendS3 = %v, want s3", BackendS3)
	}

	// Test StorageClass constants
	if StorageClassStandard != "STANDARD" {
		t.Errorf("StorageClassStandard = %v, want STANDARD", StorageClassStandard)
	}
	if StorageClassNearline != "NEARLINE" {
		t.Errorf("StorageClassNearline = %v, want NEARLINE", StorageClassNearline)
	}
	if StorageClassColdline != "COLDLINE" {
		t.Errorf("StorageClassColdline = %v, want COLDLINE", StorageClassColdline)
	}
	if StorageClassArchive != "ARCHIVE" {
		t.Errorf("StorageClassArchive = %v, want ARCHIVE", StorageClassArchive)
	}

	// Test ACL constants
	if ACLPrivate != "private" {
		t.Errorf("ACLPrivate = %v, want private", ACLPrivate)
	}
	if ACLPublicRead != "public-read" {
		t.Errorf("ACLPublicRead = %v, want public-read", ACLPublicRead)
	}

	// Test SignedURLMethod constants
	if SignedURLMethodGET != "GET" {
		t.Errorf("SignedURLMethodGET = %v, want GET", SignedURLMethodGET)
	}
	if SignedURLMethodPUT != "PUT" {
		t.Errorf("SignedURLMethodPUT = %v, want PUT", SignedURLMethodPUT)
	}
	if SignedURLMethodDELETE != "DELETE" {
		t.Errorf("SignedURLMethodDELETE = %v, want DELETE", SignedURLMethodDELETE)
	}
}

func TestNewWithNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New(nil) should return error")
	}
	if !containsString(err.Error(), "config cannot be nil") {
		t.Errorf("New(nil) error = %v, want error containing 'config cannot be nil'", err)
	}
}

func TestNewWithInvalidConfig(t *testing.T) {
	config := &Config{
		Backend: Backend("unknown"),
	}
	_, err := New(config)
	if err == nil {
		t.Error("New with invalid backend should return error")
	}
}

func TestNewWithUnregisteredBackend(t *testing.T) {
	config := &Config{
		Backend: BackendGCS,
		Timeout: 30 * time.Second,
		GCS: &GCSConfig{
			ProjectID:             "test-project",
			UseDefaultCredentials: true,
		},
	}

	// This will fail because we haven't imported the gcs package
	// which registers the factory
	_, err := New(config)
	if err == nil {
		// If no error, the factory is registered (which is fine in integration tests)
		t.Log("GCS factory is registered")
	} else {
		// Expected behavior when factory is not registered
		if !containsString(err.Error(), "unsupported storage backend") {
			t.Errorf("Expected unsupported backend error, got: %v", err)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
