package profile_manager

import (
	"path"
)

type profile struct {
	name string
	m    *ProfileManager
}

func newProfile(name string, m *ProfileManager) *profile {
	return &profile{name, m}
}

func (p *profile) Name() string {
	return p.name
}

func (p *profile) Dir() string {
	return path.Join(p.m.dir, p.name)
}

func (p *profile) buildPath(name string) string {
	s := sanitizePath(name)
	return path.Join(p.name, s)
}

func (p *profile) Write(name string, data []byte) error {
	return p.m.write(p.buildPath(name), data)
}

func (p *profile) Read(name string) ([]byte, error) {
	return p.m.read(p.buildPath(name))
}

func (p *profile) Delete(name string) error {
	return p.m.remove(p.buildPath(name))
}
