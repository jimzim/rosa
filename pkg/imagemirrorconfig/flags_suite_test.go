package imagemirrorconfig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImageMirrorConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image Mirror Config Suite")
}