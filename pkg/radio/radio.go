package radio

import "context"

type PEI interface {
	Close()
	Closed() bool
	WaitUntilClosed(ctx context.Context)
	AddIndication(prefix string, trailingLines int, handler func(lines []string)) error
	ClearSyntaxErrors(ctx context.Context) error
	Request(ctx context.Context, request string) ([]string, error)
	AT(ctx context.Context, request string) ([]string, error)
	ATs(ctx context.Context, requests ...string) error
}
