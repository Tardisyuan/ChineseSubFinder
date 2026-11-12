package customport

import (
	"os"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/sirupsen/logrus"
)

func TestCustomPortWithNewline(t *testing.T) {
	old, _ := os.Getwd()
	_ = os.Chdir(t.TempDir())
	defer os.Chdir(old)
	if err := os.WriteFile("CustomPort", []byte("19102\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if p := pkg.ReadCustomPortFile(logrus.New()); p != 19102 {
		t.Fatalf("got %d", p)
	}
}
