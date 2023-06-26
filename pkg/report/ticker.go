// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package report

import "time"

type Ticker interface {
	Ch() <-chan time.Time
	Stop()
}

type timeTicker struct {
	*time.Ticker
}

func (t *timeTicker) Ch() <-chan time.Time {
	return t.C
}
