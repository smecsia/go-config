package util_test

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/smecsia/go-utils/pkg/util"
)

func TestFormatDuration(t *testing.T) {
	RegisterTestingT(t)

	Expect(FormatDurationSec(1009010 * 1000000)).To(Equal("16m49s"))
	Expect(FormatDurationSec(3000 * 1000000)).To(Equal("3s"))
	Expect(FormatDurationSec(100 * 1000000)).To(Equal("0s"))
	Expect(FormatDurationSec(1499 * 1000000)).To(Equal("1s"))
	Expect(FormatDurationSec(1501 * 1000000)).To(Equal("2s"))
	Expect(FormatDurationSec(60000 * 1000000)).To(Equal("1m"))
	Expect(FormatDurationSec(3600000 * 1000000)).To(Equal("1h"))
}
