package talk

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClientStreamWithToken(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	done := make(chan []byte, 1)
	go func() {
		var peer *net.UDPAddr
		var pcm []byte
		buf := make([]byte, 1024)
		for {
			n, src, err := server.ReadFromUDP(buf)
			if err != nil {
				return
			}
			msg := string(buf[:n])
			if msg == "ATOMTALK secret\n" {
				peer = src
				_, _ = server.WriteToUDP([]byte("OK\n"), src)
				continue
			}
			if msg == "ATOMTALK secret STOP\n" {
				_, _ = server.WriteToUDP([]byte("OK stop\n"), src)
				done <- pcm
				return
			}
			if peer != nil && src.String() == peer.String() {
				pcm = append(pcm, buf[:n]...)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	port := server.LocalAddr().(*net.UDPAddr).Port
	client := NewClient("127.0.0.1", port, "secret")
	payload := bytes.Repeat([]byte{0x12, 0x34}, defaultFrameBytes/2)
	if err := client.Stream(ctx, bytes.NewReader(payload)); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		if !bytes.Equal(got, payload) {
			t.Fatalf("payload mismatch: got %d bytes", len(got))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestClientStreamSendsStartControlWithoutToken(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	done := make(chan []byte, 1)
	go func() {
		var peer *net.UDPAddr
		var pcm []byte
		buf := make([]byte, 1024)
		for {
			n, src, err := server.ReadFromUDP(buf)
			if err != nil {
				return
			}
			msg := string(buf[:n])
			if msg == "ATOMTALK\n" {
				peer = src
				_, _ = server.WriteToUDP([]byte("OK\n"), src)
				continue
			}
			if msg == "ATOMTALK STOP\n" {
				_, _ = server.WriteToUDP([]byte("OK stop\n"), src)
				done <- pcm
				return
			}
			if peer != nil && src.String() == peer.String() {
				pcm = append(pcm, buf[:n]...)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	port := server.LocalAddr().(*net.UDPAddr).Port
	client := NewClient("127.0.0.1", port, "")
	payload := bytes.Repeat([]byte{0x56, 0x78}, defaultFrameBytes/2)
	if err := client.Stream(ctx, bytes.NewReader(payload)); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		if !bytes.Equal(got, payload) {
			t.Fatalf("payload mismatch: got %d bytes", len(got))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestClientStreamReturnsControlRejectionWithoutToken(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	go func() {
		buf := make([]byte, 1024)
		n, src, err := server.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if string(buf[:n]) == "ATOMTALK\n" {
			_, _ = server.WriteToUDP([]byte("ERR auth\n"), src)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	port := server.LocalAddr().(*net.UDPAddr).Port
	client := NewClient("127.0.0.1", port, "")
	err = client.Stream(ctx, bytes.NewReader([]byte{0x12, 0x34}))
	if err == nil {
		t.Fatal("expected control rejection")
	}
	if !strings.Contains(err.Error(), "ERR auth") {
		t.Fatalf("unexpected error: %v", err)
	}
}
