package types

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-storage-mongo/external/adapter"
)

type Storage interface {
	Create(collection string, document interface{}) (*adapter.SaiStorageResponse, error)
	Read(collection string, criteria map[string]interface{}, options *adapter.Options, fields []string) (*adapter.SaiStorageResponse, error)
	Upsert(collection string, criteria map[string]interface{}, document interface{}) (*adapter.SaiStorageResponse, error)
	Update(collection string, criteria map[string]interface{}, document interface{}) (*adapter.SaiStorageResponse, error)
	Delete(collection string, criteria map[string]interface{}) (*adapter.SaiStorageResponse, error)
}

type storage struct {
	saiStorage adapter.SaiStorage
}

func NewStorage(address, token string) Storage {
	return &storage{
		saiStorage: adapter.SaiStorage{
			Url:   address,
			Token: token,
		},
	}
}

func (s *storage) Create(collection string, document interface{}) (*adapter.SaiStorageResponse, error) {
	storageRequest := adapter.Request{
		Method: "create",
		Data: adapter.CreateRequest{
			Collection: collection,
			Documents:  document.([]interface{}),
		},
	}

	result, err := s.saiStorage.Send(storageRequest)
	if err != nil {
		logger.Logger.Error("Create", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *storage) Read(collection string, criteria map[string]interface{}, options *adapter.Options, fields []string) (*adapter.SaiStorageResponse, error) {
	storageRequest := adapter.Request{
		Method: "read",
		Data: adapter.ReadRequest{
			Collection:    collection,
			Select:        criteria,
			IncludeFields: fields,
			Options:       options,
		},
	}

	result, err := s.saiStorage.Send(storageRequest)
	if err != nil {
		logger.Logger.Error("Read", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *storage) Update(collection string, criteria map[string]interface{}, document interface{}) (*adapter.SaiStorageResponse, error) {
	storageRequest := adapter.Request{
		Method: "update",
		Data: adapter.UpdateRequest{
			Collection: collection,
			Select:     criteria,
			Document:   bson.M{"$set": document},
		},
	}

	result, err := s.saiStorage.Send(storageRequest)
	if err != nil {
		logger.Logger.Error("Update", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *storage) Upsert(collection string, criteria map[string]interface{}, document interface{}) (*adapter.SaiStorageResponse, error) {
	storageRequest := adapter.Request{
		Method: "upsert",
		Data: adapter.UpsertRequest{
			Collection: collection,
			Select:     criteria,
			Document:   document,
		},
	}

	result, err := s.saiStorage.Send(storageRequest)
	if err != nil {
		logger.Logger.Error("Upsert", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *storage) Delete(collection string, criteria map[string]interface{}) (*adapter.SaiStorageResponse, error) {
	storageRequest := adapter.Request{
		Method: "delete",
		Data: adapter.DeleteRequest{
			Collection: collection,
			Select:     criteria,
		},
	}

	result, err := s.saiStorage.Send(storageRequest)
	if err != nil {
		logger.Logger.Error("Delete", zap.Error(err))
		return nil, err
	}

	return result, nil
}
