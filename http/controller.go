package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zloevil/jet"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Error is HTTP error object returning to clients in case of error
type Error struct {
	Code    string                 `json:"code,omitempty"`    // Code is error code provided by error producer
	Type    string                 `json:"type,omitempty"`    // Type is error type (panic, system, business)
	Message string                 `json:"message"`           // Message is error description
	Details map[string]interface{} `json:"details,omitempty"` // Details is additional info provided by error producer
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%s", e.Code, e.Message)
}

const (
	Me = "me" // Me can be used in URL whenever userId is expected. When encountered, userId from the session context is used
)

var EmptyOkResponse = struct {
	Status string `json:"status"`
}{
	Status: "OK",
}

// Controller is a base controller interface
type Controller interface {
	// MyUser returns true if current user requests his own data
	MyUser(ctx context.Context, r *http.Request) (bool, error)
	// HasRoles returns function which checks if current login has list of roles
	HasRoles(roles ...string) func(ctx context.Context, r *http.Request) (bool, error)
}

// BaseController is a base controller implementation
type BaseController struct {
	Logger jet.CLoggerFunc
}

var MediaContentTypes = [...]string{
	"image/jpeg",
	"image/png",
	"image/bmp",
	"image/gif",
	"image/tiff",
	"video/avi",
	"video/mpeg",
	"video/mp4",
	"audio/mpeg",
	"audio/wav",
}

type ResponseContentOpts struct {
	Filename     string
	ContentType  string
	ContentSize  int
	Download     bool
	ModifiedTime time.Time
}

func (c *BaseController) RespondContent(w http.ResponseWriter, r *http.Request, opts ResponseContentOpts, file []byte) {

	w.Header().Set("Cache-Control", "private, no-cache")

	if opts.ContentSize > 0 {
		contentSizeStr := strconv.Itoa(opts.ContentSize)
		w.Header().Set("Content-Length", contentSizeStr)
	}

	if opts.ContentType == "" {
		opts.ContentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", opts.ContentType)

	if !opts.Download {
		isMedia := false
		for _, mct := range MediaContentTypes {
			if strings.HasPrefix(opts.ContentType, mct) {
				isMedia = true
				break
			}
		}
		opts.Download = !isMedia
	}

	if opts.Download {
		w.Header().Set("Content-Disposition", "attachment;filename=\""+opts.Filename+"\"; filename*=UTF-8''"+opts.Filename)
	} else {
		w.Header().Set("Content-Disposition", "inline;filename=\""+opts.Filename+"\"; filename*=UTF-8''"+opts.Filename)
	}

	http.ServeContent(w, r, opts.Filename, opts.ModifiedTime, bytes.NewReader(file))

}

// GetUploadFileMultipartContent it parse body for multipart content disposition
// it expects the only one part with the following structure:
// -----------------------------4562559108110960722260982980
// Content-Disposition: form-data; name="files"; filename="my-file.jpg"
// Content-Type: image/jpeg
// ....
// .....
func (c *BaseController) GetUploadFileMultipartContent(ctx context.Context, r *http.Request) (io.Reader, string, error) {

	// parse form
	if r.Form == nil {
		err := r.ParseForm()
		if err != nil {
			return nil, "", ErrHttpMultipartParseForm(err, ctx)
		}
	}
	if r.ContentLength == 0 {
		return nil, "", ErrHttpMultipartEmptyContent(ctx)
	}

	// get content type from header
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil, "", ErrHttpMultipartNotMultipart(ctx)
	}

	// parse mime type
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", ErrHttpMultipartParseMediaType(err, ctx)
	}
	if mediaType != "multipart/form-data" {
		return nil, "", ErrHttpMultipartWrongMediaType(ctx, mediaType)
	}

	// identify boundary
	boundary, ok := params["boundary"]
	if !ok {
		return nil, "", ErrHttpMultipartMissingBoundary(ctx)
	}

	// create a new reader
	mr := multipart.NewReader(r.Body, boundary)

	// go through all parts
	for {

		// take next part
		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				// if we get here, we haven't found any useful parts, so it's wrong format
				return nil, "", ErrHttpMultipartEofReached(ctx)
			} else {
				return nil, "", ErrHttpMultipartNext(err, ctx)
			}
		}

		// check found part
		if part.FormName() == "file" {
			filename := part.FileName()
			if filename == "" {
				return nil, "", ErrHttpMultipartFilename(ctx)
			}
			// return first part
			return part, filename, nil
		} else {
			return nil, "", ErrHttpMultipartFormNameFileExpected(ctx)
		}

	}
}

func (c *BaseController) RespondJson(w http.ResponseWriter, httpStatus int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(response)
}

func (c *BaseController) RespondError(w http.ResponseWriter, err error) {

	httpErr := &Error{}
	httpStatus := http.StatusInternalServerError

	// check if this is an app error
	if appErr, ok := jet.IsAppErr(err); ok {
		httpErr.Code = appErr.Code()
		httpErr.Message = appErr.Message()
		httpErr.Type = appErr.Type()
		httpErr.Details = appErr.Fields()
		if httpSt := appErr.HttpStatus(); httpSt != nil {
			httpStatus = int(*httpSt)
		}
	} else {
		httpErr.Message = err.Error()
	}
	if c.Logger != nil {
		c.Logger().Cmp("api").Pr("rest").E(err).St().Err()
	}
	c.RespondJson(w, httpStatus, httpErr)
}

func (c *BaseController) RespondWithStatus(w http.ResponseWriter, status int, payload interface{}) {
	c.RespondJson(w, status, payload)
}

func (c *BaseController) RespondOK(w http.ResponseWriter, payload interface{}) {
	c.RespondJson(w, http.StatusOK, payload)
}

func (c *BaseController) RespondCreated(w http.ResponseWriter, payload interface{}) {
	c.RespondJson(w, http.StatusCreated, payload)
}

func (c *BaseController) DecodeRequest(ctx context.Context, r *http.Request, body interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(body); err != nil {
		return ErrHttpDecodeRequest(err, ctx)
	}
	return nil
}

func (c *BaseController) Var(ctx context.Context, r *http.Request, varName string, allowEmpty bool) (string, error) {
	if val, ok := mux.Vars(r)[varName]; ok {
		if !allowEmpty && val == "" {
			return "", ErrHttpUrlVarEmpty(ctx, varName)
		}
		return val, nil
	} else {
		return "", ErrHttpUrlVar(ctx, varName)
	}
}

func (c *BaseController) VarUUID(ctx context.Context, r *http.Request, varName string, allowEmpty bool) (string, error) {
	valStr, err := c.Var(ctx, r, varName, allowEmpty)
	if err != nil {
		return "", err
	}
	if allowEmpty && valStr == "" {
		return "", nil
	}
	err = jet.ValidateUUIDs(valStr)
	if err != nil {
		return "", ErrHttpUrlVarInvalidUUID(ctx, varName)
	}
	return valStr, nil
}

func (c *BaseController) CurrentUser(ctx context.Context) (uid string, un string, err error) {
	if rCtx, ok := jet.Request(ctx); ok {
		if rCtx.Un != "" && rCtx.Uid != "" {
			return rCtx.Uid, rCtx.Un, nil
		} else {
			return "", "", ErrHttpCurrentUser(ctx)
		}
	} else {
		return "", "", ErrHttpCurrentUser(ctx)
	}
}

func (c *BaseController) UserIdVar(ctx context.Context, r *http.Request, varName string) (string, error) {
	val, err := c.Var(ctx, r, varName, false)
	if err != nil {
		return "", err
	}
	// if current user
	if val == Me {
		if uid, _, err := c.CurrentUser(ctx); err != nil {
			return "", err
		} else {
			return uid, nil
		}
	}
	// validate UUID
	err = jet.ValidateUUIDs(val)
	if err != nil {
		return "", ErrHttpUrlVarInvalidUUID(ctx, val)
	}
	return val, nil
}

func (c *BaseController) UserNameVar(ctx context.Context, r *http.Request, varName string) (string, error) {
	val, err := c.Var(ctx, r, varName, false)
	if err != nil {
		return "", err
	}
	if val == Me {
		if _, un, err := c.CurrentUser(ctx); err != nil {
			return "", err
		} else {
			return un, nil
		}
	}
	return val, nil
}

func (c *BaseController) FormVal(ctx context.Context, r *http.Request, name string, allowEmpty bool) (string, error) {
	val := r.FormValue(name)
	if !allowEmpty && val == "" {
		return "", ErrHttpUrlFormVarEmpty(ctx, name)
	}
	return val, nil
}

func (c *BaseController) FormValList(ctx context.Context, r *http.Request, name string, allowEmpty bool) ([]string, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return nil, err
	}
	if allowEmpty && valStr == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(valStr), ","), nil
}

func (c *BaseController) FormValUUID(ctx context.Context, r *http.Request, name string, allowEmpty bool) (string, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return "", err
	}
	if allowEmpty && valStr == "" {
		return "", nil
	}
	err = jet.ValidateUUIDs(valStr)
	if err != nil {
		return "", ErrHttpUrlVarInvalidUUID(ctx, name)
	}
	return valStr, nil
}

func (c *BaseController) FormValInt(ctx context.Context, r *http.Request, name string, allowEmpty bool) (*int, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return nil, err
	}
	if allowEmpty && valStr == "" {
		return nil, nil
	}
	valInt, err := strconv.Atoi(valStr)
	if err != nil {
		return nil, ErrHttpUrlFormVarNotInt(err, ctx, name)
	}
	return &valInt, nil
}

func (c *BaseController) FormValFloat(ctx context.Context, r *http.Request, name string, allowEmpty bool) (*float64, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return nil, err
	}
	if allowEmpty && valStr == "" {
		return nil, nil
	}
	valFloat, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return nil, ErrHttpUrlFormVarNotFloat(err, ctx, name)
	}
	return &valFloat, nil
}

func (c *BaseController) FormValMetadata(ctx context.Context, r *http.Request, name string, allowEmpty bool) (map[string]string, error) {
	var metadata map[string]string
	if err := c.FormValJson(ctx, r, name, allowEmpty, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func (c *BaseController) FormValJson(ctx context.Context, r *http.Request, name string, allowEmpty bool, data interface{}) error {
	jsonStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return err
	}
	if jsonStr != "" {
		if err = json.Unmarshal([]byte(jsonStr), &data); err != nil {
			return ErrHttpFileHeaderInvalidJson(ctx, name)
		}
	}
	return nil
}

func (c *BaseController) FormValBool(ctx context.Context, r *http.Request, name string, allowEmpty bool) (*bool, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return nil, err
	}
	if allowEmpty && valStr == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(valStr)
	if err != nil {
		return nil, ErrHttpUrlFormVarNotBool(err, ctx, name)
	}
	return &b, nil
}

// FormValTime parses URL form value and checks for time in RFC3339 format(UTC)
func (c *BaseController) FormValTime(ctx context.Context, r *http.Request, name string, allowEmpty bool) (*time.Time, error) {
	valStr, err := c.FormVal(ctx, r, name, allowEmpty)
	if err != nil {
		return nil, err
	}
	if allowEmpty && valStr == "" {
		return nil, nil
	}
	valTime, err := time.Parse(time.RFC3339, valStr)
	if err != nil {
		return nil, ErrHttpUrlFormVarNotTime(err, ctx, name)
	}
	return &valTime, nil
}

func (c *BaseController) FileHeader(ctx context.Context, h *multipart.FileHeader, name string, allowEmpty bool) (string, error) {
	header := h.Header.Get(name)
	if !allowEmpty && header == "" {
		return "", ErrHttpFileHeaderEmpty(ctx, name)
	}
	return header, nil
}

func (c *BaseController) FileHeaderUUID(ctx context.Context, h *multipart.FileHeader, name string, allowEmpty bool) (string, error) {
	header := h.Header.Get(name)
	if !allowEmpty && header == "" {
		return "", ErrHttpFileHeaderEmpty(ctx, name)
	}
	err := jet.ValidateUUIDs(header)
	if err != nil {
		return "", ErrHttpFileHeaderInvalidUUID(ctx, name)
	}
	return header, nil
}

func (c *BaseController) FileHeaderJson(ctx context.Context, h *multipart.FileHeader, name string, allowEmpty bool, data interface{}) error {
	jsonStr, err := c.FileHeader(ctx, h, name, allowEmpty)
	if err != nil {
		return err
	}
	if jsonStr != "" {
		if err = json.Unmarshal([]byte(jsonStr), &data); err != nil {
			return ErrHttpFileHeaderInvalidJson(ctx, name)
		}
	}
	return nil
}

func (c *BaseController) FileHeaderMetadata(ctx context.Context, h *multipart.FileHeader, name string, allowEmpty bool) (map[string]string, error) {
	var m map[string]string
	if err := c.FileHeaderJson(ctx, h, name, allowEmpty, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// FormPaging parses URL form value for paging params. Allows specifying max page size
func (c *BaseController) FormPaging(ctx context.Context, r *http.Request, maxPageSize *int) (size *int, index *int, err error) {
	size, err = c.FormValInt(ctx, r, "size", true)
	if err != nil {
		return
	}
	index, err = c.FormValInt(ctx, r, "index", true)
	if err != nil {
		return
	}
	if maxPageSize != nil && size != nil && *size > *maxPageSize {
		err = ErrHttpUrlMaxPageSizeExceeded(ctx, *maxPageSize)
		return
	}
	return
}

// FormSort parses URL form sorting value for paging params
func (c *BaseController) FormSort(ctx context.Context, r *http.Request) ([]*jet.SortRequest, error) {
	if r.Form == nil {
		return nil, nil
	}
	var res []*jet.SortRequest
	for _, value := range r.Form["sort"] {
		sort := strings.Fields(strings.TrimSpace(value))
		if len(sort) == 0 || len(sort) > 2 {
			return nil, ErrHttpUrlInvalidPagingSortSet(ctx)
		}
		rq := &jet.SortRequest{Field: sort[0]}
		if len(sort) == 2 {
			rq.Desc = strings.ToLower(sort[1]) == "desc"
		}
		res = append(res, rq)
	}
	return res, nil
}

// MyUser returns true if current user requests his own data
func (c *BaseController) MyUser(ctx context.Context, r *http.Request) (bool, error) {
	currentUid, _, err := c.CurrentUser(ctx)
	if err != nil {
		return false, err
	}
	uid, err := c.UserIdVar(ctx, r, "userId")
	return currentUid == uid && err == nil, nil
}

// HasRoles returns true if a current user has the requested roles
func (c *BaseController) HasRoles(roles ...string) func(ctx context.Context, r *http.Request) (bool, error) {
	return func(ctx context.Context, r *http.Request) (bool, error) {
		if len(roles) == 0 {
			return true, nil
		}
		if rCtx, ok := jet.Request(ctx); ok && rCtx != nil && len(rCtx.Roles) > 0 {
			r := jet.Strings(roles)
			return len(r.Intersect(rCtx.Roles)) == len(r), nil
		}
		return false, nil
	}
}

// DecodeRequest is a generic func allowing to unmarshal request
func DecodeRequest[T any](ctx context.Context, r *http.Request) (*T, error) {
	decoder := json.NewDecoder(r.Body)
	var v T
	if err := decoder.Decode(&v); err != nil {
		return nil, ErrHttpDecodeRequest(err, ctx)
	}
	return &v, nil
}
