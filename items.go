package main

import (
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
var quotesRegexp = regexp.MustCompile(`"`)

// Item is a representation of a single config key and value
type Item struct {
	Key, Value string
}

// Export returns a string that can be evaluated by a shell to set key=value in
// the environment
func (i *Item) Export() string {
	v := i.Value
	v = quotesRegexp.ReplaceAllString(v, `\"`)

	return fmt.Sprintf("export %s=\"%s\"", strings.ToUpper(i.Key), v)
}

// EnvSafeKey returns an environment variable safe key
func (i *Item) EnvSafeKey() string {
	return strings.ToUpper(i.Key)
}

// EnvSafeValue returns an environment variable safe value
func (i *Item) EnvSafeValue() string {
	v := i.Value
	v = quotesRegexp.ReplaceAllString(v, `\"`)
	return v
}

// Collection is a collection of single key value items and the source. If
// there were source processing errors they'll be saved in .Error
type Collection struct {
	Items  map[string]*Item
	Source string
	Error  error
}

// NewCollection initializes a collection
func NewCollection() *Collection {
	return &Collection{
		Items: make(map[string]*Item),
	}
}

// AppendItem will add an item to the internal Items map if the key
// validates. If the key doesn't validate an error will be returned and no
// item will be written.
func (c *Collection) AppendItem(key, val string) error {
	if !keyRegexp.MatchString(key) {
		return errors.New(key + " contains invalid characters")
	}
	key = strings.ToUpper(key)
	c.Items[key] = &Item{Key: key, Value: val}
	return nil
}

// Len returns the number of items in the collection
func (c *Collection) Len() int {
	return len(c.Items)
}

// AsMap represents the collection as a map[string]string
func (c *Collection) AsMap() map[string]string {
	out := make(map[string]string)
	for _, i := range c.Items {
		out[i.EnvSafeKey()] = i.EnvSafeValue()
	}
	return out
}

// GetItemString will return the value of a item by key
func (c *Collection) GetItemString(key string) (string, bool) {
	item, ok := c.Items[key]
	if !ok {
		return "", false
	}
	return item.Value, true
}

// ReadItemsFromReader will read in items from an io.Reader into the collection
// Items map
func (c *Collection) ReadItemsFromReader(r io.Reader) error {
	var f map[string]interface{}
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		c.Error = err
		return err
	}
	for k, v := range f {
		switch vv := v.(type) {
		case string:
			c.AppendItem(k, vv)
		case float64:
			c.AppendItem(k, strconv.FormatFloat(vv, 'f', -1, 64))
		case bool:
			var b string
			if vv {
				b = "1"
			} else {
				b = "0"
			}
			c.AppendItem(k, b)
		}
	}
	return nil
}

// LoadItemsFromSource will write items from a source URL
// Currently assumes s3
func LoadItemsFromSource(source *url.URL) *Collection {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
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
	secrets.ReadItemsFromReader(result.Body)
	return secrets
}
