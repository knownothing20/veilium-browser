package systemproxy

import "testing"

func TestNormalizeStaticProxySingleEndpoint(t *testing.T) {
	got, err := normalizeStaticProxy("127.0.0.1:10100")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:10100" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeStaticProxyEquivalentSchemeMapping(t *testing.T) {
	got, err := normalizeStaticProxy("http=127.0.0.1:8080; https=127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:8080" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeStaticProxySOCKS(t *testing.T) {
	got, err := normalizeStaticProxy("socks=127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	if got != "socks5://127.0.0.1:1080" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeStaticProxyFailsClosed(t *testing.T) {
	tests := []string{
		"http=127.0.0.1:8080",
		"http=127.0.0.1:8080;https=127.0.0.1:8443",
		"http://user:secret@127.0.0.1:8080",
		"http://127.0.0.1:8080/path",
		"ftp=127.0.0.1:2121",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := normalizeStaticProxy(input); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
