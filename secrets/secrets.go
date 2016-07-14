package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Key validation regexp
var keyRegexp = regexp.MustCompile(`^\w+$`)

// Secret - Representation of a single secret's key and value
type Secret struct {
	Key, Value string
}

// Export returns a string that can be evaluated by a shell to set key=value in
// the environment
func (s *Secret) Export() string {
	newLines := regexp.MustCompile(`\n`)
	singleQuotes := regexp.MustCompile(`'`)
	v := s.Value
	v = newLines.ReplaceAllString(v, `\n`)
	v = singleQuotes.ReplaceAllString(v, `\'`)

	// Wrap in export $''
	return ("export " + strings.ToUpper(s.Key) + "=$'" + v + "'")
}

// Collection is a collection of single secrets and the source. If there were
// source processing errors they'll be saved in .Error
type Collection struct {
	secrets map[string]*Secret
	Source  string
	Error   error
}

// NewCollection initializes a collection
func NewCollection() *Collection {
	return &Collection{
		secrets: make(map[string]*Secret),
	}
}

// WriteSecret will write a secret to the internal Secrets map if the key
// validates. If the key doesn't validate an error will be returned and no
// secret will be written.
func (s *Collection) WriteSecret(key, val string) error {
	if !keyRegexp.MatchString(key) {
		return errors.New(key + " contains invalid characters")
	}
	key = strings.ToUpper(key)
	s.secrets[key] = &Secret{Key: key, Value: val}
	return nil
}

// Len returns the number of secrets in the collection
func (s *Collection) Len() int {
	return len(s.secrets)
}

// Exports are all the exports
func (s *Collection) Exports() string {
	var buffer bytes.Buffer
	for _, s := range s.secrets {
		buffer.WriteString(s.Export())
		buffer.WriteString("\n")
	}
	return buffer.String()
}

// Print prints the exports
func (s *Collection) Print() {
	fmt.Print(s.Exports())
}

// GetSecretString will return the value of a single secret by key
func (s *Collection) GetSecretString(key string) (string, bool) {
	secret, ok := s.secrets[key]
	if !ok {
		return "", false
	}
	return secret.Value, true
}

// ReadSecretsFromReader will read in secrets from an io.Reader
// This will read secrets into the internal Secrest map and set any errors
func (s *Collection) ReadSecretsFromReader(r io.Reader) error {
	var f map[string]interface{}
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		s.Error = err
		return err
	}
	for k, v := range f {
		switch vv := v.(type) {
		case string:
			s.WriteSecret(k, vv)
		case float64:
			s.WriteSecret(k, strconv.FormatFloat(vv, 'f', -1, 64))
		case bool:
			var b string
			if vv {
				b = "1"
			} else {
				b = "0"
			}
			s.WriteSecret(k, b)
		}
	}
	return nil
}

// LoadSecretsFromSource will write secrets from a source URL
// Currently assumes s3
func LoadSecretsFromSource(source *url.URL) *Collection {
	sess := session.New()
	region := source.Query().Get("region")
	config := aws.Config{}
	secrets := NewCollection()
	secrets.Source = source.String()

	if region != "" {
		config.Region = aws.String(region)
	}
	svc := s3.New(sess, &config)
	result, s3err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(source.Host),
		Key:    aws.String(source.Path),
	})

	if s3err != nil {
		secrets.Error = s3err
		return secrets
	}

	defer result.Body.Close()
	secrets.ReadSecretsFromReader(result.Body)
	return secrets
}
