package network

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

const (
	ProblemMimetype    = "application/problem+json; charset=utf-8"
	ProblemNamespace   = "https://github.com/spikeekips/mitum/problems"
	DefaultProblemType = "others"
)

var (
	ProblemType = hint.Type("mitum-problem")
	ProblemHint = hint.NewHint(ProblemType, "v0.0.1")
)

var (
	UnknownProblem     = NewProblem(DefaultProblemType, "unknown problem occurred")
	unknownProblemJSON []byte
)

// Problem implements "Problem Details for HTTP
// APIs"<https://tools.ietf.org/html/rfc7807>.
type Problem struct {
	t      string // NOTE http problem type
	title  string
	detail string
	extra  map[string]interface{}
}

func NewProblem(t, title string) Problem {
	return Problem{t: t, title: title}
}

func NewProblemFromError(err error) Problem {
	return Problem{
		t:      DefaultProblemType,
		title:  fmt.Sprintf("%s", err),
		detail: fmt.Sprintf("%+v", err),
	}
}

func (Problem) Hint() hint.Hint {
	return ProblemHint
}

func (pr Problem) Error() string {
	return pr.title
}

func (pr Problem) Type() string {
	return pr.t
}

func (pr Problem) Title() string {
	return pr.title
}

func (pr Problem) Detail() string {
	return pr.detail
}

func (pr Problem) SetDetail(detail string) Problem {
	pr.detail = detail

	return pr
}

func (pr Problem) Extra() map[string]interface{} {
	return pr.extra
}

func (pr Problem) AddExtra(k string, v interface{}) Problem {
	if pr.extra == nil {
		pr.extra = map[string]interface{}{}
	}

	pr.extra[k] = v

	return pr
}

func WriteProblemWithError(w http.ResponseWriter, status int, err error) {
	WritePoblem(w, status, NewProblemFromError(err))
}

func WritePoblem(w http.ResponseWriter, status int, pr Problem) {
	if status == 0 {
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", ProblemMimetype)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var output []byte
	if b, err := jsonenc.Marshal(pr); err != nil {
		output = unknownProblemJSON
	} else {
		output = b
	}

	w.WriteHeader(status)
	_, _ = w.Write(output)
}

func LoadProblemFromResponse(res *http.Response) (Problem, error) {
	var pr Problem
	if m := res.Header.Get("Content-Type"); ProblemMimetype != m {
		return pr, errors.Errorf("unknown mimetype for problem, %q", m)
	}

	if i, err := ioutil.ReadAll(res.Body); err != nil {
		return pr, errors.Wrap(err, "failed to read body for loading problem")
	} else if err := jsonenc.Unmarshal(i, &pr); err != nil {
		return pr, errors.Wrap(err, "failed to unmarshal for loading problem")
	}

	return pr, nil
}
