// Copyright 2012-2018 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	if parseSize(nil) != -1 {
		t.Fatal("Should error on nil byte slice")
	}
	n := []byte("12345678")
	if pn := parseSize(n); pn != 12345678 {
		t.Fatalf("Did not parse %q correctly, res=%d\n", n, pn)
	}
}

func TestParseSInt64(t *testing.T) {
	if parseInt64(nil) != -1 {
		t.Fatal("Should error on nil byte slice")
	}
	n := []byte("12345678")
	if pn := parseInt64(n); pn != 12345678 {
		t.Fatalf("Did not parse %q correctly, res=%d\n", n, pn)
	}
}

func TestParseHostPort(t *testing.T) {
	check := func(hostPort string, defaultPort int, expectedHost string, expectedPort int, expectedErr bool) {
		h, p, err := parseHostPort(hostPort, defaultPort)
		if expectedErr {
			if err == nil {
				stackFatalf(t, "Expected an error, did not get one")
			}
			// expected error, so we are done
			return
		}
		if !expectedErr && err != nil {
			stackFatalf(t, "Unexpected error: %v", err)
		}
		if expectedHost != h {
			stackFatalf(t, "Expected host %q, got %q", expectedHost, h)
		}
		if expectedPort != p {
			stackFatalf(t, "Expected port %d, got %d", expectedPort, p)
		}
	}
	check("addr:1234", 5678, "addr", 1234, false)
	check(" addr:1234 ", 5678, "addr", 1234, false)
	check(" addr : 1234 ", 5678, "addr", 1234, false)
	check("addr", 5678, "addr", 5678, false)
	check(" addr ", 5678, "addr", 5678, false)
	check("addr:-1", 5678, "addr", 5678, false)
	check(" addr:-1 ", 5678, "addr", 5678, false)
	check(" addr : -1 ", 5678, "addr", 5678, false)
	check("addr:0", 5678, "addr", 5678, false)
	check(" addr:0 ", 5678, "addr", 5678, false)
	check(" addr : 0 ", 5678, "addr", 5678, false)
	check("addr:addr", 0, "", 0, true)
	check("addr:::1234", 0, "", 0, true)
	check("", 0, "", 0, true)
}

func TestURLsAreEqual(t *testing.T) {
	check := func(t *testing.T, u1Str, u2Str string, expectedSame bool) {
		t.Helper()
		u1, err := url.Parse(u1Str)
		if err != nil {
			t.Fatalf("Error parsing url %q: %v", u1Str, err)
		}
		u2, err := url.Parse(u2Str)
		if err != nil {
			t.Fatalf("Error parsing url %q: %v", u2Str, err)
		}
		same := urlsAreEqual(u1, u2)
		if expectedSame && !same {
			t.Fatalf("Expected %v and %v to be the same, they were not", u1, u2)
		} else if !expectedSame && same {
			t.Fatalf("Expected %v and %v to be different, they were not", u1, u2)
		}
	}
	check(t, "nats://localhost:4222", "nats://localhost:4222", true)
	check(t, "nats://ivan:pwd@localhost:4222", "nats://ivan:pwd@localhost:4222", true)
	check(t, "nats://ivan@localhost:4222", "nats://ivan@localhost:4222", true)
	check(t, "nats://ivan:@localhost:4222", "nats://ivan:@localhost:4222", true)
	check(t, "nats://host1:4222", "nats://host2:4222", false)
}

func BenchmarkParseInt(b *testing.B) {
	b.SetBytes(1)
	n := "12345678"
	for i := 0; i < b.N; i++ {
		strconv.ParseInt(n, 10, 0)
	}
}

func BenchmarkParseSize(b *testing.B) {
	b.SetBytes(1)
	n := []byte("12345678")
	for i := 0; i < b.N; i++ {
		parseSize(n)
	}
}

func deferUnlock(mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
	// see noDeferUnlock
	if false {
		return
	}
}

func BenchmarkDeferMutex(b *testing.B) {
	var mu sync.Mutex
	b.SetBytes(1)
	for i := 0; i < b.N; i++ {
		deferUnlock(&mu)
	}
}

func noDeferUnlock(mu *sync.Mutex) {
	mu.Lock()
	// prevent staticcheck warning about empty critical section
	if false {
		return
	}
	mu.Unlock()
}

func BenchmarkNoDeferMutex(b *testing.B) {
	var mu sync.Mutex
	b.SetBytes(1)
	for i := 0; i < b.N; i++ {
		noDeferUnlock(&mu)
	}
}

func createTestSub() *subscription {
	return &subscription{
		subject: []byte("foo"),
		queue:   []byte("bar"),
		sid:     []byte("22"),
	}
}

func BenchmarkArrayRand(b *testing.B) {
	b.StopTimer()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Create an array of 10 items
	subs := []*subscription{}
	for i := 0; i < 10; i++ {
		subs = append(subs, createTestSub())
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		index := r.Intn(len(subs))
		_ = subs[index]
	}
}

func BenchmarkMapRange(b *testing.B) {
	b.StopTimer()
	// Create an map of 10 items
	subs := map[int]*subscription{}
	for i := 0; i < 10; i++ {
		subs[i] = createTestSub()
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		for range subs {
			break
		}
	}
}
