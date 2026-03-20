package git

import "context"

type recordingRunner struct {
	outputs  map[string]string
	stderr   map[string]string
	errors   map[string]error
	commands map[string]struct{}
}

func (f *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	if f.commands == nil {
		f.commands = map[string]struct{}{}
	}
	k := key(append([]string{name}, args...)...)
	f.commands[k] = struct{}{}
	if err, ok := f.errors[k]; ok {
		return nil, []byte(f.stderr[k]), err
	}
	out := []byte(f.outputs[k])
	errOut := []byte(f.stderr[k])
	if out != nil || errOut != nil {
		return out, errOut, nil
	}
	return nil, nil, nil
}
