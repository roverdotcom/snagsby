package resolvers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
)

func getAwsSession() (*session.Session, error) {
	return session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
}

func readJSONString(input string) (map[string]string, error) {
	var f map[string]interface{}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(input), &f); err != nil {
		return out, err
	}
	for k, v := range f {
		k = strings.ToUpper(k)
		switch vv := v.(type) {
		case string:
			out[k] = vv
		case float64:
			out[k] = strconv.FormatFloat(vv, 'f', -1, 64)
		case bool:
			var b string
			if vv {
				b = "1"
			} else {
				b = "0"
			}
			out[k] = b
		}
	}
	return out, nil
}
