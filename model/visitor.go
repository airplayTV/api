package model

import "gorm.io/gorm"

type Visitor struct {
	Id    int    `json:"id"`
	Ip    string `json:"ip"`
	Count uint   `json:"count"`
}

func (Visitor) TableName() string {
	return "visitor"
}

func (x Visitor) CreateOrUpdate(ip string) error {
	var m Visitor
	if err := DB().Table(x.TableName()).Where("ip = ?", ip).Find(&m).Error; err != nil {
		return err
	}
	if m.Id > 0 {
		return DB().Table(x.TableName()).Where("id = ?", m.Id).UpdateColumn("count", gorm.Expr("count + ?", 1)).Error
	} else {
		return DB().Table(x.TableName()).Create(&Visitor{Ip: ip, Count: 1}).Error
	}
}

func (x Visitor) Total() int64 {
	var n int64
	DB().Table(x.TableName()).Count(&n)
	return n
}
