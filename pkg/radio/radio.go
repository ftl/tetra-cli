package radio

import (
	"context"
	"io"
	"sync"
	"time"
)

type PEI interface {
	Close()
	Closed() bool
	WaitUntilClosed(ctx context.Context)
	OnDisconnect(callback func())
	AddIndication(prefix string, trailingLines int, handler func(lines []string)) error
	ClearSyntaxErrors(ctx context.Context) error
	Request(ctx context.Context, request string) ([]string, error)
	AT(ctx context.Context, request string) ([]string, error)
	ATs(ctx context.Context, requests ...string) error
}

type Initializer interface {
	Initialize(context.Context, PEI) error
}

type InitializerFunc func(context.Context, PEI) error

func (f InitializerFunc) Initialize(ctx context.Context, pei PEI) error {
	return f(ctx, pei)
}

// A LoopFunc can be executed in a separate goroutine, while a radio is connected to a PEI device.
// The LoopFunc must terminate when ctx.Done() is closed.
type LoopFunc func(context.Context, PEI)

// Radio provides access to a PEI device on a higher level of abstration.
// It allows to define a custom initializer for the PEI device.
// It allows to run a loop function in context of the PEI device.
// It also implements the PEI interface.
type Radio struct {
	pei            PEI
	tracePEIWriter io.Writer

	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopGroup  *sync.WaitGroup

	scanInterval time.Duration
	scanTimeout  time.Duration
}

// Open the radio using the given PEI instance. The optionally given initializer is invoked
// to initialize the radio.
func Open(ctx context.Context, pei PEI, initializer Initializer) (*Radio, error) {
	loopCtx, loopCancel := context.WithCancel(context.Background())

	result := &Radio{
		pei: pei,

		loopCtx:    loopCtx,
		loopCancel: loopCancel,
		loopGroup:  new(sync.WaitGroup),
	}
	err := result.initialize(ctx, initializer)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Radio) initialize(ctx context.Context, initializer Initializer) error {
	err := r.pei.ClearSyntaxErrors(ctx)
	if err != nil {
		return err
	}

	// initialize the PEI
	err = r.pei.ATs(ctx,
		"ATZ",
		"ATE0",
		"AT+CSCS=8859-1",
	)
	if err != nil {
		return err
	}

	if initializer == nil {
		return nil
	}
	return initializer.Initialize(ctx, r.pei)
}

// Connected indicates if this radio is currently connected to a PEI device.
func (r *Radio) Connected() bool {
	return r.pei != nil && !r.pei.Closed()
}

// Close the connection to the radio. All running loops are terminated before the connection
// to the radio is shut down.
func (r *Radio) Close() {
	if !r.Connected() {
		return
	}

	// stop the running loops and wait until they are stopped
	r.loopCancel()
	r.loopGroup.Wait()

	// reset the PEI to defaults
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	r.pei.AT(shutdownCtx, "ATZ")
}

func (r *Radio) Closed() bool {
	return r.pei.Closed()
}

func (r *Radio) WaitUntilClosed(ctx context.Context) {
	r.pei.WaitUntilClosed(ctx)
}

func (r *Radio) OnDisconnect(callback func()) {
	r.pei.OnDisconnect(callback)
}

func (r *Radio) AddIndication(prefix string, trailingLines int, handler func(lines []string)) error {
	return r.pei.AddIndication(prefix, trailingLines, handler)
}

func (r *Radio) ClearSyntaxErrors(ctx context.Context) error {
	return r.pei.ClearSyntaxErrors(ctx)
}

func (r *Radio) Request(ctx context.Context, request string) ([]string, error) {
	return r.Request(ctx, request)
}

func (r *Radio) AT(ctx context.Context, request string) ([]string, error) {
	return r.AT(ctx, request)
}

func (r *Radio) ATs(ctx context.Context, requests ...string) error {
	return r.ATs(ctx, requests...)
}

// RunLoop executeds the given loop function in a separate goroutine while this radio is
// connected to a PEI device.
func (r *Radio) RunLoop(loop LoopFunc) {
	if !r.Connected() {
		return
	}

	r.loopGroup.Go(func() {
		loop(r.loopCtx, r.pei)
	})
}
