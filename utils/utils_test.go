package utils

import (
	"os"
	"testing"
)

func Test_WithWorkingDirectory(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	restore := WithWorkingDirectory("..")

	after, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if after == current {
		t.Errorf("Working directory didn't change\n Before: %s\n After: %s\n", current, after)
	}

	restore()

	after, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if after != current {
		t.Errorf("Working directory should be the same after restore\n Before: %s\n After: %s\n", current, after)
	}
}
