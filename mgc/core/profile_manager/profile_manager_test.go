package profile_manager

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/spf13/afero"
	"magalu.cloud/core/utils"
)

type testFsEntry struct {
	path string
	mode fs.FileMode
	data []byte
}

type testCaseProfileManager struct {
	name          string
	expectedError error
	expectedFs    []testFsEntry
	providedFs    []testFsEntry
	run           func(m *ProfileManager) error
}

func findFsEntry(path string, entries []testFsEntry) (testFsEntry, error) {
	for _, e := range entries {
		if e.path == path {
			return e, nil
		}
	}
	return testFsEntry{}, fmt.Errorf("%q: %w", path, fs.ErrNotExist)
}

func getDirs(p string) (dirs []string) {
	for i, c := range p {
		if c == '/' && i > 0 {
			dirs = append(dirs, p[:i])
		}
	}
	return
}

func mergeFsEntries(toBeMerged ...[]testFsEntry) (merged []testFsEntry) {
	knownPaths := map[string]bool{}
	for _, entries := range toBeMerged {
		for _, e := range entries {
			if !knownPaths[e.path] {
				knownPaths[e.path] = true
				merged = append(merged, e)
			}
		}
	}
	return
}

func autoMkdirAll(entries []testFsEntry) (expanded []testFsEntry) {
	knownPaths := map[string]bool{}
	for _, e := range entries {
		knownPaths[e.path] = true
	}

	for _, e := range entries {
		for _, d := range getDirs(e.path) {
			if !knownPaths[d] {
				knownPaths[d] = true
				expanded = append(expanded, testFsEntry{
					path: d,
					mode: fs.ModeDir | utils.DIR_PERMISSION,
					data: nil,
				})
			}
		}
		expanded = append(expanded, e)
	}

	return expanded
}

func prepareFs(afs afero.Fs, provided []testFsEntry) (err error) {
	for _, p := range provided {
		if p.mode&fs.ModeDir != 0 {
			err = afs.Mkdir(p.path, p.mode)
		} else {
			err = afero.WriteFile(afs, p.path, p.data, p.mode)
		}
		if err != nil {
			return
		}
	}
	return
}

func checkFs(afs afero.Fs, expected []testFsEntry) (err error) {
	existingFiles := 0
	err = afero.Walk(afs, "/", func(path string, info fs.FileInfo, e error) (err error) {
		if e != nil {
			return e
		}
		if path == "/" {
			return nil
		}
		fsEntry, err := findFsEntry(path, expected)
		if err != nil {
			return
		}
		if fsEntry.mode != info.Mode() {
			return fmt.Errorf("%s: expected mode %x, got %x", path, fsEntry.mode, info.Mode())
		}
		if fsEntry.mode&fs.ModeDir == 0 {
			var data []byte
			if data, err = afero.ReadFile(afs, path); err != nil {
				return
			}
			if !bytes.Equal(fsEntry.data, data) {
				return fmt.Errorf("%s: expected data %q, got %q", path, fsEntry.data, data)
			}
		}
		existingFiles++
		return nil
	})
	if err != nil {
		return
	}

	if len(expected) != existingFiles {
		return fmt.Errorf("expected %d FS entries, got %d", len(expected), existingFiles)
	}

	return
}

func createProfileManagerGetTest(name string, expectedError error) testCaseProfileManager {
	return testCaseProfileManager{
		name:          fmt.Sprintf("ProfileManager.Get(%q)", name),
		expectedError: expectedError,
		run: func(m *ProfileManager) error {
			p, err := m.Get(name)
			if err != nil {
				return err
			}
			if name != p.Name {
				return fmt.Errorf("expected name %q, got %q", name, p.Name)
			}
			return nil
		},
	}
}

func createProfileManagerCurrentTest(testName string, profileName string, provided []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	return testCaseProfileManager{
		name:       fmt.Sprintf("ProfileManager.Current()==%q[%s]", profileName, testName),
		providedFs: provided,
		expectedFs: provided,
		run: func(m *ProfileManager) error {
			p := m.Current()
			if profileName != p.Name {
				return fmt.Errorf("expected name %q, got %q", profileName, p.Name)
			}
			return nil
		},
	}
}

func createProfileManagerSetCurrentTest(testName string, profileName string, provided []testFsEntry, expected []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	expected = autoMkdirAll(mergeFsEntries(expected, provided))
	return testCaseProfileManager{
		name:       fmt.Sprintf("ProfileManager.SetCurrent(%q)[%s]", profileName, testName),
		providedFs: provided,
		expectedFs: expected,
		run: func(m *ProfileManager) error {
			p, err := m.Get(profileName)
			if err != nil {
				return err
			}
			return m.SetCurrent(p)
		},
	}
}

func createProfileManagerCreateTest(testName string, profileName string, expectedError error, provided []testFsEntry, expected []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	expected = autoMkdirAll(mergeFsEntries(expected, provided))
	return testCaseProfileManager{
		name:          fmt.Sprintf("ProfileManager.Create(%q)[%s]", profileName, testName),
		providedFs:    provided,
		expectedFs:    expected,
		expectedError: expectedError,
		run: func(m *ProfileManager) error {
			p, err := m.Create(profileName)
			if err != nil {
				return err
			}
			if profileName != p.Name {
				return fmt.Errorf("expected name %q, got %q", profileName, p.Name)
			}
			return nil
		},
	}
}

func createProfileManagerDeleteTest(testName string, profileName string, expectedError error, provided []testFsEntry, expected []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	expected = autoMkdirAll(expected)
	return testCaseProfileManager{
		name:          fmt.Sprintf("ProfileManager.Delete(%q)[%s]", profileName, testName),
		providedFs:    provided,
		expectedFs:    expected,
		expectedError: expectedError,
		run: func(m *ProfileManager) error {
			p, err := m.Get(profileName)
			if err != nil {
				return err
			}
			return m.Delete(p)
		},
	}
}

func createProfileManagerListTest(testName string, profileNames []string, provided []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	slices.Sort(profileNames)
	return testCaseProfileManager{
		name:       fmt.Sprintf("ProfileManager.List()=%v[%s]", profileNames, testName),
		providedFs: provided,
		expectedFs: provided,
		run: func(m *ProfileManager) error {
			profiles := m.List()
			if len(profileNames) != len(profiles) {
				return fmt.Errorf("expected %d profiles, got %d", len(profileNames), len(profiles))
			}

			for i, p := range profiles {
				if profileNames[i] != p.Name {
					return fmt.Errorf("expected profile %d to be named %q, got %q", i, profileNames[i], p.Name)
				}
			}

			return nil
		},
	}
}

func createProfileManagerCopyTest(testName string, srcName string, dstName string, expectedError error, provided []testFsEntry, expected []testFsEntry) testCaseProfileManager {
	provided = autoMkdirAll(provided)
	expected = autoMkdirAll(mergeFsEntries(provided, expected))
	return testCaseProfileManager{
		name:          fmt.Sprintf("ProfileManager.Copy(%q,%q)[%s]", srcName, dstName, testName),
		providedFs:    provided,
		expectedFs:    expected,
		expectedError: expectedError,
		run: func(m *ProfileManager) error {
			src, err := m.Get(srcName)
			if err != nil {
				return err
			}
			dst, err := m.Get(dstName)
			if err != nil {
				return err
			}
			return m.Copy(src, dst)
		},
	}
}

func TestProfileManager(t *testing.T) {
	dir, err := buildMGCPath()
	if err != nil {
		t.Errorf("buildMGCPath: %s", err.Error())
	}

	tests := []testCaseProfileManager{
		// Get()
		createProfileManagerGetTest("default", nil),
		createProfileManagerGetTest("current", errorNameNotAllowed),
		createProfileManagerGetTest("*+&le", errorInvalidName),
		// Current()
		createProfileManagerCurrentTest("empty-fs", defaultProfileName, nil),
		createProfileManagerCurrentTest("empty-file", defaultProfileName, []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte(""),
			},
		}),
		createProfileManagerCurrentTest("provided", "a-profile-name", []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte("a-profile-name"),
			},
		}),
		// SetCurrent()
		createProfileManagerSetCurrentTest("empty-fs", defaultProfileName, nil, []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte(defaultProfileName),
			},
		}),
		createProfileManagerSetCurrentTest("provided", "other-name",
			[]testFsEntry{
				{
					path: path.Join(dir, currentProfileNameFile),
					mode: utils.FILE_PERMISSION,
					data: []byte("a-profile-name"),
				},
			}, []testFsEntry{
				{
					path: path.Join(dir, currentProfileNameFile),
					mode: utils.FILE_PERMISSION,
					data: []byte("other-name"),
				},
			},
		),
		// Create()
		createProfileManagerCreateTest("empty-fs", defaultProfileName, nil, nil, []testFsEntry{
			{
				path: path.Join(dir, defaultProfileName),
				mode: utils.DIR_PERMISSION | fs.ModeDir,
			},
		}),
		createProfileManagerCreateTest("other-profile", "a-profile-name", nil,
			[]testFsEntry{
				{
					path: path.Join(dir, "other-name"),
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			}, []testFsEntry{
				{
					path: path.Join(dir, "a-profile-name"),
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			},
		),
		createProfileManagerCreateTest("existing-profile", "existing-profile", errorProfileAlreadyExists,
			[]testFsEntry{
				{
					path: path.Join(dir, "existing-profile"),
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			}, nil,
		),
		createProfileManagerCreateTest("invalid-name", "*+&le", errorInvalidName, nil, nil),
		// Delete()
		createProfileManagerDeleteTest("empty-fs", defaultProfileName, errorDeleteCurrentNotAllowed, nil, nil),
		createProfileManagerDeleteTest("missing-profile", "a-profile-name", nil, nil, nil),
		createProfileManagerDeleteTest("existing-profile", "existing-name", nil,
			[]testFsEntry{
				{
					path: path.Join(dir, "existing-name"),
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			}, []testFsEntry{
				{
					path: dir,
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			},
		),
		createProfileManagerDeleteTest("existing-profile-with-files", "existing-name", nil,
			[]testFsEntry{
				{
					path: path.Join(dir, "existing-name/some-file"),
					mode: utils.FILE_PERMISSION,
					data: []byte("some contents"),
				},
			}, []testFsEntry{
				{
					path: dir,
					mode: utils.DIR_PERMISSION | fs.ModeDir,
				},
			},
		),
		// List()
		createProfileManagerListTest("empty-fs", []string{defaultProfileName}, nil),
		createProfileManagerListTest("missing-current", []string{"a-profile-name"}, []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte("a-profile-name"),
			},
		}),
		createProfileManagerListTest("existing-current-should-not-duplicate", []string{"a-profile-name"}, []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte("a-profile-name"),
			},
			{
				path: path.Join(dir, "a-profile-name"),
				mode: utils.DIR_PERMISSION | fs.ModeDir,
			},
		}),
		createProfileManagerListTest("multiple", []string{defaultProfileName, "a-profile-name", "other-name"}, []testFsEntry{
			{
				path: path.Join(dir, currentProfileNameFile),
				mode: utils.FILE_PERMISSION,
				data: []byte("a-profile-name"),
			},
			{
				path: path.Join(dir, defaultProfileName),
				mode: utils.DIR_PERMISSION | fs.ModeDir,
			},
			{
				path: path.Join(dir, "other-name"),
				mode: utils.DIR_PERMISSION | fs.ModeDir,
			},
		}),
		// Copy()
		createProfileManagerCopyTest("empty-fs", defaultProfileName, "a-profile-name", nil, nil, nil),
		createProfileManagerCopyTest("copy", defaultProfileName, "a-profile-name", nil,
			[]testFsEntry{
				{
					path: path.Join(dir, defaultProfileName, "some-file"),
					mode: utils.FILE_PERMISSION,
					data: []byte("contents-here"),
				},
				{
					path: path.Join(dir, defaultProfileName, "other-file"),
					mode: utils.FILE_PERMISSION,
					data: []byte("other-contents-here"),
				},
			},
			[]testFsEntry{
				{
					path: path.Join(dir, "a-profile-name", "some-file"),
					mode: utils.FILE_PERMISSION,
					data: []byte("contents-here"),
				},
				{
					path: path.Join(dir, "a-profile-name", "other-file"),
					mode: utils.FILE_PERMISSION,
					data: []byte("other-contents-here"),
				},
			},
		),
		createProfileManagerCopyTest("self-copy", defaultProfileName, defaultProfileName, errorCopyToSelf, nil, nil),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			m := &ProfileManager{dir, fs}
			err := prepareFs(fs, tc.providedFs)
			if err != nil {
				t.Errorf("could not prepare provided FS: %s", err.Error())
			}
			err = tc.run(m)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected err == %#v, found: %#v", tc.expectedError, err)
			}
			err = checkFs(fs, tc.expectedFs)
			if err != nil {
				t.Errorf("unexpected FS state: %s", err.Error())
			}
		})
	}
}
