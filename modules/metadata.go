package modules

type Metadata struct {
	Name string
	Hash string
}

func NewMetadata(name, hash string) Metadata {
	return Metadata{
		Name: name,
		Hash: hash,
	}
}

func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}
