package contestlib

import (
	"bytes"
	"html/template"
	"sync"

	"github.com/spikeekips/mitum/util"
	"gopkg.in/yaml.v3"
)

type Vars struct {
	sync.RWMutex
	m map[string]interface{}
}

func NewVars(m map[string]interface{}) *Vars {
	if m == nil {
		m = map[string]interface{}{}
	}

	return &Vars{m: m}
}

func (vs *Vars) Set(k string, v interface{}) *Vars {
	vs.Lock()
	defer vs.Unlock()

	vs.m[k] = v

	return vs
}

func (vs *Vars) Format(t *template.Template) string {
	vs.RLock()
	defer vs.RUnlock()

	var bf bytes.Buffer
	if err := t.Execute(&bf, vs.m); err != nil {
		return ""
	}

	return bf.String()
}

func (vs *Vars) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(vs.m)
}

func (vs *Vars) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]interface{}
	if err := value.Decode(&m); err != nil {
		return err
	}

	*vs = *NewVars(m)

	return nil
}
