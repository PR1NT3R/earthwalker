package badgerdb

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/dgraph-io/badger"
	"gitlab.com/glatteis/earthwalker/domain"
)

// TODO: FIXME: lots of repetition in this file.
// TODO: FIXME: indexed objects must be deleted from the leaves inward, because
//				deleting an object also deletes the index with its ID
//				In the meantime, just make sure you don't leave orphaned objects
//				in the db.

// == DB Object Handling ========

// Init opens and returns a badger database connection
// don't forget to close it
func Init(path string) (*badger.DB, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Close closes the given badger database connection
// (provided so you don't have to import badger just to do this)
func Close(db *badger.DB) {
	db.Close()
}

// == Utilities ========

// TODO: make store and get more symmetrical?
func storeStruct(db *badger.DB, key string, t interface{}) error {
	err := db.Update(func(txn *badger.Txn) error {
		var buffer bytes.Buffer
		gob.Register(map[string]interface{}{})
		gob.Register([]interface{}{})
		err := gob.NewEncoder(&buffer).Encode(t)
		if err != nil {
			return err
		}
		return txn.Set([]byte(key), buffer.Bytes())
	})
	return err
}

func getBytes(db *badger.DB, key string) ([]byte, error) {
	var byteSlice []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			byteSlice = append([]byte{}, val...)
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	return byteSlice, err
}

func deleteKey(db *badger.DB, key string) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	return err
}

// == Domain Objects ========

// MapStore badger implementation (see domain)
type MapStore struct {
	DB    *badger.DB
	Index *IndexStore
}

const mapPrefix = "map-"
const mapIndexGroup = "allMaps"

// Insert a domain.Map into store's badger db
func (store MapStore) Insert(m domain.Map) error {
	err := store.Index.append(mapIndexGroup, m.MapID)
	if err != nil {
		return fmt.Errorf("failed to add map to index: %v", err)
	}
	err = storeStruct(store.DB, mapPrefix+m.MapID, m)
	if err != nil {
		return fmt.Errorf("failed to write map to badger DB: %v", err)
	}
	return nil
}

// Get a domain.Map with the given mapID from store's badger db
func (store MapStore) Get(mapID string) (domain.Map, error) {
	mapBytes, err := getBytes(store.DB, mapPrefix+mapID)
	if err != nil || len(mapBytes) == 0 {
		return domain.Map{}, fmt.Errorf("failed to read map from badger DB: %v", err)
	}

	var foundMap domain.Map
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	err = gob.NewDecoder(bytes.NewBuffer(mapBytes)).Decode(&foundMap)
	if err != nil {
		return domain.Map{}, fmt.Errorf("failed to decode map from bytes: %v", err)
	}
	return foundMap, nil
}

// GetAll Map to display on main page
func (store MapStore) GetAll() ([]domain.Map, error) {
	ind, err := store.Index.get(mapIndexGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get Maps index: %v", err)
	}
	results := make([]domain.Map, len(ind.ObjectIDs))
	i := 0
	for mapID := range ind.ObjectIDs {
		mapObj, err := store.Get(mapID)
		if err != nil {
			return nil, fmt.Errorf("failed to get a Map listed in the index: %v", err)
		}
		results[i] = mapObj
		i++
	}
	return results, nil
}

func (store MapStore) Delete(mapID string) error {
	err := deleteKey(store.DB, mapPrefix+mapID)
	if err != nil {
		return fmt.Errorf("failed to delete Map: %v", err)
	}
	// TODO: Index deletion is awk
	err = store.Index.delete_(mapID)
	if err != nil {
		return fmt.Errorf("during deletion of Map '%s', failed to delete "+
			"index of Challenge: %v", mapID, err)
	}
	err = store.Index.remove(mapIndexGroup, mapID)
	if err != nil {
		return fmt.Errorf("failed to remove map ID from index: %v", err)
	}
	return nil
}

// ChallengeStore badger implementation (see domain)
type ChallengeStore struct {
	DB    *badger.DB
	Index *IndexStore
}

const challengePrefix = "challenge-"

// Insert a domain.Challenge into store's badger db
func (store ChallengeStore) Insert(c domain.Challenge) error {
	err := store.Index.append(c.MapID, c.ChallengeID)
	if err != nil {
		return fmt.Errorf("failed to add challenge to index: %v", err)
	}
	err = storeStruct(store.DB, challengePrefix+c.ChallengeID, c)
	if err != nil {
		return fmt.Errorf("failed to write challenge to badger DB: %v", err)
	}
	return nil
}

// Get a domain.Challenge with the given challengeID from store's badger db
func (store ChallengeStore) Get(challengeID string) (domain.Challenge, error) {
	var challengeBytes []byte
	challengeBytes, err := getBytes(store.DB, challengePrefix+challengeID)
	if err != nil {
		return domain.Challenge{}, fmt.Errorf("failed to read challenge from badger DB: %v", err)
	}

	var foundChallenge domain.Challenge
	err = gob.NewDecoder(bytes.NewBuffer(challengeBytes)).Decode(&foundChallenge)
	if err != nil {
		return domain.Challenge{}, fmt.Errorf("failed to decode challenge from bytes: %v", err)
	}
	return foundChallenge, nil
}

// GetList of Challenge for a given mapID
func (store ChallengeStore) GetList(mapID string) ([]string, error) {
	ind, err := store.Index.get(mapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results index: %v", err)
	}
	results := make([]string, len(ind.ObjectIDs))
	i := 0
	for challengeID := range ind.ObjectIDs {
		results[i] = challengeID
		i++
	}
	return results, nil
}

// GetAll Challenge for a given mapID
func (store ChallengeStore) GetAll(mapID string) ([]domain.Challenge, error) {
	ind, err := store.Index.get(mapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results index: %v", err)
	}
	results := make([]domain.Challenge, len(ind.ObjectIDs))
	i := 0
	for challengeID := range ind.ObjectIDs {
		challenge, err := store.Get(challengeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get a challenge result listed in the index: %v", err)
		}
		results[i] = challenge
		i++
	}
	return results, nil
}

func (store ChallengeStore) Delete(challengeID string) error {
	err := deleteKey(store.DB, challengePrefix+challengeID)
	if err != nil {
		return fmt.Errorf("failed to delete challenge: %v", err)
	}
	err = store.Index.delete_(challengeID)
	if err != nil {
		return fmt.Errorf("during deletion of Challenge '%s', failed to "+
			"delete Index of ChallengeResults: %v", challengeID, err)
	}
	return nil
}

// DeleteAll Challenge for a given mapID
func (store ChallengeStore) DeleteAll(mapID string) error {
	ind, err := store.Index.get(mapID)
	if err != nil {
		return fmt.Errorf("failed to get challenges index: %v", err)
	}
	for challengeID := range ind.ObjectIDs {
		err := store.Delete(challengeID)
		if err != nil {
			return fmt.Errorf("failed to delete a challenge listed in the index: %v", err)
		}
	}
	return nil
}

// note: no ChallengePlaceStore implementation,
// because we just store the entire Challenge as a blob
// one will probably be necessary for relational databases
// (which don't take well to arbitrary length fields)

// ChallengeResultStore badger implementation (see domain)
type ChallengeResultStore struct {
	DB    *badger.DB
	Index *IndexStore
}

const challengeResultPrefix = "result-"

// Insert a domain.ChallengeResult into store's badger db
func (store ChallengeResultStore) Insert(r domain.ChallengeResult) error {
	err := store.Index.append(r.ChallengeID, r.ChallengeResultID)
	if err != nil {
		return fmt.Errorf("failed to add challenge result to index: %v", err)
	}
	err = storeStruct(store.DB, challengeResultPrefix+r.ChallengeResultID, r)
	if err != nil {
		return fmt.Errorf("failed to write challenge result to badger DB: %v", err)
	}
	return nil
}

// Get a domain.ChallengeResult with the given challengeResultID from store's badger db
func (store ChallengeResultStore) Get(challengeResultID string) (domain.ChallengeResult, error) {
	resultBytes, err := getBytes(store.DB, challengeResultPrefix+challengeResultID)
	if err != nil {
		return domain.ChallengeResult{}, fmt.Errorf("failed to read result from badger DB: %v", err)
	}

	var foundResult domain.ChallengeResult
	err = gob.NewDecoder(bytes.NewBuffer(resultBytes)).Decode(&foundResult)
	if err != nil {
		return domain.ChallengeResult{}, fmt.Errorf("failed to decode result from bytes: %v", err)
	}
	return foundResult, nil
}

// GetAll ChallengeResult for a given challengeID
func (store ChallengeResultStore) GetAll(challengeID string) ([]domain.ChallengeResult, error) {
	ind, err := store.Index.get(challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results index: %v", err)
	}
	results := make([]domain.ChallengeResult, len(ind.ObjectIDs))
	i := 0
	for challengeResultID := range ind.ObjectIDs {
		challengeResult, err := store.Get(challengeResultID)
		if err != nil {
			return nil, fmt.Errorf("failed to get a challenge result listed in the index: %v", err)
		}
		results[i] = challengeResult
		i++
	}
	return results, nil
}

func (store ChallengeResultStore) Delete(challengeResultID string) error {
	err := deleteKey(store.DB, challengeResultPrefix+challengeResultID)
	if err != nil {
		return fmt.Errorf("failed to delete challenge result: %v", err)
	}
	return nil
}

// DeleteAll ChallengeResult for a given challengeID
func (store ChallengeResultStore) DeleteAll(challengeID string) error {
	ind, err := store.Index.get(challengeID)
	if err != nil {
		return fmt.Errorf("failed to get results index: %v", err)
	}
	for challengeResultID := range ind.ObjectIDs {
		err := store.Delete(challengeResultID)
		if err != nil {
			return fmt.Errorf("failed to delete a challenge result listed in the index: %v", err)
		}
	}
	return nil
}
