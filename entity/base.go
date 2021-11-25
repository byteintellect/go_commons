package entity

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

const (
	active   = 0
	inactive = 1
)

type Status int

var statusMapping map[string]int
var statusReverseMapping map[int]string

func init() {
	statusMapping = make(map[string]int)
	statusReverseMapping = make(map[int]string)
	statusMapping["active"] = active
	statusMapping["inactive"] = inactive
	statusReverseMapping[active] = "active"
	statusReverseMapping[inactive] = "inactive"
}

func GetStatusInt(status string) int {
	return statusMapping[status]
}

func GetStatusStr(status int) string {
	return statusReverseMapping[status]
}

type EntityCreator func() Base

type DomainName string

type DomainFactory struct {
	entityMappings map[DomainName]EntityCreator
}

func (d *DomainFactory) RegisterMapping(domainName DomainName, creator EntityCreator) {
	d.entityMappings[domainName] = creator
}

func (d *DomainFactory) GetMapping(domainName DomainName) EntityCreator {
	return d.entityMappings[domainName]
}

func NewDomainFactory() *DomainFactory {
	return &DomainFactory{entityMappings: make(map[DomainName]EntityCreator)}
}

type Base interface {
	GetExternalId() string
	GetTable() DomainName
	GetId() uint64
	GetStatus() Status
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetDeletedAt() time.Time
	ToDto() interface{}
	FromDto(dto interface{}) (Base, error)
	Merge(other interface{})
	FromSqlRow(rows *sql.Rows) (Base, error)
	SetExternalId(externalId string)
	MarshalBinary() ([]byte, error)
	ToJson() (string, error)
	String() string
	UnmarshalBinary(buffer []byte) error
}

type Attribute interface {
	GetKey() string
	GetValue() string
}

type AttributeWithLanguage interface {
	Attribute
	GetLanguage() string
}

type ExternalIdSetter func(externalId string, base Base) Base

type CFTime time.Time

func (et CFTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%v\"", time.Time(et).Format(time.RFC3339))
	return []byte(stamp), nil
}

func (et *CFTime) Value() (driver.Value, error) {
	if &et == nil {
		return nil, nil
	}
	return fmt.Sprintf("\"%v\"", time.Time(*et).Format(time.RFC3339)), nil
}

func (et *CFTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}
	eTime, err := time.Parse(time.RFC3339, string(s))
	if err != nil {
		return err
	}
	x := CFTime(eTime)
	et = &x
	return nil
}

type BaseDomain struct {
	ExternalId string     `json:"external_id" gorm:"type:varchar(100);uniqueIndex"`
	Id         uint64     `json:"id" gorm:"primaryKey;AUTO_INCREMENT"`
	CreatedAt  *time.Time `json:"created_at" type:"date"`
	UpdatedAt  *time.Time `type:"date"`
	DeletedAt  *time.Time `type:"date"`
	Status     int        `type:"int"`
}

func (bd BaseDomain) GetExternalId() string {
	return bd.ExternalId
}

func (bd BaseDomain) GetId() uint64 {
	return bd.Id
}

func (bd BaseDomain) GetStatus() Status {
	return Status(bd.Status)
}

func (bd BaseDomain) GetCreatedAt() time.Time {
	return *bd.CreatedAt
}

func (bd BaseDomain) GetUpdatedAt() time.Time {
	return *bd.UpdatedAt
}

func (bd BaseDomain) GetDeletedAt() time.Time {
	return *bd.DeletedAt
}

func (bd BaseDomain) SetExternalId(externalId string) {
	bd.ExternalId = externalId
}

func (bd BaseDomain) ToJson() (string, error) {
	jBytes, err := json.Marshal(bd)
	if err != nil {
		return "", err
	}
	return string(jBytes), nil
}

func (bd BaseDomain) String() string {
	bdString, _ := bd.ToJson()
	return bdString
}

// BeforeCreate will be called by gorm
func (bd BaseDomain) BeforeCreate(tx *gorm.DB) error {
	if bd.ExternalId == "" {
		tx.Statement.SetColumn("external_id", uuid.New().String())
	}
	return nil
}
//
// BeforeUpdate will be called by gorm
func (bd BaseDomain) BeforeUpdate(tx *gorm.DB) error {
	if bd.ExternalId == "" {
		tx.Statement.SetColumn("external_id", uuid.New().String())
	}
	return nil
}

type Event interface {
	GetEntityId() string
	GetEntityType() string
	GetId() string
	ToBytes() []byte
	FromByte(bytes []byte)
	Entity() interface{}
}
