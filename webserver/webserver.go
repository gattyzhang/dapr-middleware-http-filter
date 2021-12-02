package main

/*--
 fasthttp web server, for unit test.


--*/

import (
	"bytes"
	//"strings"

	//"strings"
	"time"

	"encoding/json"
	"fmt"
	"log"

	//"strings"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

const (
	testForward_Normal      = "Normal"
	testForward_Redirect    = "Redirect"
	testForward_GrayRun     = "GrayRun"
	header_key_flag_forward = "Flag-Forward"
)

type respJson struct {
	Appid         string
	Token         string
	Session_Token string `json:"Session-Token"`
	Flag_Forward  string `json:"Flag-Forward"`
}

func printRequest(ctx *fasthttp.RequestCtx, method string) string {
	var sb bytes.Buffer
	sb.WriteString("\n\n")
	sb.WriteString(time.Now().String())
	sb.WriteString("\n---")
	sb.WriteString(method)
	sb.WriteString("---Page request-->")
	sb.WriteString(string(ctx.Request.RequestURI()))
	sb.WriteString("\nRequest Body:\n")
	sb.WriteString(string(ctx.Request.Body()))
	sb.WriteString("\nRequest Header:\n")
	ctx.Request.Header.VisitAll(func(k []byte, v []byte) {
		sb.WriteString(string(k))
		sb.WriteString(":")
		sb.WriteString(string(v))
		sb.WriteString(" ")
		sb.WriteString("\n")
	})
	fmt.Println(sb.String())
	return sb.String()

}
func ErrorPage(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, printRequest(ctx, "Default-error"))
}
func Rediect(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, printRequest(ctx, "Redirect"))
}
func GrayRun(ctx *fasthttp.RequestCtx) {
	reqStr := printRequest(ctx, "GrayRun")
	fmt.Fprintf(ctx, reqStr)

	ctx.Response.Header.Add("Gray-Run-Header", time.Now().String())
	var c fasthttp.Cookie
	c.SetKey("Gray-Run-Cookie")
	c.SetValue(time.Now().String())
	c.SetPath("/")
	ctx.Response.Header.SetCookie(&c)
}

func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, printRequest(ctx, "Normal filter"))

	body := ctx.Request.Body()
	var rJson respJson
	err := json.Unmarshal(body, &rJson)
	if err != nil {
		fmt.Fprintf(ctx, "failed to convert response body to a json. err: ", err)
		//return
	} else {
		fmt.Println("×××××× the json's value in body :", rJson)
		if rJson.Appid != "" {
			ctx.Response.Header.Add("Appid", rJson.Appid)
		}
		if rJson.Token != "" {
			ctx.Response.Header.Add("Token", rJson.Token)
		}
		if rJson.Session_Token != "" {
			ctx.Response.Header.Add("Session-Token", rJson.Session_Token)
		}
		if rJson.Flag_Forward != "" {
			ctx.Response.Header.Add(header_key_flag_forward, rJson.Flag_Forward)
		}

		var bt bytes.Buffer
		bt.WriteString("http://")
		bt.WriteString(string(ctx.Request.Host()))

		if rJson.Flag_Forward == testForward_Normal {
			//nothing
		} else if rJson.Flag_Forward == testForward_Redirect {
			bt.WriteString("/default-error/?testRedirect")
			ctx.Response.Header.Add("Filter_err_url", bt.String())
		} else if rJson.Flag_Forward == testForward_GrayRun {
			bt.WriteString("/grayrun/?testGrayRun")
			ctx.Response.Header.Add("Filter_err_url", bt.String())
		}
	}

	fmt.Fprintf(ctx, "<br>test fasthttp web server!", string(ctx.Request.Host()), ctx.RemoteAddr(), string(ctx.RequestURI()), ctx.QueryArgs().String())

}

func main() {
	router := fasthttprouter.New()

	router.GET("/redirect/", Rediect)
	router.POST("/redirect", Rediect)
	router.GET("/grayrun/", GrayRun)
	router.POST("/grayrun/", GrayRun)
	router.POST("/default-error/", ErrorPage)
	router.GET("/default-error/", ErrorPage)
	router.GET("/", Index)
	router.POST("/", Index)
	log.Fatal(fasthttp.ListenAndServe(":8001", router.Handler))
}
