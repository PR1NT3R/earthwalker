package badgerdb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger"
)

// TODO: I find instantiating this index system to be very awkward. Better way?

const indexPrefix = "index-"

// index tracks the IDs of all members of the group with ID GroupID
// Our badgerdb Store implementations use these to retrieve group members
// without scanning the entire db.
type index struct {
	GroupID   string
	ObjectIDs map[string]bool
}

type IndexStore struct {
	DB *badger.DB
}

func (store IndexStore) insert(ind index) error {
	err := storeStruct(store.DB, indexPrefix+ind.GroupID, ind)
	if err != nil {
		return fmt.Errorf("failed to write index to badger DB: %v", err)
	}
	return nil
}

func (store IndexStore) get(groupID string) (index, error) {
	var foundInd index
	indBytes, err := getBytes(store.DB, indexPrefix+groupID)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			// assume index hasn't been created for groupID and return a new one
			return index{GroupID: groupID, ObjectIDs: make(map[string]bool)}, nil
		} else {
			return foundInd,
				fmt.Errorf(
					"failed to retrieve existing index from badger DB: %v",
					err)
		}
	}
	err = gob.NewDecoder(bytes.NewBuffer(indBytes)).Decode(&foundInd)
	if err != nil {
		return foundInd,
			fmt.Errorf("failed to decode existing index from bytes: %v", err)
	}
	return foundInd, nil
}

func (store IndexStore) append(groupID string, objectID string) error {
	ind, err := store.get(groupID)
	if err != nil {
		return fmt.Errorf("failed to get index: %v", err)
	}
	ind.ObjectIDs[objectID] = true
	err = store.insert(ind)
	if err != nil {
		return fmt.Errorf("failed to insert modified index: %v", err)
	}
	return nil
}

// remove objectID from Index groupID
func (store IndexStore) remove(groupID string, objectID string) error {
	ind, err := store.get(groupID)
	if err != nil {
		return fmt.Errorf("failed to get index: %v", err)
	}
	ind.ObjectIDs[objectID] = true
	delete(ind.ObjectIDs, objectID)
	err = store.insert(ind)
	if err != nil {
		return fmt.Errorf("failed to insert modified index: %v", err)
	}
	return nil
}

// delete_ Index groupID completely
// TODO: FIXME: what happens if we try to deleteKey a key which doesn't exist
func (store IndexStore) delete_(groupID string) error {
	return deleteKey(store.DB, indexPrefix+groupID)
}
