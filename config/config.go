package config

import (
	"os"
	"strings"

	"github.com/hashicorp/go-memdb"
	"github.com/r3labs/diff"
	"github.com/spf13/afero"
	"github.com/stuartcarnie/gopm/model"
)

// Config memory representations of supervisor configuration file
type Config struct {
	db *memdb.MemDB

	Environment  *Environment
	ProgramGroup *ProcessGroup
}

// NewConfig create Config object
func NewConfig() *Config {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"process": {
				Name: "process",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"group": {
				Name: "group",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"server": {
				Name: "server",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"file": {
				Name: "file",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"local_file": {
				Name: "local_file",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	return &Config{
		db:           db,
		ProgramGroup: NewProcessGroup(),
	}
}

func (c *Config) LoadString(s string) (memdb.Changes, error) {
	var (
		m   *model.Root
		err error
	)

	var r model.Reader
	m, err = r.LoadReader(strings.NewReader(s))
	if err != nil {
		return nil, err
	}

	return c.update(m)
}

// Load loads the configuration and return the loaded programs
func (c *Config) LoadPath(configFile string) (memdb.Changes, error) {
	var (
		m   *model.Root
		err error
	)

	var r model.Reader
	m, err = r.LoadPath(configFile)
	if err != nil {
		return nil, err
	}

	return c.update(m)
}

func (c *Config) update(m *model.Root) (memdb.Changes, error) {
	if err := model.Validate(m); err != nil {
		return nil, err
	}

	// TODO(sgc): This should be expanded over the in-memory data
	if err := ExpandEnv(m); err != nil {
		return nil, err
	}

	txn := c.db.Txn(true)
	txn.TrackChanges()
	err := ApplyUpdates(txn, m)
	if err != nil {
		txn.Abort()
		return nil, err
	}

	ri, err := txn.Get("file", "id")
	if err != nil {
		panic(err)
	}

	var files []*File
	for {
		f, ok := ri.Next().(*File)
		if !ok {
			break
		}
		files = append(files, f)
	}

	// update local files table
	if len(files) > 0 {
		fs := NewFileSystemWriter(afero.NewOsFs())
		root, localFiles, err := fs.Commit(files[0].Root, files)
		if err != nil {
			txn.Abort()
			return nil, err
		}

		_ = os.Setenv("GOPM_FS_ROOT", root)

		for _, lf := range localFiles {
			raw, _ := txn.First("local_file", "id", lf.Name)
			if orig, ok := raw.(*LocalFile); ok && !diff.Changed(orig, lf) {
				continue
			}
			_ = txn.Insert("local_file", lf)
			// TODO(sgc): Want to merge these with each process environment
			key := strings.ToUpper("GOPM_FS_" + lf.Name)
			_ = os.Setenv(key, lf.FullPath)
		}
	}

	ch := txn.Changes()
	txn.Commit()

	return ch, nil
}

func (c *Config) Processes() Processes {
	res := make(Processes, 0)
	txn := c.db.Txn(false)
	defer txn.Commit()

	iter, _ := txn.Get("process", "id")
	for {
		p, ok := iter.Next().(*Process)
		if !ok {
			return res.Sorted()
		}
		res = append(res, p)
	}
}

func (c *Config) ProcessNames() []string {
	return c.Processes().Names()
}

func (c *Config) GetProcess(name string) *Process {
	txn := c.db.Txn(false)
	defer txn.Commit()
	raw, err := txn.First("process", "id", name)
	if raw == nil || err != nil {
		return nil
	}
	return raw.(*Process)
}

func (c *Config) GetGrpcServer() *Server {
	txn := c.db.Txn(false)
	defer txn.Commit()
	raw, err := txn.First("server", "id", "grpc")
	if raw == nil || err != nil {
		return nil
	}
	return raw.(*Server)
}

func (c *Config) GetHttpServer() *Server {
	txn := c.db.Txn(false)
	defer txn.Commit()
	raw, err := txn.First("server", "id", "http")
	if raw == nil || err != nil {
		return nil
	}
	return raw.(*Server)
}
