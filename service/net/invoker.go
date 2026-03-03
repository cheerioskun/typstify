package net

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type bodyDecoder func(body io.Reader, resp interface{}) error

type NetworkError struct {
	StatusCode int    `json:"code"`
	Message    string `json:"message"`
}

type HttpInvoker struct {
	httpClient *http.Client
	req        *http.Request
	url        string
	headers    map[string][]string
}

func (e NetworkError) Error() string {
	return fmt.Sprintf("[network error] code: %d, message: %s", e.StatusCode, e.Message)
}

func NewInvoker(client *http.Client, url string) *HttpInvoker {
	return &HttpInvoker{
		httpClient: client,
		url:        url,
		headers:    make(map[string][]string),
	}
}

func (c *HttpInvoker) Get(resp interface{}) *HttpInvoker {
	return c.Build("GET", c.url, resp)
}

func (c *HttpInvoker) Post(req interface{}) *HttpInvoker {
	return c.Build("POST", c.url, req)
}

func (c *HttpInvoker) Put(req interface{}) *HttpInvoker {
	return c.Build("PUT", c.url, req)
}

func (c *HttpInvoker) Delete(req interface{}) *HttpInvoker {
	return c.Build("DELETE", c.url, req)
}

func (c *HttpInvoker) PostFile(fieldName string, filename string, file io.Reader) error {
	_, err := c.buildMultipart("POST", c.url, fieldName, filename, file)
	return err
}

func (c *HttpInvoker) Build(method string, url string, body interface{}) *HttpInvoker {
	var reqBody io.Reader = nil
	if body != nil {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)
		reqBody = &buf
	}

	request, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		panic(err)
	}

	request.Header.Set("Content-Type", "application/json")
	for key, values := range c.headers {
		for _, v := range values {
			request.Header.Add(key, v)
		}
	}

	c.req = request
	return c
}

func (c *HttpInvoker) buildMultipart(method string, url string, fieldName string, filename string, file io.Reader) (*HttpInvoker, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())

	for key, values := range c.headers {
		for _, v := range values {
			request.Header.Add(key, v)
		}
	}

	c.req = request
	return c, nil
}

func (c *HttpInvoker) SetHeader(key string, value string) *HttpInvoker {
	if key == "" {
		return c
	}

	c.headers[key] = append(c.headers[key], value)

	if c.req != nil {
		c.req.Header.Add(key, value)
	}
	return c
}

func (c *HttpInvoker) SetHeaders(headers map[string]string) *HttpInvoker {
	if headers == nil {
		return c
	}

	for k, v := range headers {
		c.SetHeader(k, v)
	}

	return c
}

func (c *HttpInvoker) SetAuthorization(apiToken string) *HttpInvoker {
	if len(apiToken) > 0 {
		c.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	}
	return c
}

func (c *HttpInvoker) Decode(resp interface{}) error {
	return c.decode(resp, decodeJson)
}

func (c *HttpInvoker) ReadAsFile(resp interface{}) error {
	return c.decode(resp, decodeFile)
}

func (c *HttpInvoker) decode(resp interface{}, decoder bodyDecoder) error {
	if c.req == nil {
		panic("Client not initialized")
	}

	response, err := c.httpClient.Do(c.req)
	if err != nil {
		// url.Error
		return err
	}

	defer response.Body.Close()

	contentType := response.Header.Get("Content-Type")
	isJsonResponse := strings.HasPrefix(contentType, "application/json")

	if response.StatusCode >= http.StatusBadRequest {
		// 4xx-5xx error
		// resp should be of a pointer type
		var httpErr = NetworkError{StatusCode: response.StatusCode}

		if !isJsonResponse {
			return httpErr
		} else {
			e := json.NewDecoder(response.Body).Decode(&httpErr)
			if e != nil {
				return err
			}
			if httpErr.StatusCode <= 0 {
				httpErr.StatusCode = response.StatusCode
			}
		}

		return httpErr
	}

	if resp == nil {
		return nil
	}

	if decoder != nil {
		return decoder(response.Body, resp)
	}

	return nil
}

func decodeJson(body io.Reader, resp interface{}) error {
	return json.NewDecoder(body).Decode(resp)
}

func decodeFile(body io.Reader, resp interface{}) error {
	var err error
	switch resp.(type) {
	case []byte:
		resp, err = io.ReadAll(body)
		if err != nil {
			return err
		}
	case io.Writer:
		w := resp.(io.Writer)
		_, err = io.Copy(w, body)
		return err
	}

	return nil
}
