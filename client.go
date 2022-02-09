package http_client_helper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type DataWrapper interface {
	DataSetter
	ErrInt
}

type DataSetter interface {
	SetData(interface{}) DataWrapper //设置 data 并返回 一个 copy
}

type ErrInt interface {
	Err() error
}

type H struct {
	c           *http.Client
	domain      string
	data        DataWrapper
	requestFunc func(*http.Request)
}

//最合理的接口用这个就行
func NewDefault(url string, requestFunc func(*http.Request)) *H {
	return New(http.DefaultClient, url, &Resp{}, requestFunc)
}

func New(c *http.Client, url string, data DataWrapper, requestFunc func(*http.Request)) *H {
	return &H{c: c, domain: url, data: data, requestFunc: requestFunc}
}

func (c *H) GetUrl(u string, p fmt.Stringer) string {
	if p == nil {
		return fmt.Sprintf("%s%s", c.domain, u)
	}
	return fmt.Sprintf("%s%s?%s", c.domain, u, p)
}

func (c *H) Do(req *http.Request, des interface{}) error {
	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)

	if err := json.Unmarshal(body, des); err != nil {
		return fmt.Errorf("unmarshal failed, req:%s, resp:%s, data:%s, err:%w", req.URL, string(body), des, err)
	}
	return nil
}

func (c *H) Post(ctx context.Context, uri string, p, des interface{}) (err error) {
	return c.DoMethod(ctx, uri, p, des, http.MethodPost)
}

func (c *H) Put(ctx context.Context, uri string, p, des interface{}) (err error) {
	return c.DoMethod(ctx, uri, p, des, http.MethodPut)
}

func (c *H) Delete(ctx context.Context, uri string, p, des interface{}) (err error) {
	return c.DoMethod(ctx, uri, p, des, http.MethodDelete)
}

func (c *H) Get(ctx context.Context, uri string, p Params, des interface{}) (err error) {
	u := c.GetUrl(uri, p)
	by, err := json.Marshal(p)
	if err != nil {
		return
	}
	b := bytes.NewReader(by)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, b)
	c.requestFunc(req)
	resp := c.data.SetData(des)
	err = c.Do(req, &resp)
	if err != nil {
		return
	}
	return resp.Err()
}

func (c *H) DoMethod(ctx context.Context, uri string, p, des interface{}, method string) (err error) {
	u := c.GetUrl(uri, nil)
	fmt.Println(u)
	by, err := json.Marshal(p)
	if err != nil {
		return
	}
	b := bytes.NewReader(by)
	req, err := http.NewRequestWithContext(ctx, method, u, b)
	c.requestFunc(req)
	resp := c.data.SetData(des)
	err = c.Do(req, &resp)
	if err != nil {
		return
	}
	return resp.Err()
}

type Params map[string]interface{}

func (p Params) String() string {
	a := &url.Values{}
	for k, v := range p {
		var r string
		switch vv := v.(type) {
		case string:
			r = vv
		case int:
			r = strconv.Itoa(vv)
		case int64:
			r = strconv.FormatInt(vv, 10)
		}
		a.Add(k, r)
	}
	return a.Encode()
}

// default Resp
type Resp struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (r Resp) SetData(i interface{}) DataWrapper {
	r.Data = i
	return &r
}

func (r Resp) Err() (err error) {
	if r.Code != 0 {
		err = fmt.Errorf("%s code: %d", r.Message, r.Code)
	}
	if e, ok := r.Data.(ErrInt); ok {
		return e.Err()
	}
	return
}
