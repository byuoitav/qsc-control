package dsp

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/byuoitav/connpool"
	"go.uber.org/zap"
)

type DSP struct {
	pool *connpool.Pool
	log  *zap.Logger
}

const _kTimeoutInSeconds = 2.0

func New(addr string, opts ...Option) *DSP {
	options := options{
		ttl:    30 * time.Second,
		delay:  500 * time.Millisecond,
		logger: zap.NewNop(),
	}

	for _, o := range opts {
		o.apply(&options)
	}

	d := &DSP{
		pool: &connpool.Pool{
			TTL:    options.ttl,
			Delay:  options.delay,
			Logger: options.logger.Sugar(),
		},
		log: options.logger,
	}

	d.pool.NewConnection = func(ctx context.Context) (net.Conn, error) {
		dial := net.Dialer{}
		conn, err := dial.DialContext(ctx, "tcp", addr+":1710")
		if err != nil {
			return nil, err
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(5 * time.Second)
		}

		conn.SetDeadline(deadline)

		buf := make([]byte, 64)
		for i := range buf {
			buf[i] = 0x01
		}
		for !bytes.Contains(buf, []byte{0x00}) {
			_, err := conn.Read(buf)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("unable to read new connection prompt: %w", err)
			}
		}

		return conn, nil
	}

	return d
}
