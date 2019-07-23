package common

import "golang.org/x/xerrors"

func SetStringMap(args ...interface{}) (map[string]interface{}, error) {
	if len(args)%2 != 0 {
		return nil, xerrors.Errorf("invalid number of args: %v", len(args))
	}

	r := map[string]interface{}{}
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			return nil, xerrors.Errorf("key is not string: %T", args[i])
		}

		r[k] = args[i+1]
	}

	return r, nil
}
