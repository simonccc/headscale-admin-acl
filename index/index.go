package index

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/google/uuid"
)

type IndexFileInfo struct {
	Path string `json:"path"`
}

// Main index to keep track of files by a profile name
type Index struct {
	idxDir      string // index JSON filename
	aclFilename string // target ACL filename to update
	mu          sync.Mutex
	files       map[string]IndexFileInfo // key is the profile name, value is the information of the file
}

func (idx *Index) profileDirPath() string {
	// uses "profiles" folder to store individual profiles.
	return path.Join(idx.idxDir, "profiles")
}

func (idx *Index) profileIndexPath() string {
	// uses "index.json" file to store profile information
	return path.Join(idx.profileDirPath(), "index.json")
}

// create or ensure the existence of the Index directory
func (idx *Index) createIdxDir() error {
	var stat fs.FileInfo
	var err error

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// check if the idxDir base directory exists
	stat, err = os.Stat(idx.idxDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		// create the directory if it does not exist
		if err = os.MkdirAll(idx.idxDir, 0755); err != nil {
			return err
		}

		// get the updated state of the newly created directory
		stat, err = os.Stat(idx.idxDir)
		if err != nil {
			return err
		}
	}

	// stat should be populated at this point. ensure it is a directory
	if !stat.IsDir() {
		return fmt.Errorf("idxDir '%s' is not a valid directory", idx.idxDir)
	}

	// and ensure the directory is writable
	if stat.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("idxDir '%s' is not writable", idx.idxDir)
	}

	// create the profile directory if it does not exist
	if err = os.MkdirAll(idx.profileDirPath(), 0755); err != nil {
		return err
	}

	return nil
}

func (idx *Index) initializeIdx() error {
	var err error

	// ensure directory is created and valid
	if err = idx.createIdxDir(); err != nil {
		return err
	}

	// check if the index file exists
	_, err = os.Stat(idx.profileIndexPath())
	if err != nil {
		// if the index JSON file does not exist, create it with empty data
		if !os.IsNotExist(err) {
			return err
		}

		if err = idx.setIdxData(); err != nil {
			return err
		}
	} else {
		// index file exists. load the contents
		if data, err := os.ReadFile(idx.profileIndexPath()); err != nil {
			return err
		} else {
			return json.Unmarshal(data, &idx.files)
		}
	}

	return nil
}

// sets the data of the ACL file
func (idx *Index) setAclData(data []byte) error {
	var err error

	f, err := os.Create(idx.aclFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// Save the index
func (idx *Index) setIdxData() error {
	var err error

	f, err := os.Create(idx.profileIndexPath())
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.MarshalIndent(idx.files, "", "    ")
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}
func (idx *Index) Remove(profileName string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.files, profileName)
	return idx.setIdxData()
}

// assign data to a new or existing profile
func (idx *Index) Set(profileName string, profileData []byte) error {
	var err error
	var profilePath string
	var isNew bool = false

	idx.mu.Lock()
	defer idx.mu.Unlock()

	if info, ok := idx.files[profileName]; ok {
		// take the existing path
		profilePath = info.Path
	} else {
		// generate a new UUIDv4
		id, err := uuid.NewRandom()
		if err != nil {
			return err
		}

		profilePath = path.Join(idx.profileDirPath(), fmt.Sprintf("%s.hujson", id.String()))
		isNew = true
	}

	f, err := os.Create(profilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(profileData)
	if err != nil {
		os.Remove(profilePath)
		return err
	}

	if isNew {
		idx.files[profileName] = IndexFileInfo{
			Path: profilePath,
		}
	}

	return idx.setIdxData()
}

func (idx *Index) Apply(profileName string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if info, ok := idx.files[profileName]; ok {
		f, err := os.Open(info.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		return idx.setAclData(data)
	}

	return ErrProfileNotFound
}

func (idx *Index) RenameProfile(profileNameOld, profileNameNew string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// check if the old profile name exists
	if _, ok := idx.files[profileNameOld]; !ok {
		return ErrProfileNotFound
	}

	// check if the new profile name exists
	if _, ok := idx.files[profileNameNew]; ok {
		return ErrProfileExists
	}

	idx.files[profileNameNew] = idx.files[profileNameOld]
	delete(idx.files, profileNameOld)
	return idx.setIdxData()
}

// Create a new index container for keeping track of files
func CreateNewIndex(idxDir, aclFilename string) (*Index, error) {
	var err error

	idx := &Index{
		idxDir:      idxDir,
		aclFilename: aclFilename,
		mu:          sync.Mutex{},
		files:       make(map[string]IndexFileInfo),
	}

	// create the directory if necessary
	if err = idx.initializeIdx(); err != nil {
		return nil, err
	}

	return idx, nil
}
