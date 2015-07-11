package auth

import "testing"

func TestDecodeAuth(t *testing.T) {
	authString := "VGhpc0lzQVRlc3Q="
	expected := "ThisIsATest"
	result, err := decodeAuth(authString)
	if err != nil {
		t.Error(err)
	}

	if result != expected {
		t.Errorf("Wanted %s, got %s instead", expected, result)
	}
}
