package config

import "context"

type noopConfig struct{}

func NewNoop() *noopConfig {
	return &noopConfig{}
}

func (noopConfig) Load(context.Context) (Values, error) {
	return Values{}, nil
}

func (noopConfig) Watch(ctx context.Context) (<-chan Update, error) {
	ch := make(chan Update)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}

func (noopConfig) Scan(context.Context, any) error {
	return nil
}

func (noopConfig) Sources() []Source {
	return nil
}
