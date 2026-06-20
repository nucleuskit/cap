package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	ContentTypeJSON = "application/json"
	ContentTypeForm = "application/x-www-form-urlencoded"
)

type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
	RetryStatus []int
}

type Request struct {
	Method      string
	URL         string
	Header      http.Header
	Body        []byte
	Timeout     time.Duration
	ContentType string
	Retry       RetryPolicy
	Metadata    map[string]string
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
	Attempt    int
}

type EventKind string

const (
	EventRequestStarted   EventKind = "request_started"
	EventRequestCompleted EventKind = "request_completed"
	EventAttemptStarted   EventKind = "attempt_started"
	EventAttemptCompleted EventKind = "attempt_completed"
)

type Event struct {
	Kind        EventKind
	Method      string
	URL         string
	Attempt     int
	MaxAttempts int
	StatusCode  int
	StartedAt   time.Time
	Duration    time.Duration
	Error       error
	Metadata    map[string]string
}

type Hook interface {
	HandleHTTPClientEvent(context.Context, Event) error
}

type HookFunc func(context.Context, Event) error

func (fn HookFunc) HandleHTTPClientEvent(ctx context.Context, event Event) error {
	if fn == nil {
		return nil
	}
	return fn(ctx, event.Clone())
}

type Client interface {
	Do(context.Context, Request) (Response, error)
}

func NewRequest(method, url string, body []byte) Request {
	return Request{Method: method, URL: url, Body: append([]byte(nil), body...)}
}

func JSON(method, url string, value any) (Request, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:      method,
		URL:         url,
		Body:        data,
		ContentType: ContentTypeJSON,
	}, nil
}

func Query(method, requestURL string, value any) (Request, error) {
	values, err := QueryValues(value)
	if err != nil {
		return Request{}, err
	}
	if len(values) == 0 {
		return NewRequest(method, requestURL, nil), nil
	}
	nextURL, err := appendQuery(requestURL, values)
	if err != nil {
		return Request{}, err
	}
	return NewRequest(method, nextURL, nil), nil
}

func QueryValues(value any) (url.Values, error) {
	return valuesFromStruct(value, "query")
}

func Form(method, url string, values url.Values) Request {
	return Request{
		Method:      method,
		URL:         url,
		Body:        []byte(values.Encode()),
		ContentType: ContentTypeForm,
	}
}

func FormStruct(method, requestURL string, value any) (Request, error) {
	values, err := FormValues(value)
	if err != nil {
		return Request{}, err
	}
	return Form(method, requestURL, values), nil
}

func FormValues(value any) (url.Values, error) {
	return valuesFromStruct(value, "form")
}

func (r Response) DecodeJSON(target any) error {
	if isNil(target) {
		return errors.New("decode json response: nil target")
	}
	if len(bytes.TrimSpace(r.Body)) == 0 {
		return errors.New("decode json response: empty body")
	}
	if err := json.Unmarshal(r.Body, target); err != nil {
		return fmt.Errorf("decode json response: invalid JSON: %w", err)
	}
	return nil
}

func (e Event) Clone() Event {
	e.Metadata = CloneMetadata(e.Metadata)
	return e
}

func CloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func (p RetryPolicy) Attempts() int {
	if p.MaxAttempts <= 0 {
		return 1
	}
	return p.MaxAttempts
}

func (p RetryPolicy) ShouldRetry(statusCode int, err error, attempt int) bool {
	if attempt >= p.Attempts() {
		return false
	}
	if len(p.RetryStatus) == 0 && statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		return false
	}
	if err != nil {
		return true
	}
	if len(p.RetryStatus) == 0 {
		return statusCode >= http.StatusInternalServerError
	}
	for _, status := range p.RetryStatus {
		if status == statusCode {
			return true
		}
	}
	return false
}

func BreakerFailure(statusCode int, err error) error {
	if err != nil {
		return err
	}
	if statusCode >= http.StatusInternalServerError {
		statusText := http.StatusText(statusCode)
		if statusText == "" {
			statusText = "server error"
		}
		return fmt.Errorf("http status %d: %s", statusCode, statusText)
	}
	return nil
}

func JoinURL(baseURL, requestURL string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return requestURL, nil
	}
	if strings.TrimSpace(requestURL) == "" {
		return baseURL, nil
	}
	parsedRequest, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	if parsedRequest.IsAbs() {
		return parsedRequest.String(), nil
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	return parsedBase.ResolveReference(parsedRequest).String(), nil
}

func CloneBody(body []byte) *bytes.Reader {
	return bytes.NewReader(append([]byte(nil), body...))
}

func appendQuery(requestURL string, values url.Values) (string, error) {
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, entries := range values {
		for _, entry := range entries {
			query.Add(key, entry)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func valuesFromStruct(value any, tagName string) (url.Values, error) {
	result := url.Values{}
	v := reflect.ValueOf(value)
	if isNil(value) {
		return result, fmt.Errorf("%s values: nil struct", tagName)
	}
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return result, fmt.Errorf("%s values: expected struct, got %s", tagName, v.Kind())
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omitEmpty, ok := tagInfo(field.Tag.Get(tagName))
		if !ok {
			continue
		}
		fieldValue := v.Field(i)
		if omitEmpty && fieldValue.IsZero() {
			continue
		}
		if err := addValue(result, name, field.Name, tagName, fieldValue); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func tagInfo(tag string) (name string, omitEmpty bool, ok bool) {
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return "", false, false
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitEmpty = true
			break
		}
	}
	return parts[0], omitEmpty, true
}

func addValue(values url.Values, name, fieldName, tagName string, value reflect.Value) error {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := addValue(values, name, fieldName, tagName, value.Index(i)); err != nil {
				return err
			}
		}
	case reflect.String:
		values.Add(name, value.String())
	case reflect.Bool:
		values.Add(name, strconv.FormatBool(value.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		values.Add(name, strconv.FormatInt(value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		values.Add(name, strconv.FormatUint(value.Uint(), 10))
	default:
		return fmt.Errorf("unsupported %s field %q: %s", tagName, fieldName, value.Type())
	}
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
