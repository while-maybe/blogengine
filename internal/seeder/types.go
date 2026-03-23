package seeder

import "blogengine/internal/storage"

type BlogManifest struct {
	Title             string                   `yaml:"title"`
	Description       string                   `yaml:"description"`
	Visibility        storage.Visibility       `yaml:"visibility"`
	RegistrationMode  storage.RegistrationMode `yaml:"registration_mode"`
	RegistrationLimit *int64                   `yaml:"registration_limit"`
	Owner             string                   `yaml:"owner"`
}

type PostFrontmatter struct {
	Title         string  `yaml:"title"`
	Description   string  `yaml:"description"`
	Slug          string  `yaml:"slug"`
	IsListed      bool    `yaml:"is_listed"`
	PublishedAt   *string `yaml:"published_at"`
	IsEncrypted   bool    `yaml:"is_encrypted"`
	EncryptionIV  *string `yaml:"encryption_iv"`
	RequiresAuth  bool    `yaml:"requires_auth"`
	AllowComments bool    `yaml:"allow_comments"`
}
