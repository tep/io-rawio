// Copyright © 2018 Timothy E. Peoples <eng@toolman.org>
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package rawio provides utilities for performing cancellable I/O operations.
package rawio // import "toolman.org/io/rawio"

import (
	"context"
	"io"
	"sync"

	"golang.org/x/sys/unix"
)

// A RawReader performs read operations that may be cancelled -- either by
// closing the reader from a separate goroutine or, through its ReadContext
// method, if the context is cancelled.
type RawReader struct {
	fd   int
	open bool
	sync.RWMutex
	threadLatch
}

// NewRawReader creates a new RawReader from fd or nil and an error if the
// RawReader cannot be created.
func NewRawReader(fd int) (*RawReader, error) {
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	return &RawReader{fd: fd, open: true}, nil
}

// Close will close the RawReader.  Any concurrently blocked read operations
// will return io.EOF.
// Close implements io.Closer.
func (r *RawReader) Close() error {
	r.Lock()
	defer r.Unlock()

	if !r.open {
		return nil
	}

	if err := unix.Close(r.fd); err != nil {
		return err
	}

	if err := r.notify(NotifySignal); err != nil {
		return err
	}

	r.fd = -1
	r.open = false

	return nil
}

// Fd returns the file descriptor assocated with this RawReader.
func (r *RawReader) Fd() int {
	return r.fd
}

func (r *RawReader) closed() bool {
	r.Lock()
	defer r.Unlock()
	return !r.open
}

// Read implements io.Reader. If Read is blocked and r is closed in a separate
// goroutine, Read will return 0 and io.EOF.
func (r *RawReader) Read(p []byte) (int, error) {
	return r.readContext(nil, p)
}

// ReadContext behaves similar to Read but may also be cancelled through the
// given context. If ctx is cancelled, then ReadContext will return ctx.Err().
func (r *RawReader) ReadContext(ctx context.Context, p []byte) (int, error) {
	return r.readContext(ctx, p)
}

func (r *RawReader) readContext(ctx context.Context, p []byte) (int, error) {
	done := make(chan struct{})
	defer close(done)

	var cancelled bool

	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				cancelled = true
				r.notify(NotifySignal)

			case <-done:
			}
		}()
	}

	ep, err := prepEpoll(r.fd, unix.EPOLLIN|unix.EPOLLRDHUP)
	if err != nil {
		return 0, err
	}
	defer ep.Close()

	var n int
	for {
		if n, err = unix.Read(r.fd, p); err == nil || !IsEAGAIN(err) {
			break
		}

		ev, err := ep.latchWait(&r.threadLatch)
		if err != nil {
			if IsEAGAIN(err) {
				if cancelled || r.closed() {
					break
				}
			} else {
				break
			}
		}

		if ev&(unix.EPOLLHUP|unix.EPOLLRDHUP) > 0 {
			err = io.EOF
			break
		}

		if err = ep.rearm(); err != nil {
			break
		}
	}

	if IsEAGAIN(err) {
		switch {
		case r.closed():
			err = io.EOF

		case cancelled:
			// n.b. ctx cannot be nil if cancelled is true
			err = ctx.Err()
		}
	}

	return n, err
}
