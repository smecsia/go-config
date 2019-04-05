package yamledit_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/smecsia/go-utils/pkg/util/yamledit"
)

func TestYamlEdit(t *testing.T) {
	RegisterTestingT(t)

	tmpDir, err := ioutil.TempDir("", "tmpdir")
	sdPath := path.Join(tmpDir, "sd.yaml")

	bytes, err := ioutil.ReadFile("testdata/sd.yaml")
	Expect(err).To(BeNil())

	err = ioutil.WriteFile(sdPath, bytes, os.ModePerm)
	Expect(err).To(BeNil())

	yedit := YamlEdit{WriteInPlace: true, SkipNotExisting: true}
	err = yedit.ModifyProperty(sdPath, "compose.armory.image", "${ARMORY_IMAGE}")
	err = yedit.ModifyProperty(sdPath, "compose.armory.digest", "${ARMORY_DIGEST}")
	err = yedit.ModifyProperty(sdPath, "compose.armory.tag", "${ARMORY_TAG}")
	err = yedit.ModifyProperty(sdPath, "links.binary.tag", "${ARMORY_TAG}")
	err = yedit.ModifyProperty(sdPath, "links.binary.name", "${ARMORY_IMAGE}")
	err = yedit.ModifyProperty(sdPath, "links.armory", "${ARMORY_IMAGE}")
	Expect(err).To(BeNil())

	bytes, err = ioutil.ReadFile(sdPath)
	Expect(err).To(BeNil())

	Expect(string(bytes)).To(ContainSubstring("image: ${ARMORY_IMAGE}"))
	Expect(string(bytes)).To(ContainSubstring("digest: ${ARMORY_DIGEST}"))
	Expect(string(bytes)).To(ContainSubstring("tag: ${ARMORY_TAG}"))
	Expect(string(bytes)).NotTo(ContainSubstring("tag: null"))
	Expect(string(bytes)).NotTo(ContainSubstring("armory: ${ARMORY_IMAGE}"))
	Expect(string(bytes)).To(ContainSubstring("name: ${ARMORY_IMAGE}"))
}
