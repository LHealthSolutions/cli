package v2action_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2action/v2actionfakes"
	"code.cloudfoundry.org/ykk"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource Actions", func() {
	var (
		actor                     *Actor
		fakeCloudControllerClient *v2actionfakes.FakeCloudControllerClient
		srcDir                    string
	)

	BeforeEach(func() {
		fakeCloudControllerClient = new(v2actionfakes.FakeCloudControllerClient)
		actor = NewActor(fakeCloudControllerClient, nil)

		var err error
		srcDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		subDir := filepath.Join(srcDir, "level1", "level2")
		err = os.MkdirAll(subDir, 0777)
		Expect(err).ToNot(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(subDir, "tmpFile1"), []byte("why hello"), 0666)
		Expect(err).ToNot(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(srcDir, "tmpFile2"), []byte("Hello, Binky"), 0666)
		Expect(err).ToNot(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(srcDir, "tmpFile3"), []byte("Bananarama"), 0666)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GatherArchiveResources", func() {
		Context("when the archive exists", func() {
			var archive string

			BeforeEach(func() {
				tmpfile, err := ioutil.TempFile("", "gather-archive-resource-test")
				Expect(err).ToNot(HaveOccurred())
				defer tmpfile.Close()
				archive = tmpfile.Name()

				err = zipit(srcDir, archive, "")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(archive)).ToNot(HaveOccurred())
			})

			It("gathers a list of all files in a source archive", func() {
				resources, err := actor.GatherArchiveResources(archive)
				Expect(err).ToNot(HaveOccurred())

				Expect(resources).To(Equal(
					[]Resource{
						{Filename: "/"},
						{Filename: "/level1/"},
						{Filename: "/level1/level2/"},
						{Filename: "/level1/level2/tmpFile1", SHA1: "9e36efec86d571de3a38389ea799a796fe4782f4", Size: 9, Mode: 0666},
						{Filename: "/tmpFile2", SHA1: "e594bdc795bb293a0e55724137e53a36dc0d9e95", Size: 12, Mode: 0666},
						{Filename: "/tmpFile3", SHA1: "f4c9ca85f3e084ffad3abbdabbd2a890c034c879", Size: 10, Mode: 0666},
					}))
			})
		})

		Context("when the archive does not exist", func() {
			It("returns an error if the file is problematic", func() {
				_, err := actor.GatherArchiveResources("/does/not/exist")
				Expect(os.IsNotExist(err)).To(BeTrue())
			})
		})
	})

	Describe("GatherDirectoryResources", func() {
		It("gathers a list of all directories files in a source directory", func() {
			resources, err := actor.GatherDirectoryResources(srcDir)
			Expect(err).ToNot(HaveOccurred())

			Expect(resources).To(Equal(
				[]Resource{
					{Filename: "level1"},
					{Filename: "level1/level2"},
					{Filename: "level1/level2/tmpFile1", SHA1: "9e36efec86d571de3a38389ea799a796fe4782f4", Size: 9, Mode: 0766},
					{Filename: "tmpFile2", SHA1: "e594bdc795bb293a0e55724137e53a36dc0d9e95", Size: 12, Mode: 0766},
					{Filename: "tmpFile3", SHA1: "f4c9ca85f3e084ffad3abbdabbd2a890c034c879", Size: 10, Mode: 0766},
				}))
		})
	})

	Describe("ZipResources", func() {
		var (
			resultZip  string
			resources  []Resource
			executeErr error
		)

		BeforeEach(func() {
			resources = []Resource{
				{Filename: "level1"},
				{Filename: "level1/level2"},
				{Filename: "level1/level2/tmpFile1", SHA1: "9e36efec86d571de3a38389ea799a796fe4782f4", Size: 9, Mode: 0766},
				{Filename: "tmpFile2", SHA1: "e594bdc795bb293a0e55724137e53a36dc0d9e95", Size: 12, Mode: 0766},
				{Filename: "tmpFile3", SHA1: "f4c9ca85f3e084ffad3abbdabbd2a890c034c879", Size: 10, Mode: 0766},
			}
		})

		JustBeforeEach(func() {
			resultZip, executeErr = actor.ZipResources(srcDir, resources)
		})

		AfterEach(func() {
			err := os.RemoveAll(srcDir)
			Expect(err).ToNot(HaveOccurred())

			err = os.RemoveAll(resultZip)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when zipping on windows", func() {
			It("zips the directory and sets all the file modes to 07XX", func() {
				Expect(executeErr).ToNot(HaveOccurred())

				Expect(resultZip).ToNot(BeEmpty())
				zipFile, err := os.Open(resultZip)
				Expect(err).ToNot(HaveOccurred())
				defer zipFile.Close()

				zipInfo, err := zipFile.Stat()
				Expect(err).ToNot(HaveOccurred())

				reader, err := ykk.NewReader(zipFile, zipInfo.Size())
				Expect(err).ToNot(HaveOccurred())

				Expect(reader.File).To(HaveLen(5))
				Expect(reader.File[2].Mode()).To(Equal(os.FileMode(0766)))
				Expect(reader.File[3].Mode()).To(Equal(os.FileMode(0766)))
				Expect(reader.File[4].Mode()).To(Equal(os.FileMode(0766)))
			})
		})
	})
})
