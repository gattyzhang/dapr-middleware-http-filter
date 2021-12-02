package filter

import (
	//"net/http/httptest"
	"testing"

	"github.com/dapr/components-contrib/middleware"
	"github.com/stretchr/testify/assert"
)

var metaTest middleware.Metadata

const (
	//filter_url                    = "filter_url"
	//req_header_cookie_parms       = "req_header_cookie_parms"
	//filter_err_url                = "filter_err_url"
	filter_url_value              = "http://httpbin.org/post?key=123"
	req_header_cookie_parms_value = "Appid,Token,Session-Token,Flag-Forward"
	filter_err_url_value          = "https://www.pcauto.com.cn"
)

func init() {
	metaTest.Properties = make(map[string]string)
	metaTest.Properties[req_header_cookie_parms] = req_header_cookie_parms_value
	metaTest.Properties[filter_url] = filter_url_value
	metaTest.Properties[filter_err_url] = filter_err_url_value

}
func TestFilterInit(t *testing.T) {

	var m Middleware
	metaResult, err := m.getNativeMetadata(metaTest)
	t.Log("call m.getNativeMetadata, err:", err)
	t.Log("metaResult:", metaResult)
	assert.Equal(t, filter_url_value, metaResult.Filter_url)
	assert.Equal(t, req_header_cookie_parms_value, metaResult.Req_header_cookie_parms)
	assert.Equal(t, filter_err_url_value, metaResult.Filter_err_url)
	for idx, key := range metaResult.Tmp_parms {
		t.Log(idx, key)
		assert.Contains(t, req_header_cookie_parms_value, key)
	}

}
func TestFilterHandler(t *testing.T) {
	var m Middleware
	_, err1 := m.GetHandler(metaTest)
	t.Log("call m.GetHandler(),err1:", err1)
	assert.Equal(t, nil, err1)

}
