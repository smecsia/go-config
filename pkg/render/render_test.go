package render

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Object struct {
	Name string
}

func TestExportedValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "json", FormatJSON)
	assert.Equal(t, "yaml", FormatYAML)
	assert.Equal(t, []string{"json", "yaml"}, Formats)
}

func TestWrite(t *testing.T) {
	RegisterTestingT(t)
	t.Parallel()

	d := Object{
		Name: "foo",
	}

	type testCase struct {
		format  string
		want    string
		wantErr error
	}
	cases := []testCase{
		{
			format:  "template://test/something/info.tpl",
			want:    "SomeTemplate: foo",
			wantErr: nil,
		},
	}
	for i, c := range cases {
		ti := i
		tc := c
		t.Run(fmt.Sprintf("[%d]", ti), func(t *testing.T) {
			t.Parallel()

			var b bytes.Buffer
			err := Write(&b, tc.format, d)
			if tc.wantErr != nil {
				assert.EqualError(t, tc.wantErr, err.Error())
				return
			}
			require.NoError(t, err)

			got := b.String()
			Expect(got).To(ContainSubstring(tc.want))
		})
	}
}
