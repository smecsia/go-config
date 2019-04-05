package util_test

import (
	. "github.com/onsi/gomega"
	. "github.com/smecsia/go-utils/pkg/util"
	"testing"
)

func TestReturnNilForNil(t *testing.T) {
	RegisterTestingT(t)

	v, err := GetValue("a.b.c.d", nil)
	Expect(err).To(BeNil())
	Expect(v).To(BeNil())
}

func TestKeyNotExist(t *testing.T) {
	RegisterTestingT(t)

	_, err := GetValue("a.b.c.d", map[string]interface{}{})
	Expect(err).NotTo(BeNil())
	Expect(err.Error()).To(MatchRegexp("key not present. \\[key:a\\]"))
}

func TestGetDeepValues(t *testing.T) {
	RegisterTestingT(t)

	obj := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"str": map[string]string{
					"val": "val",
				},
				"c": map[string]interface{}{
					"d": 10,
					"e": "string",
					"f": map[string]string{
						"a": "foobar",
					},
				},
			},
		},
	}
	v, err := GetValue("a.b.c.d", obj)
	Expect(err).To(BeNil())
	Expect(v).To(Equal(10))

	v, err = GetValue("a.b.str.val", obj)
	Expect(err).To(BeNil())
	Expect(v).To(Equal("val"))

	v, err = GetValue("a.b.c.e", obj)
	Expect(err).To(BeNil())
	Expect(v).To(Equal("string"))

	v, err = GetValue("a.b.c.f", obj)
	Expect(err).To(BeNil())
	Expect(v).To(Equal(map[string]string{
		"a": "foobar",
	}))
}
