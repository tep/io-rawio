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
	"sort"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"toolman.org/base/log/v2"
)

type epoll struct {
	pollFD int
	commFD int32
	events uint32
}

func prepEpoll(fd int, events uint32) (*epoll, error) {
	pfd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	p := &epoll{
		pollFD: pfd,
		commFD: int32(fd),
		events: events,
	}

	if err := unix.EpollCtl(pfd, unix.EPOLL_CTL_ADD, fd, p.event()); err != nil {
		unix.Close(pfd)
		return nil, err
	}

	return p, nil
}

func (p *epoll) rearm() error {
	return unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_MOD, int(p.commFD), p.event())
}

func (p *epoll) Close() error {
	return unix.Close(p.pollFD)
}

func logEpollEvents(evts []unix.EpollEvent) {
	log.Info("EpollEvents:")
	for i, e := range evts {
		log.Infof("    %d) &{Fd: %d, Events: %v}", i, e.Fd, events(e.Events))
	}
}

func (p *epoll) latchWait(l *threadLatch) (uint32, error) {
	l.latch()
	defer l.unlatch()
	return p.wait()
}

func (p *epoll) wait() (uint32, error) {
	if log.V(1) {
		log.Infof("Calling epollWait for fd=%d events=%v", p.commFD, events(p.events))
	}
	evts := make([]unix.EpollEvent, 1)
	n, err := unix.EpollWait(p.pollFD, evts, -1)
	if err != nil {
		return 0, err
	}

	if log.V(1) {
		logEpollEvents(evts)
	}

	if n != 1 {
		return 0, unix.EBADFD
	}

	if evts[0].Fd != p.commFD {
		return 0, unix.EBADE
	}

	return evts[0].Events, nil
}

func (p *epoll) event() *unix.EpollEvent {
	return &unix.EpollEvent{
		Fd:     p.commFD,
		Events: unix.EPOLLET | unix.EPOLLONESHOT | p.events,
	}
}

// IsEAGAIN returns true if err represents a system error indicating that an
// operation should be tried again.
func IsEAGAIN(err error) bool {
	if err == nil {
		return false
	}

	if errno, ok := err.(syscall.Errno); ok && (errno == syscall.EAGAIN || errno == syscall.EINTR) {
		return true
	}

	return false
}

var eventNames = map[uint32]string{
	unix.EPOLLIN:      "EPOLLIN",
	unix.EPOLLPRI:     "EPOLLPRI",
	unix.EPOLLOUT:     "EPOLLOUT",
	unix.EPOLLERR:     "EPOLLERR",
	unix.EPOLLHUP:     "EPOLLHUP",
	unix.EPOLLRDHUP:   "EPOLLRDHUP",
	unix.EPOLLET:      "EPOLLLET",
	unix.EPOLLONESHOT: "EPOLLONESHOT",
	unix.EPOLLWAKEUP:  "EPOLLWAKEUP",
}

var eventNameList = func() []string {
	list := make([]string, len(eventNames))
	var i int
	for _, n := range eventNames {
		list[i] = n
		i++
	}
	sort.Strings(list)
	return list
}()

type events uint32

func (e events) has(v uint32) bool {
	return uint32(e)&v != 0
}

func (e events) String() string {
	var list []string
	for evt, name := range eventNames {
		if evt&uint32(e) > 0 {
			list = append(list, name)
		}
	}
	return strings.Join(list, "|")
}

func eventString(e uint32) string {
	return events(e).String()
}
