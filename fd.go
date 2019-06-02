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

package rawio

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
)

type filer interface {
	File() (*os.File, error)
}

// ErrNoFD indicates that no file descriptor could be extracted.
var ErrNoFD = errors.New("no manner to discern file descriptor from object")

// ExtractFD attempts to extract a usable file descriptor from x -- which must
// be either an *os.File or an object with a method having a signature of
// "File() (*os.File, error)", such as most implementations of net.Conn. If x
// is something else, ErrNoFD is returned.
//
// If x is an *os.File, its Fd method is called -- otherwise, an *os.File is
// aquired from the aforementioned File method and its Fd method is called.
// In either case, a file descriptor is acquired then duped and the duplicate
// file descriptor is returned.
//
// Prior to returning, Close is called on the *os.File used to acquire the
// original file descriptor and, if x is not an *os.File but is an io.Closer,
// x will also be closed. If any of these close operations result in an error,
// the duped file descriptor is also closed and an error is returned.
//
// On success, a usable file descriptor and a nil error are returned and the
// object from which it was extracted will be closed and no longer usable.
//
// Note: We dupe the file descriptor before returning it because os.File has
// a finalizer routine which calls Close when the object is garbage collected.
// This also closes the file descriptor returned by its Fd method.
//
// Deprecated: With the advent of the syscall.RawConn interface introduced in
// Go v1.12, ExtractFD is no longer necessary.
func ExtractFD(x interface{}) (int, error) {
	var fp *os.File
	var closeX bool

	switch v := x.(type) {
	case *os.File:
		fp = v

	case filer:
		closeX = true
		var err error
		if fp, err = v.File(); err != nil {
			return 0, err
		}
	}

	if fp == nil {
		return 0, ErrNoFD
	}

	var haveFD bool
	fd, fderr := syscall.Dup(int(fp.Fd()))
	if fderr == nil {
		haveFD = true
	}

	fperr := fp.Close()

	var xerr error
	if closeX {
		if c, ok := x.(io.Closer); ok {
			xerr = c.Close()
		}
	}

	if fderr == nil && fperr == nil && xerr == nil {
		return fd, nil
	}

	var derr error
	if haveFD {
		derr = syscall.Close(fd)
	}

	var errs []string
	for _, err := range []error{fderr, fperr, xerr, derr} {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	return 0, errors.New(strings.Join(errs, " -and- "))
}
