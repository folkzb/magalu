package profile_manager

import (
	"path"
)

type Profile struct {
	Name string
	m    *ProfileManager
}

func newProfile(name string, m *ProfileManager) *Profile {
	return &Profile{name, m}
}

func (p *Profile) Dir() string {
	return path.Join(p.m.dir, p.Name)
}

func (p *Profile) buildPath(name string) string {
	s := sanitizePath(name)
	return path.Join(p.Name, s)
}

func (p *Profile) Write(name string, data []byte) error {
	return p.m.write(p.buildPath(name), data)
}

func (p *Profile) Read(name string) ([]byte, error) {
	return p.m.read(p.buildPath(name))
}

func (p *Profile) Delete(name string) error {
	return p.m.remove(p.buildPath(name))
}
