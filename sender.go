//
// obs-teleport. OBS Studio plugin.
// Copyright (C) 2021-2022 Florian Zwoch <fzwoch@gmail.com>
//
// This file is part of obs-teleport.
//
// obs-teleport is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 2 of the License, or
// (at your option) any later version.
//
// obs-teleport is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with obs-teleport. If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"net"
	"sync"
	"sync/atomic"
)

type Sender struct {
	sync.Mutex
	sync.WaitGroup
	outstandingWriteCount atomic.Int32
	conns map[net.Conn]any
}

func (s *Sender) SenderAdd(c net.Conn) {
	s.Lock()
	defer s.Unlock()

	if s.conns == nil {
		s.conns = make(map[net.Conn]any)
	}

	s.conns[c] = nil
}

func (s *Sender) SenderGetNumConns() int {
	s.Lock()
	defer s.Unlock()

	return len(s.conns)
}

func (s *Sender) SenderSend(b []byte) {
	s.Lock()
	defer s.Unlock()

	for c := range s.conns {
		s.Add(1)

		go func(c net.Conn) {
			defer s.Done()

			s.outstandingWriteCount.Add(1)
			_, err := c.Write(b)
			s.outstandingWriteCount.Add(-1)
			if err != nil {
				c.Close()
				s.Lock()
				delete(s.conns, c)
				s.Unlock()
			}
		}(c)
	}
}

func (s *Sender) SenderClose() {
	s.Lock()

	for c := range s.conns {
		c.Close()
	}

	s.conns = make(map[net.Conn]any)

	s.Unlock()
	s.Wait()
}
