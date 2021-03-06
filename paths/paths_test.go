package paths

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ubuntu/tutorial-deployment/testtools"
)

func TestDetectPaths(t *testing.T) {
	testCases := []struct {
		websitePath  string
		exportPath   string
		metadataPath string
		apiPath      string
		imagePath    string
		cwd          string

		// the wanted paths are relative to cwd defined above
		wantWebsitePath  string
		wantExportPath   string
		wantMetaDataPath string
		wantAPIPath      string
		wantImagesPath   string
		wantErr          bool
	}{
		{"/defined/website", "/other/export", "/metadata", "/api", "/images/assets", "", "/defined/website", "/other/export", "/metadata", "/api", "/images/assets", false},
		{"/defined/website", "export/path", "alt/metadata", "alt/api", "alt/images/assets", "", "/defined/website", "export/path", "alt/metadata", "alt/api", "alt/images/assets", false},
		{"", "export/path", "alt/metadata", "alt/api", "alt/images/assets", "", "", "export/path", "alt/metadata", "alt/api", "alt/images/assets", false},                                             // The 3 parameters are sufficient to avoid needing website root detection
		{"", paths.Export, "alt/metadata", "alt/api", "alt/images/assets", "testdata/nosite", "", "", "", "", "alt/images/assets", true},                                                              // Error due to no site detected
		{"", paths.Export, "alt/metadata", "alt/api", "alt/images/assets", "testdata/partialwebsite", "", "", "", "", "alt/images/assets", true},                                                      // Error due to no site detected
		{"", paths.Export, "alt/metadata", "alt/api", "alt/images/assets", "testdata/website", ".", defaultRelativeExportPath, "alt/metadata", "alt/api", "alt/images/assets", false},                 // Defined path are always relative to cwd, not website
		{"", paths.Export, "alt/metadata", "alt/api", "alt/images/assets", "testdata/website/subdir", "..", "../" + defaultRelativeExportPath, "alt/metadata", "alt/api", "alt/images/assets", false}, // Subdir path detection
		{"", paths.Export, paths.MetaData, paths.API, paths.Images, "testdata/website", ".", defaultRelativeExportPath, defaultRelativeMetadataPath, defaultRelativeAPIPath, defaultRelativeImagesPath, false},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("(website: %s), (export: %s), (metadata: %s), (api: %s), , (images: %s) in [%s]",
			tc.websitePath, tc.exportPath, tc.metadataPath, tc.apiPath, tc.imagePath, tc.cwd), func(t *testing.T) {
			// Setup/Teardown
			defer testtools.Chdir(t, tc.cwd)()
			cachepath, teardown := MockPath()
			defer teardown()
			cachepath.Website = tc.websitePath
			cachepath.Export = tc.exportPath
			cachepath.MetaData = tc.metadataPath
			cachepath.API = tc.apiPath
			cachepath.Images = tc.imagePath
			if tc.wantWebsitePath != "" {
				tc.wantWebsitePath = testtools.AbsPath(t, tc.wantWebsitePath)
			}
			tc.wantExportPath = testtools.AbsPath(t, tc.wantExportPath)
			tc.wantMetaDataPath = testtools.AbsPath(t, tc.wantMetaDataPath)
			tc.wantAPIPath = testtools.AbsPath(t, tc.wantAPIPath)
			tc.wantImagesPath = testtools.AbsPath(t, tc.wantImagesPath)

			// Test
			p := New()
			err := p.DetectPaths()

			// Error checking
			if err != nil != tc.wantErr {
				t.Errorf("DetectPaths() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err != nil {
				// Error is fatal, we don't care about paths
				return
			}

			// Paths checks
			if p.Website != tc.wantWebsitePath {
				t.Errorf("Website: got %s; want %s", p.Website, tc.wantWebsitePath)
			}
			if p.Export != tc.wantExportPath {
				t.Errorf("Export: got %s; want %s", p.Export, tc.wantExportPath)
			}
			if p.MetaData != tc.wantMetaDataPath {
				t.Errorf("Metadata: got %s; want %s", p.MetaData, tc.wantMetaDataPath)
			}
			if p.API != tc.wantAPIPath {
				t.Errorf("API: got %s; want %s", p.API, tc.wantAPIPath)
			}
			if p.Images != tc.wantImagesPath {
				t.Errorf("Images: got %s; want %s", p.Images, tc.wantImagesPath)
			}
		})
	}
}

func TestImportTutorialPaths(t *testing.T) {
	website := "/ws/"
	testCases := []struct {
		paths     []string
		wantPaths []string
	}{
		{nil, []string{website + defaultTutorialPath}},
		{[]string{"/rep1", "/rep2/tut1.md", "/rep3/rep5"}, []string{"/rep1", "/rep2/tut1.md", "/rep3/rep5"}},
		{[]string{"rep1", "../rep2/tut1.md", "rep3/rep5"}, []string{"rep1", "../rep2/tut1.md", "rep3/rep5"}},
		{[]string{"/foo/rep1"}, []string{"/foo/rep1"}},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("path argument: %+v", tc.paths), func(t *testing.T) {
			// Setup/Teardown
			p := Path{
				Website: website,
			}
			for i, want := range tc.wantPaths {
				tc.wantPaths[i] = testtools.AbsPath(t, want)
			}

			// Test
			err := p.ImportTutorialPaths(tc.paths)
			if err != nil {
				t.Errorf("err: %s", err)
			}

			if !reflect.DeepEqual(p.TutorialInputs, tc.wantPaths) {
				t.Errorf("Import path: got %+v; want %+v", p.TutorialInputs, tc.wantPaths)
			}
		})
	}
}

func TestCreateTempPathHandling(t *testing.T) {
	p := Path{}

	// Create temp dir
	if err := p.CreateTempOutPath(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.API == "" || p.Export == "" || p.Images == "" {
		t.Errorf("one of API (%s) or Export (%s) or Images (%s) is empty", p.API, p.Export, p.Images)
		return
	}
	tmpdir := p.API[:len(p.API)-len(defaultRelativeAPIPath)]
	if !strings.HasPrefix(p.Export, tmpdir) {
		t.Errorf("API (%s) and Export (%s) don't have the same temporary prefix", p.API, p.Export)
	}
	if !strings.HasPrefix(p.Images, tmpdir) {
		t.Errorf("API (%s) and Images (%s) don't have the same temporary prefix", p.API, p.Images)
	}
	if _, err := os.Stat(tmpdir); os.IsNotExist(err) {
		t.Errorf("%s doesn't exists", tmpdir)
		return
	}

	// Remove temp dir
	if err := p.CleanTempPath(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := os.Stat(tmpdir); err == nil {
		t.Errorf("%s still exists", tmpdir)
	}
	if p.API != "" || p.Export != "" || p.Images != "" {
		t.Errorf("API (%s), Export (%s) and Images (%s) should now be empty", p.API, p.Export, p.Images)
	}
}

func TestTryCleanNonTempDir(t *testing.T) {
	p := Path{}

	if err := p.CleanTempPath(); err == nil {
		t.Errorf("Cleaning a non temporary path object should have returned an error: %+v", p)
	}
}
