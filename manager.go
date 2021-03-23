package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoManagerError struct {
	msg string
}

func (mme *MongoManagerError) Error() string {
	return mme.msg
}

func newMongoManagerError(msg string) *MongoManagerError {
	return &MongoManagerError{msg}
}

type Collection struct {
	Name        string
	typeHolder  interface{}
	deactivated bool
	ctx         context.Context
}

type ICollection interface {
	Drop() error
	InsertOne(data interface{}) error
	InsertMany(data []interface{}) error
	FindOne() (interface{}, error)
	FindMany() ([]interface{}, error)
	FindOneAndUpdate() error
	FindOneAndReplace() error
	FindOneAndDelete() error
	UpdateOne() error
	UpdateMany() error
	DeleteOne() error
	DeleteMany() error

	Count() (int, error)
}

// ******************CLIENT******************
type Client struct {
	ctx context.Context
	mClient *mongo.Client
	databases []*Database
	// dbs = mClient.ListDatabaseNames(ctx, filter?): ListDatabasesResult, err
	// dbs.Databases
}

type IClient interface {
	CreateDatabase(name string) error
	GetDatabases() []*Database
	Disconnect() error
}

func (c *Client) CreateDatabase(name string) error {
	return newMongoManagerError("Database " + name + " can not be created.")
}

func (c *Client) GetDatabases() []*Database {
	return []*Database{}
}

func (c *Client) Disconnect() error {
	err := c.mClient.Disconnect(c.ctx)
	if err != nil {
		return errors.Wrap(err, "Unable to disconnect mongoDB")
	}
	return nil
}
// ******************CLIENT******************

// ******************DATABASE******************
type Database struct {
	ctx context.Context
	mDB *mongo.Database
	collections []*Collection
	// cls = mDB.ListCollectionSpecifications(ctx. filter?): []CollectionSpecification, err
	// cls.Names

	// need to be able to register collection by name and type
}

type IDatabase interface {
	GetCollections() []*Collection
	GetCollection(name string) *Collection
	RegisterCollection(name string, typeHolder interface{}) bool
	UnregisterCollection(name string) bool
	ListRawCollectionNames() []string
	Drop() error
}
// ******************DATABASE******************

type TransactionAction func()

type Manager struct {
	client        IClient
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
}

type IManager interface {
	Close() error

	Transaction(action TransactionAction) error
	CreateDatabase(name string) error
	GetDatabases() []IDatabase
	GetCollection(collection string) ICollection
	GetCollections() []ICollection
	RegisterCollection(name string, typeHolder interface{}) bool
	UnregisterCollection(name string) bool
	ListRawCollectionNames() []string
}

func NewManager(uri string) (*Manager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.Wrap(err, "Unable to connect Mongo server")
	}
	return &Manager{ &Client{ctx, client, nil}, ctx, cancel}, nil
}

func (m *Manager) Close() error {
	m.ctxCancelFunc()
	err := m.client.Disconnect()
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) Transaction(action TransactionAction) error {
	action()
	return newMongoManagerError("not supported")
}