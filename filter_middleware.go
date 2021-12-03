// ------------------------------------------------------------
// Copyright (c) Gatty.
// Licensed under the MIT License.
// ------------------------------------------------------------

package filter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dapr/components-contrib/middleware"
	"github.com/valyala/fasthttp"
)

// Metadata is the oFilter middleware config.
type oFilterMiddlewareMetadata struct {
	Req_header_cookie_parms string `json:"req_header_cookie_parms"`
	Filter_url              string `json:"filter_url"`
	Filter_err_url          string `json:"filter_err_url"`
	Filter_skip_uir         string `json:"filter_skip_uri"`
	Tmp_parms               []string
	Tmp_skip_uri            []string
	Tmp_cookie_flag         bool
}

// NewOAuth2Middleware returns a new oAuth2 middleware.
func NewOFilterMiddleware() *Middleware {
	return &Middleware{}
}

// Middleware is a oFilter middleware to call specific filter service.
type Middleware struct{}

const (
	filter_url              = "Filter_url"
	req_header_cookie_parms = "Req_header_cookie_parms"
	filter_err_url          = "Filter_err_url"

	header_key_cookie_check_key      = "Appid"
	header_key_flag_forward          = "Flag-Forward"
	header_key_flag_forward_Normal   = "Normal"
	header_key_flag_forward_Redirect = "Redirect"
	header_key_flag_forward_GrayRun  = "GrayRun"

	header_key_host       = "Host"
	header_key_user_agent = "User-Agent"
	header_key_referer    = "Referer"
	header_key_xforward   = "X-Forwarded-For"
	header_key_uri        = "Uri"
	header_key_remote_ip  = "Remote-Ip"

	uri      = "request-uri"
	host     = "request-host"
	remoteIp = "remote-ip"
)

func (m *Middleware) findItem(item []byte, list []string) bool {
	if list == nil || item == nil {
		return false
	}
	for _, k := range list {
		if bytes.Equal(item, []byte(k)) {
			return true
		}
	}
	return false
}
func (m *Middleware) skipUri(uri []byte, list []string) bool {
	if uri == nil || list == nil {
		return false
	}

	for _, k := range list {
		if strings.HasPrefix(k, string(uri)) {
			return true
		}
	}

	return false
}

func (m *Middleware) grayRunHandler(ctx *fasthttp.RequestCtx, val_filter_err_url []byte) error {
	reqGray := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(reqGray) // 用完需要释放资源
	reqGray.Header.SetContentType("application/json")
	reqGray.Header.SetMethod("POST")
	reqGray.SetRequestURI(string(val_filter_err_url))
	//duplicaate the request's header and body
	reqGray.SetBody(ctx.Request.Body())
	ctx.Request.Header.VisitAll(func(k []byte, v []byte) {
		fmt.Println("----1.5.1 copy reqeust ", string(k), string(v))
		reqGray.Header.Add(string(k), string(v))
	})

	respGray := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(respGray) // 用完需要释放资源

	if errGray := fasthttp.Do(reqGray, respGray); errGray != nil {
		return errGray
	}
	fmt.Println("gatty---3.4-- GrayRun ", string(respGray.Body()))

	//copy back the returns
	ctx.Response.SetBody(respGray.Body())
	respGray.Header.VisitAll(func(k []byte, v []byte) {
		//fmt.Println("---- 1.5.2 copy back the response ", string(k), string(v))
		ctx.Response.Header.Add(string(k), string(v))
	})

	return nil
}

func (m *Middleware) writeBackHeaders(meta *oFilterMiddlewareMetadata, ctx *fasthttp.RequestCtx, filterResp *fasthttp.Response) {
	// 覆盖地回填指定的内容
	filterResp.Header.VisitAll(func(k []byte, v []byte) {
		if m.findItem(k, meta.Tmp_parms) == true {
			newV := string(v) + "-changed"
			if meta.Tmp_cookie_flag == false {
				ctx.Response.Header.Del(string(k))
				ctx.Response.Header.Add(string(k), newV)
			} else {
				ctx.Response.Header.DelCookie(string(k))
				//default expires: session
				var c fasthttp.Cookie
				c.SetKey(string(k))
				c.SetValue(newV)
				c.SetPath("/")
				ctx.Response.Header.SetCookie(&c)
			}
		}
	})
}

// GetHandler retruns the HTTP handler provided by the middleware.
func (m *Middleware) GetHandler(metadata middleware.Metadata) (func(h fasthttp.RequestHandler) fasthttp.RequestHandler, error) {
	//fmt.Println("gatty---1-- middleware.Metadata ", metadata)
	meta, err := m.getNativeMetadata(metadata)
	if err != nil {
		return nil, err
	}
	fmt.Println("gatty---2-- after formatting, meta: ", meta)

	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {

		return func(ctx *fasthttp.RequestCtx) {
			fmt.Println("---gatty.3.6--- original request body:", string(ctx.Request.Body()))
			// 0. 如果是跳过的URI，就直接往后处理。
			uri := ctx.RequestURI()
			if m.skipUri(uri, meta.Tmp_skip_uri) == true {
				h(ctx)
				return
			}

			// 1 整理需要提交到filter的参数。
			var mParms map[string]string
			mParms = make(map[string]string)

			// 1.1 meta配置指定要的参数的生成。
			//for cookies, H5 only
			meta.Tmp_cookie_flag = false
			if ctx.Request.Header.Cookie(header_key_cookie_check_key) != nil {
				meta.Tmp_cookie_flag = true
				//3. get the specific item from Cookie first.
				var cookieV []byte
				for _, k := range meta.Tmp_parms {
					cookieV = ctx.Request.Header.Cookie(k)
					if cookieV != nil {
						mParms[k] = string(cookieV)
					}
				}
			}

			//for headers
			header_key_cookie_key := []byte("Cookie")
			ctx.Request.Header.VisitAll(func(k []byte, v []byte) {
				fmt.Println("---- header>", string(k), string(v))
				if bytes.Equal(header_key_cookie_key, k) {
					if meta.Tmp_cookie_flag == true {
						//1. skip the cookie header
						fmt.Println("---- pass through the cookie item:"+string(k), ",", string(v))
					}
					//2. next the else...get the normal item
				} else if bytes.Equal([]byte(header_key_referer), k) {
					mParms[header_key_referer] = string(v)
				} else if bytes.Equal([]byte(header_key_xforward), k) {
					mParms[header_key_xforward] = string(v)
				} else if bytes.Equal([]byte(header_key_user_agent), k) {
					mParms[header_key_user_agent] = string(v)
				} else if bytes.Equal([]byte(header_key_host), k) {
					mParms[header_key_host] = string(v)
				} else if meta.Tmp_cookie_flag == false && m.findItem(k, meta.Tmp_parms) == true {
					//3. get the specific item.
					mParms[string(k)] = string(v)
				}
			})

			// 1.2 固定的参数的生成。
			mParms[header_key_uri] = string(uri)
			mParms[header_key_remote_ip] = ctx.RemoteAddr().String()
			//mParms[header_key_user_agent] = string(ctx.Request.Header.UserAgent())
			//mParms[header_key_host] = string(ctx.Host())

			// 1.3 调用filter的实现微服务.
			req := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(req) // 用完需要释放资源
			req.Header.SetContentType("application/json")
			req.Header.SetMethod("POST")
			req.SetRequestURI(meta.Filter_url)

			jsonParms, err := json.Marshal(mParms)
			if err != nil {
				fmt.Println("生成filter请求的参数时异常-json。", err)
				return
			}
			//requestBody := []byte(jsonParms)
			req.SetBody(jsonParms)

			resp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(resp) // 用完需要释放资源

			if err := fasthttp.Do(req, resp); err != nil {
				fmt.Println("请求filter微服务异常-fasthttp.do。", err.Error())
				return
			}
			//1.4 处理filter的结果
			val_filter_err_url := resp.Header.Peek(filter_err_url)
			val_flag_forward := string(resp.Header.Peek(header_key_flag_forward))
			fmt.Println("---- 1.4  filter_err_url:", string(val_filter_err_url), ",flag-forward:", val_flag_forward)

			if val_flag_forward == header_key_flag_forward_Redirect {
				// redirect to new location now.
				fmt.Println("gatty---3.3-- REDIRECT(no filter output attached.) ", string(resp.Body()))
				if val_filter_err_url == nil {
					ctx.Redirect(string(meta.Filter_err_url), 302)
				} else {
					ctx.Redirect(string(val_filter_err_url), 302)
				}
				return
			} else if val_flag_forward == header_key_flag_forward_GrayRun {
				//gray run
				errGray := m.grayRunHandler(ctx, val_filter_err_url)
				if errGray != nil {
					fmt.Println("请求Gray Run微服务异常-fasthttp.do。", errGray.Error())
					return
				}
				//回写filter指定的header
				m.writeBackHeaders(meta, ctx, resp)
				return
			} else {
				// normal
				m.writeBackHeaders(meta, ctx, resp)
				fmt.Println("gatty---3.2-- NORMAL ", string(resp.Body()))
				// go back to the business process...
				h(ctx)
			}
			// end of the filter's work.
		}
	}, nil
}

func (m *Middleware) getNativeMetadata(metadata middleware.Metadata) (*oFilterMiddlewareMetadata, error) {
	b, err := json.Marshal(metadata.Properties)
	if err != nil {
		return nil, err
	}

	var middlewareMetadata oFilterMiddlewareMetadata
	err = json.Unmarshal(b, &middlewareMetadata)
	if err != nil {
		return nil, err
	}

	if middlewareMetadata.Filter_url == "" ||
		middlewareMetadata.Req_header_cookie_parms == "" {
		return nil, errors.New("filter的配置文件配置项不完整。")
	}
	if strings.Index(middlewareMetadata.Req_header_cookie_parms, "_") > 0 {
		return nil, errors.New("filter的配置Req_header_cookie_parms包含了不兼容的字符_。")
	}

	//middlewareMetadata.Tmp_err_code = ""
	middlewareMetadata.Tmp_cookie_flag = false
	middlewareMetadata.Tmp_parms = strings.Split(middlewareMetadata.Req_header_cookie_parms, ",")
	middlewareMetadata.Tmp_skip_uri = strings.Split(middlewareMetadata.Filter_skip_uir, ",")

	return &middlewareMetadata, nil
}
