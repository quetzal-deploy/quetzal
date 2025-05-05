package filter

import (
	"testing"

	"github.com/DBCDK/morph/nix"
)

func getHosts() (hosts []nix.Host) {
	hosts = append(hosts, nix.Host{Name: "abc"})
	hosts = append(hosts, nix.Host{Name: "def"})
	hosts = append(hosts, nix.Host{Name: "ghi"})

	return
}

func TestFoo(t *testing.T) {
	allHosts := getHosts()

	hosts1, err := MatchHosts(allHosts, "*")

	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}

	if len(hosts1) != len(allHosts) {
		t.Fatalf("Expected %d hosts, got %d: %v", len(allHosts), len(hosts1), hosts1)
	}
}
