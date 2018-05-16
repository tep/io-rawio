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
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type threadLatch struct {
	active   bool
	threadID int
	short    sync.Mutex
	long     sync.RWMutex
}

func (tl *threadLatch) latch() {
	tl.short.Lock()
	defer tl.short.Unlock()

	tl.activate()
	tl.long.RLock()
}

func (tl *threadLatch) activate() {
	tl.long.Lock()
	defer tl.long.Unlock()

	runtime.LockOSThread()
	tl.threadID = unix.Gettid()
	tl.active = true
}

func (tl *threadLatch) unlatch() {
	tl.short.Lock()
	defer tl.short.Unlock()

	tl.long.RUnlock()
	tl.deactivate()
}

func (tl *threadLatch) deactivate() {
	tl.long.Lock()
	defer tl.long.Unlock()

	runtime.UnlockOSThread()
	tl.threadID = 0
	tl.active = false
}

// NotifySignal is the signal that will be used to interrupt a blocked Read operation.
var NotifySignal = syscall.SIGIO

func (tl *threadLatch) notify(signal syscall.Signal) error {
	tl.long.RLock()
	defer tl.long.RUnlock()

	if !tl.active {
		return nil
	}

	if tl.threadID == unix.Gettid() {
		return nil
	}

	if signal == 0 {
		signal = NotifySignal
	}

	return unix.Tgkill(unix.Getpid(), tl.threadID, signal)
}
