package hilbert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
)

func TestConformanceFixtures(t *testing.T) {
	root := filepath.Join("..", "testdata", "conformance")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", root, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()
		path := filepath.Join(root, name)
		t.Run(name, func(t *testing.T) {
			cert, err := certificates.LoadJSONFile(path)
			if strings.HasPrefix(name, "valid_") && err != nil {
				t.Fatalf("LoadJSONFile(%q) error = %v", path, err)
			}
			if strings.HasPrefix(name, "invalid_") && err != nil {
				return
			}

			_, err = VerifyCertificate(cert)
			switch {
			case strings.HasPrefix(name, "valid_") && err != nil:
				t.Fatalf("VerifyCertificate(%q) error = %v, want success", path, err)
			case strings.HasPrefix(name, "invalid_") && err == nil:
				t.Fatalf("VerifyCertificate(%q) error = nil, want failure", path)
			}
		})
	}
}
