package mysql

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"github.com/pkg/errors"
	"log"
	"os"
	"sync"
)

type SQLManager struct {
	orm            orm.Ormer
	logger         *log.Logger
	rwLock         *sync.RWMutex
	hasInitialized bool
}

type ISQLManager interface {
	RegisterORM(holder interface{}) error
	RegisterORMs(holder []interface{}) error
	setInitialized(init bool)
	HasInitialized() bool
	withRead(callback func())
	withWrite(callback func())
	Start() error

	Read(query interface{}) (interface{}, error)
	Create(entity interface{}) (int64, error)
	Delete(query interface{}) error
	Upsert(queryEntity interface{}) error
	InsertMany(quantity int, queryEntities interface{}) error

	All(container interface{}) (interface{}, error)
}

func (m *SQLManager) withRead(callback func()) {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	callback()
}

func (m *SQLManager) withWrite(callback func()) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	callback()
}

func (m *SQLManager) setInitialized(init bool) {
	m.withWrite(func() {
		m.hasInitialized = init
	})
}

func (m *SQLManager) HasInitialized() bool {
	var hasInit bool
	m.withRead(func() {
		hasInit = m.hasInitialized
	})
	return hasInit
}

func NewSQLManager(url, userName, password, database string) (*SQLManager, error) {
	dataSource := fmt.Sprintf("%s:%s@tcp(%s)/%s", userName, password, url, database)
	err := orm.RegisterDataBase("default", "mysql", dataSource)
	if err != nil {
		return nil, err
	}
	manager := &SQLManager{
		nil,
		log.New(os.Stdout, "SQLManager", log.Ldate|log.Ltime|log.Lshortfile),
		new(sync.RWMutex),
		false,
	}
	return manager, nil
}

func (m *SQLManager) RegisterORM(holder interface{}) error {
	orm.RegisterModel(holder)
	if m.HasInitialized() {
		return errors.New("No model should be registered after the manager has started")
	}
	return nil
}

func (m *SQLManager) RegisterORMs(holders []interface{}) error {
	if m.HasInitialized() {
		return errors.New("No model should be registered after the manager has started")
	}
	for _, holder := range holders {
		err := m.RegisterORM(holder)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *SQLManager) Start() error {
	err := orm.RunSyncdb("default", false, true)
	if err != nil {
		return err
	}
	o := orm.NewOrm()
	m.withWrite(func() {
		m.orm = o
		m.hasInitialized = true
	})
	return nil
}

func (m *SQLManager) Read(query interface{}) (interface{}, error) {
	if !m.HasInitialized() {
		return nil, errors.New("SQLManager has not started yet.")
	}
	err := m.orm.Read(query)
	if err != nil {
		return nil, err
	}
	return query, err
}

func (m *SQLManager) Create(entity interface{}) (int64, error) {
	if !m.HasInitialized() {
		return -1, errors.New("SQLManager has not started yet.")
	}
	return m.orm.Insert(entity)
}

func (m *SQLManager) Delete(query interface{}) error {
	if !m.HasInitialized() {
		return errors.New("SQLManager has not started yet.")
	}
	_, err := m.orm.Delete(query)
	return err
}

func (m *SQLManager) Upsert(queryEntity interface{}) error {
	if !m.HasInitialized() {
		return errors.New("SQLManager has not started yet.")
	}
	_, err := m.orm.InsertOrUpdate(queryEntity)
	return err
}

func (m *SQLManager) InsertMany(quantity int, queryEntities interface{}) error {
	if !m.HasInitialized() {
		return errors.New("SQLManager has not started yet.")
	}
	_, err := m.orm.InsertMulti(quantity, queryEntities)
	return err
}

// container should be of type *[]X only
func (m *SQLManager) All(holder interface{}, container interface{}) (interface{}, error) {
	_, err := m.orm.QueryTable(holder).All(container)
	return container, err
}

// ref:
// advanced query: https://blog.csdn.net/qq_30505673/article/details/82974458
// multi table query: https://blog.csdn.net/kuangshp128/article/details/109446043
// basic: https://blog.csdn.net/weixin_42488050/article/details/115064312
